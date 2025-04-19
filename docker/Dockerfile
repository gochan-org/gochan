FROM golang:1.24-alpine

WORKDIR /opt/gochan

COPY . .
COPY docker/gochan-docker.json /etc/gochan/gochan.json

ARG GOCHAN_DBTYPE
ARG GOCHAN_DBHOST

ENV DBTYPE=${GOCHAN_DBTYPE}
ENV DBHOST=${GOCHAN_DBHOST}

COPY docker/build-image.sh .

RUN apk update && apk add --no-cache python3 nodejs npm
RUN npm --prefix frontend/ install
RUN ["python3", "build.py", "js"]
RUN ["./build-image.sh"]

CMD ["/opt/gochan/docker/startup.sh"]
