# Chronograf

Chronograf is a simple to install graphing and visualization application that you deploy behind your firewall to perform ad hoc exploration of your InfluxDB data, It includes support for templates and a library of intelligent, pre-configured dashboards for common data sets.

### Exposed Ports
- 10000 (default port)

- 80 & 443 (Additional Ports)

### Using this Image
##### Run using the default configuration
By default, Chronograf runs on localhost port 10000
```
docker run --net=host chronograf
```
### Using a config file from host
```
docker run -v ~/path/on/host/:/root/  -v ~/path/on/host/config.toml:/opt/chronograf/config.toml:ro  --net=host chronograf
```

Chronograf 0.10 has following config variables


| FLAG  | ENV VAR | DEFAULT VALUE |
| ------------- | ------------- |------------- |
| BIND  | CHRONOGRAF_BIND  | http://127.0.0.1:10000|
| LocalDatabase  | CHRONOGRAF_LOCAL_DATABASE  |/opt/chronograf/chronograf.db|
| QueryResponseBytesLimit  | CHRONOGRAF_QUERY_RESPONSE_BYTES_LIMIT  | 2500000|

All of these can be provided in the config file or overrided using the environment variables 

### Overriding the configs using env vars

```
 docker run --env CHRONOGRAF_BIND="0.0.0.0:10000" -p 10000:10000 chronograf 
```

### Using a db file from host
```
docker run -v ~/some/path/on/host/:/root/ --env CHRONOGRAF_LOCAL_DATABASE=/root/chronograf.db  --net=host chronograf
```

### Follow [this](https://docs.influxdata.com/chronograf/v0.10/introduction/getting_started/) to create the visualizations
