FROM golang:1.20-alpine3.18

WORKDIR /opt/gochan

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
	mkdir -p /var/www/gochan && \
	echo "Building gochan executable" && \
	./build.py && ./build.py install --symlinks

COPY docker/gochan-docker.json /etc/gochan/gochan.json
ENV DBTYPE=mysql
ENV DATABASE_HOST=gochan-mariadb
ENV DATABASE_PORT=3306

CMD ["/opt/gochan/docker/startup.sh"]
