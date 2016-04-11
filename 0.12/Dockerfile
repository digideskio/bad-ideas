FROM buildpack-deps:jessie-curl

RUN gpg \
    --keyserver hkp://pgp.mit.edu \
    --recv-keys 05CE15085FC09D18E99EFB22684A14CF2582E0C5

ENV CHRONOGRAF_VERSION 0.12.0
RUN wget -q https://s3.amazonaws.com/get.influxdb.org/chronograf/chronograf_${CHRONOGRAF_VERSION}_amd64.deb.asc && \
    wget -q https://s3.amazonaws.com/get.influxdb.org/chronograf/chronograf_${CHRONOGRAF_VERSION}_amd64.deb && \
    gpg --batch --verify chronograf_${CHRONOGRAF_VERSION}_amd64.deb.asc chronograf_${CHRONOGRAF_VERSION}_amd64.deb && \
    dpkg -i chronograf_${CHRONOGRAF_VERSION}_amd64.deb && \
    rm -f chronograf_${CHRONOGRAF_VERSION}_amd64.deb*

EXPOSE 10000

ENV PATH /opt/chronograf/:$PATH

ENV CHRONOGRAF_BIND 0.0.0.0:10000
ENV CHRONOGRAF_LOCAL_DATABASE /var/lib/chronograf/chronograf.db
VOLUME /var/lib/chronograf

COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
CMD ["chronograf"]
