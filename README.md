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

By default, Chronograf runs on localhost port `10000`. The simplest
invocation of the container is:

    docker run --net=host chronograf

Chronograf exposes a shared volume under `/var/lib/chronograf`, which it uses to store configuration and database data. You can mount a host directory to `/var/lib/chronograf` for shared access.

    docker run --net=host \
           -v /path/on/host:/var/lib/chronograf \
           chronograf

##### Using a custom config file

Assuming you are using `/path/on/host` as your shared volume, place a configuration file `config.toml` in that directory and then:

    docker run --net=host \
           -v /path/on/host:/var/lib/chronograf \
           chronograf -config /var/lib/config.toml

#### Config variables

Chronograf 0.10 has following config variables

| FLAG                    | ENV VAR                               | DEFAULT VALUE                     |
|-------------------------|---------------------------------------|-----------------------------------|
| BIND                    | CHRONOGRAF_BIND                       | 0.0.0.0:10000                     |
| LocalDatabase           | CHRONOGRAF_LOCAL_DATABASE             | /var/lib/chronograf/chronograf.db |
| QueryResponseBytesLimit | CHRONOGRAF_QUERY_RESPONSE_BYTES_LIMIT | 2500000                           |

All of these can be provided in the config file or overrided using the environment variables

##### Using environment variables

    docker run --env CHRONOGRAF_BIND="0.0.0.0:10000" -p 10000:10000 chronograf

##### Using a custom database file

    docker run --net=host \
           -v /path/on/host/:/var/lib/other_chronograf.db \
           --env CHRONOGRAF_LOCAL_DATABASE=/var/lib/other_chronograf db \
           chronograf

#### Official Docs

See the [official docs](https://docs.influxdata.com/chronograf/latest/introduction/getting_started/) for information on creating
visualizations.

## Supported Docker versions

This image is officially supported on Docker version 1.10.1
