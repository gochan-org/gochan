FROM golang:1.24-alpine3.20

WORKDIR /opt/gochan

COPY . .
COPY examples/configs/gochan.example.json /opt/gochan/gochan-init.json

ARG GOCHAN_PORT="80"
ARG GOCHAN_SITE_HOST="127.0.0.1"
ARG GOCHAN_DB_TYPE
ARG GOCHAN_DB_HOST

ENV GOCHAN_PORT=${GOCHAN_PORT}
ENV SITE_HOST=${GOCHAN_SITE_HOST}
ENV DB_TYPE=${GOCHAN_DB_TYPE}
ENV DB_HOST=${GOCHAN_DB_HOST}

COPY docker/build-image.sh .

RUN ["./build-image.sh"]

CMD ["/opt/gochan/docker/startup.sh"]
