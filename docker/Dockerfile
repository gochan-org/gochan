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
	ln -sn /opt/gochan/html/* /var/www/gochan/

COPY docker/gochan-docker.json /etc/gochan/gochan.json
ENV DBTYPE=mysql
ENV DATABASE_HOST=gochan-mariadb
ENV DATABASE_PORT=3306

CMD ["/opt/gochan/docker/startup.sh"]
