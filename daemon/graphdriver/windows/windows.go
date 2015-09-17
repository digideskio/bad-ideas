//+build windows

package windows

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/autogen/dockerversion"
	"github.com/docker/docker/daemon/graphdriver"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/chrootarchive"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/random"
	"github.com/microsoft/hcsshim"
)

func init() {
	graphdriver.Register("windowsfilter", InitFilter)
	graphdriver.Register("windowsdiff", InitDiff)
}

const (
	diffDriver = iota
	filterDriver
)

type Driver struct {
	info       hcsshim.DriverInfo
	sync.Mutex // Protects concurrent modification to active
	active     map[string]int
}

// New returns a new Windows storage filter driver.
func InitFilter(home string, options []string) (graphdriver.Driver, error) {
	logrus.Debugf("WindowsGraphDriver InitFilter at %s", home)
	d := &Driver{
		info: hcsshim.DriverInfo{
			HomeDir: home,
			Flavour: filterDriver,
		},
		active: make(map[string]int),
	}
	return d, nil
}

// New returns a new Windows differencing disk driver.
func InitDiff(home string, options []string) (graphdriver.Driver, error) {
	logrus.Debugf("WindowsGraphDriver InitDiff at %s", home)
	d := &Driver{
		info: hcsshim.DriverInfo{
			HomeDir: home,
			Flavour: diffDriver,
		},
		active: make(map[string]int),
	}
	return d, nil
}

func (d *Driver) String() string {
	switch d.info.Flavour {
	case diffDriver:
		return "windowsdiff"
	case filterDriver:
		return "windowsfilter"
	default:
		return "Unknown driver flavour"
	}
}

func (d *Driver) Status() [][2]string {
	return [][2]string{
		{"Windows", ""},
	}
}

// Exists returns true if the given id is registered with
// this driver
func (d *Driver) Exists(id string) bool {
	rId, err := d.resolveId(id)
	if err != nil {
		return false
	}
	result, err := hcsshim.LayerExists(d.info, rId)
	if err != nil {
		return false
	}
	return result
}

func (d *Driver) Create(id, parent string) error {
	rPId, err := d.resolveId(parent)
	if err != nil {
		return err
	}

	parentChain, err := d.getLayerChain(rPId)
	if err != nil {
		return err
	}

	var layerChain []string

	parentIsInit := strings.HasSuffix(rPId, "-init")

	if !parentIsInit && rPId != "" {
		parentPath, err := hcsshim.GetLayerMountPath(d.info, rPId)
		if err != nil {
			return err
		}
		layerChain = []string{parentPath}
	}

	layerChain = append(layerChain, parentChain...)

	if parentIsInit {
		if len(layerChain) == 0 {
			return fmt.Errorf("Cannot create a read/write layer without a parent layer.")
		}
		if err := hcsshim.CreateSandboxLayer(d.info, id, layerChain[0], layerChain); err != nil {
			return err
		}
	} else {
		if err := hcsshim.CreateLayer(d.info, id, rPId); err != nil {
			return err
		}
	}

	if _, err := os.Lstat(d.dir(parent)); err != nil {
		if err2 := hcsshim.DestroyLayer(d.info, id); err2 != nil {
			logrus.Warnf("Failed to DestroyLayer %s: %s", id, err2)
		}
		return fmt.Errorf("Cannot create layer with missing parent %s: %s", parent, err)
	}

	if err := d.setLayerChain(id, layerChain); err != nil {
		if err2 := hcsshim.DestroyLayer(d.info, id); err2 != nil {
			logrus.Warnf("Failed to DestroyLayer %s: %s", id, err2)
		}
		return err
	}

	return nil
}

func (d *Driver) dir(id string) string {
	return filepath.Join(d.info.HomeDir, filepath.Base(id))
}

// Remove unmounts and removes the dir information
func (d *Driver) Remove(id string) error {
	rId, err := d.resolveId(id)
	if err != nil {
		return err
	}

	return hcsshim.DestroyLayer(d.info, rId)
}

// Get returns the rootfs path for the id. This will mount the dir at it's given path
func (d *Driver) Get(id, mountLabel string) (string, error) {
	logrus.Debugf("WindowsGraphDriver Get() id %s mountLabel %s", id, mountLabel)
	var dir string

	d.Lock()
	defer d.Unlock()

	rId, err := d.resolveId(id)
	if err != nil {
		return "", err
	}

	// Getting the layer paths must be done outside of the lock.
	layerChain, err := d.getLayerChain(rId)
	if err != nil {
		return "", err
	}

	if d.active[rId] == 0 {
		if err := hcsshim.ActivateLayer(d.info, rId); err != nil {
			return "", err
		}
		if err := hcsshim.PrepareLayer(d.info, rId, layerChain); err != nil {
			if err2 := hcsshim.DeactivateLayer(d.info, rId); err2 != nil {
				logrus.Warnf("Failed to Deactivate %s: %s", id, err)
			}
			return "", err
		}
	}

	mountPath, err := hcsshim.GetLayerMountPath(d.info, rId)
	if err != nil {
		if err2 := hcsshim.DeactivateLayer(d.info, rId); err2 != nil {
			logrus.Warnf("Failed to Deactivate %s: %s", id, err)
		}
		return "", err
	}

	d.active[rId]++

	// If the layer has a mount path, use that. Otherwise, use the
	// folder path.
	if mountPath != "" {
		dir = mountPath
	} else {
		dir = d.dir(id)
	}

	return dir, nil
}

func (d *Driver) Put(id string) error {
	logrus.Debugf("WindowsGraphDriver Put() id %s", id)

	rId, err := d.resolveId(id)
	if err != nil {
		return err
	}

	d.Lock()
	defer d.Unlock()

	if d.active[rId] > 1 {
		d.active[rId]--
	} else if d.active[rId] == 1 {
		if err := hcsshim.UnprepareLayer(d.info, rId); err != nil {
			return err
		}
		if err := hcsshim.DeactivateLayer(d.info, rId); err != nil {
			return err
		}
		delete(d.active, rId)
	}

	return nil
}

func (d *Driver) Cleanup() error {
	return nil
}

// Diff produces an archive of the changes between the specified
// layer and its parent layer which may be "".
func (d *Driver) Diff(id, parent string) (arch archive.Archive, err error) {
	rId, err := d.resolveId(id)
	if err != nil {
		return
	}

	// Getting the layer paths must be done outside of the lock.
	layerChain, err := d.getLayerChain(rId)
	if err != nil {
		return
	}

	d.Lock()

	// To support export, a layer must be activated but not prepared.
	if d.info.Flavour == filterDriver {
		if d.active[rId] == 0 {
			if err = hcsshim.ActivateLayer(d.info, rId); err != nil {
				d.Unlock()
				return
			}
			defer func() {
				if err := hcsshim.DeactivateLayer(d.info, rId); err != nil {
					logrus.Warnf("Failed to Deactivate %s: %s", rId, err)
				}
			}()
		} else {
			if err = hcsshim.UnprepareLayer(d.info, rId); err != nil {
				d.Unlock()
				return
			}
			defer func() {
				if err := hcsshim.PrepareLayer(d.info, rId, layerChain); err != nil {
					logrus.Warnf("Failed to re-PrepareLayer %s: %s", rId, err)
				}
			}()
		}
	}

	d.Unlock()

	return d.exportLayer(rId, layerChain)
}

// Changes produces a list of changes between the specified layer
// and its parent layer. If parent is "", then all changes will be ADD changes.
func (d *Driver) Changes(id, parent string) ([]archive.Change, error) {
	return nil, fmt.Errorf("The Windows graphdriver does not support Changes()")
}

// ApplyDiff extracts the changeset from the given diff into the
// layer with the specified id and parent, returning the size of the
// new layer in bytes.
func (d *Driver) ApplyDiff(id, parent string, diff archive.Reader) (size int64, err error) {
	rPId, err := d.resolveId(parent)
	if err != nil {
		return
	}

	if d.info.Flavour == diffDriver {
		start := time.Now().UTC()
		logrus.Debugf("WindowsGraphDriver ApplyDiff: Start untar layer")
		destination := d.dir(id)
		destination = filepath.Dir(destination)
		if size, err = chrootarchive.ApplyUncompressedLayer(destination, diff); err != nil {
			return
		}
		logrus.Debugf("WindowsGraphDriver ApplyDiff: Untar time: %vs", time.Now().UTC().Sub(start).Seconds())

		return
	}

	parentChain, err := d.getLayerChain(rPId)
	if err != nil {
		return
	}
	parentPath, err := hcsshim.GetLayerMountPath(d.info, rPId)
	if err != nil {
		return
	}
	layerChain := []string{parentPath}
	layerChain = append(layerChain, parentChain...)

	if size, err = d.importLayer(id, diff, layerChain); err != nil {
		return
	}

	if err = d.setLayerChain(id, layerChain); err != nil {
		return
	}

	return
}

// DiffSize calculates the changes between the specified layer
// and its parent and returns the size in bytes of the changes
// relative to its base filesystem directory.
func (d *Driver) DiffSize(id, parent string) (size int64, err error) {
	rPId, err := d.resolveId(parent)
	if err != nil {
		return
	}

	changes, err := d.Changes(id, rPId)
	if err != nil {
		return
	}

	layerFs, err := d.Get(id, "")
	if err != nil {
		return
	}
	defer d.Put(id)

	return archive.ChangesSize(layerFs, changes), nil
}

func (d *Driver) RestoreCustomImages(tagger graphdriver.Tagger, recorder graphdriver.Recorder) (imageIDs []string, err error) {
	strData, err := hcsshim.GetSharedBaseImages()
	if err != nil {
		return nil, fmt.Errorf("Failed to restore base images: %s", err)
	}

	type customImageInfo struct {
		Name        string
		Version     string
		Path        string
		Size        int64
		CreatedTime time.Time
	}
	type customImageInfoList struct {
		Images []customImageInfo
	}

	var infoData customImageInfoList

	if err = json.Unmarshal([]byte(strData), &infoData); err != nil {
		err = fmt.Errorf("JSON unmarshal returned error=%s", err)
		logrus.Error(err)
		return nil, err
	}

	for _, imageData := range infoData.Images {
		_, folderName := filepath.Split(imageData.Path)

		// Use crypto hash of the foldername to generate a docker style id.
		h := sha512.Sum384([]byte(folderName))
		id := fmt.Sprintf("%x", h[:32])

		if !recorder.Exists(id) {
			// Register the image.
			img := &image.Image{
				ID:            id,
				Created:       imageData.CreatedTime,
				DockerVersion: dockerversion.VERSION,
				Architecture:  runtime.GOARCH,
				OS:            runtime.GOOS,
				Size:          imageData.Size,
			}

			if err := recorder.Register(img, nil); err != nil {
				return nil, err
			}

			// Create tags for the new image.
			if err := tagger.Tag(strings.ToLower(imageData.Name), imageData.Version, img.ID, true); err != nil {
				return nil, err
			}

			// Create the alternate ID file.
			if err := d.setId(img.ID, folderName); err != nil {
				return nil, err
			}

			imageIDs = append(imageIDs, img.ID)
		}
	}

	return imageIDs, nil
}

func (d *Driver) GetMetadata(id string) (map[string]string, error) {
	m := make(map[string]string)
	m["dir"] = d.dir(id)
	return m, nil
}

func (d *Driver) exportLayer(id string, parentLayerPaths []string) (arch archive.Archive, err error) {
	layerFolder := d.dir(id)

	tempFolder := layerFolder + "-" + strconv.FormatUint(uint64(random.Rand.Uint32()), 10)
	if err = os.MkdirAll(tempFolder, 0755); err != nil {
		logrus.Errorf("Could not create %s %s", tempFolder, err)
		return
	}
	defer func() {
		if err != nil {
			_, folderName := filepath.Split(tempFolder)
			if err2 := hcsshim.DestroyLayer(d.info, folderName); err2 != nil {
				logrus.Warnf("Couldn't clean-up tempFolder: %s %s", tempFolder, err2)
			}
		}
	}()

	if err = hcsshim.ExportLayer(d.info, id, tempFolder, parentLayerPaths); err != nil {
		return
	}

	archive, err := archive.Tar(tempFolder, archive.Uncompressed)
	if err != nil {
		return
	}
	return ioutils.NewReadCloserWrapper(archive, func() error {
		err := archive.Close()
		d.Put(id)
		_, folderName := filepath.Split(tempFolder)
		if err2 := hcsshim.DestroyLayer(d.info, folderName); err2 != nil {
			logrus.Warnf("Couldn't clean-up tempFolder: %s %s", tempFolder, err2)
		}
		return err
	}), nil

}

func (d *Driver) importLayer(id string, layerData archive.Reader, parentLayerPaths []string) (size int64, err error) {
	layerFolder := d.dir(id)

	tempFolder := layerFolder + "-" + strconv.FormatUint(uint64(random.Rand.Uint32()), 10)
	if err = os.MkdirAll(tempFolder, 0755); err != nil {
		logrus.Errorf("Could not create %s %s", tempFolder, err)
		return
	}
	defer func() {
		_, folderName := filepath.Split(tempFolder)
		if err2 := hcsshim.DestroyLayer(d.info, folderName); err2 != nil {
			logrus.Warnf("Couldn't clean-up tempFolder: %s %s", tempFolder, err2)
		}
	}()

	start := time.Now().UTC()
	logrus.Debugf("Start untar layer")
	if size, err = chrootarchive.ApplyLayer(tempFolder, layerData); err != nil {
		return
	}
	logrus.Debugf("Untar time: %vs", time.Now().UTC().Sub(start).Seconds())

	if err = hcsshim.ImportLayer(d.info, id, tempFolder, parentLayerPaths); err != nil {
		return
	}

	return
}

func (d *Driver) resolveId(id string) (string, error) {
	content, err := ioutil.ReadFile(filepath.Join(d.dir(id), "layerId"))
	if os.IsNotExist(err) {
		return id, nil
	} else if err != nil {
		return "", err
	}
	return string(content), nil
}

func (d *Driver) setId(id, altId string) error {
	err := ioutil.WriteFile(filepath.Join(d.dir(id), "layerId"), []byte(altId), 0600)
	if err != nil {
		return err
	}
	return nil
}

func (d *Driver) getLayerChain(id string) ([]string, error) {
	jPath := filepath.Join(d.dir(id), "layerchain.json")
	content, err := ioutil.ReadFile(jPath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("Unable to read layerchain file - %s", err)
	}

	var layerChain []string
	err = json.Unmarshal(content, &layerChain)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshall layerchain json - %s", err)
	}

	return layerChain, nil
}

func (d *Driver) setLayerChain(id string, chain []string) error {
	content, err := json.Marshal(&chain)
	if err != nil {
		return fmt.Errorf("Failed to marshall layerchain json - %s", err)
	}

	jPath := filepath.Join(d.dir(id), "layerchain.json")
	err = ioutil.WriteFile(jPath, content, 0600)
	if err != nil {
		return fmt.Errorf("Unable to write layerchain file - %s", err)
	}

	return nil
}