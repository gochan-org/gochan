FROM golang:1.20-alpine3.18

COPY docker/build-image.sh .

WORKDIR /opt/gochan

# Get all
COPY . .

RUN apk add \
	mariadb-client \
	ffmpeg \
	python3 \
	git \
	gcc \
	musl-dev \
	openssl \
	exiftool && \
	mkdir -p /var/www/ && \
	ln -sn /opt/gochan/html/* /var/www/


COPY docker/gochan-docker.json /etc/gochan/gochan.json

RUN ["./build.py"]

CMD ["/opt/gochan/docker/startup.sh"]
