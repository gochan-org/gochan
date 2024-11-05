FROM golang:1.20-alpine3.18

WORKDIR /opt/gochan

COPY . .
COPY docker/gochan-docker.json /etc/gochan/gochan.json

ARG GOCHAN_DBTYPE
ARG GOCHAN_DBHOST

ENV DBTYPE=${GOCHAN_DBTYPE}
ENV DBHOST=${GOCHAN_DBHOST}

COPY docker/build-image.sh .

RUN ["./build-image.sh"]

CMD ["/opt/gochan/docker/startup.sh"]
