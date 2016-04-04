# Chronograf

Chronograf is a simple to install graphing and visualization
application that you deploy behind your firewall to perform ad-hoc
exploration of your InfluxDB data. It includes support for templates
and a library of intelligent, pre-configured dashboards for common
data sets.

## Using this image

##### Exposed Ports

 - 10000 (default port)

##### Using the default configuration

By default, Chronograf runs on localhost port `10000`. Chronograf
exposes a shared volume under `/var/lib/chronograf`, which it uses to
store database data. You can mount a host directory to
`/var/lib/chronograf` for host access to persisted container data. A
typical invocation of the Chronograf container might be:

    docker run --net=host \
           -v /path/on/host:/var/lib/chronograf \
           chronograf

You can also have Docker control the volume mountpoint by using a named volume.

    docker run -p 8083:8083 -p 8086:8086 \
           -v chronograf:/var/lib/chronograf \
           chronograf

##### Using a custom config file

Assuming a custom configuration file on your host at
`/path/to/config.toml` you can mount a shared volume as follows:

    docker run --net=host \
           -v /path/to:/opt/chronograf:ro \
           chronograf -config /opt/chronograf/config.toml

#### Config variables

Chronograf 0.10 has following config variables

| FLAG                    | ENV VAR                               | DEFAULT VALUE                     |
|-------------------------|---------------------------------------|-----------------------------------|
| LocalDatabase           | CHRONOGRAF_LOCAL_DATABASE             | /var/lib/chronograf/chronograf.db |
| QueryResponseBytesLimit | CHRONOGRAF_QUERY_RESPONSE_BYTES_LIMIT | 2500000                           |

All of these can be provided in the config file or overrided using the
environment variables

##### Binding to a different port

To bind to a different port you can use Docker's `-p` flag. For example, to run Chronograf on port `9999`:

    docker run -p 9999:10000 chronograf

##### Using a different database file

    docker run --net=host \
           -v /path/on/host/:/var/lib/other_chronograf.db \
           --env CHRONOGRAF_LOCAL_DATABASE=/var/lib/other_chronograf db \
           chronograf

#### Official Docs

See the [official docs](https://docs.influxdata.com/chronograf/latest/introduction/getting_started/) for information on creating
visualizations.

## Supported Docker versions

This image is officially supported on Docker version 1.10.1
