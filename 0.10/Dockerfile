FROM ubuntu:14.04

EXPOSE 10000

EXPOSE 80
EXPOSE 443

RUN apt-get update && \
    apt-get install -y wget

RUN wget https://s3.amazonaws.com/get.influxdb.org/chronograf/chronograf_0.10.0_amd64.deb && \
    dpkg -i chronograf_0.10.0_amd64.deb && \
    rm chronograf_0.10.0_amd64.deb

ENV PATH /opt/chronograf/:$PATH

ENTRYPOINT ["chronograf", "-config", "/opt/chronograf/config.toml"]
