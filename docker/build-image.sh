#!/bin/sh

set -euo pipefail

apk add \
	mariadb-client		\
	nginx 				\
	ffmpeg				\
	python3				\
	git					\
	gcc					\
	musl-dev			\
	openssl				\
	exiftool

mkdir -p /root/bin

ln -s /usr/lib/go-1.20/bin/* /root/bin/
export PATH=$PATH:/root/bin
echo "export PATH=$PATH:/root/bin" >> /root/.bashrc
rm -f /etc/nginx/sites-enabled/* /etc/nginx/sites-available/*
mkdir -p /var/lib/nginx
mkdir -p /var/lib/nginx/tmp
mkdir -p /run/nginx/

rm -f /etc/nginx/http.d/default.conf


# The openssl command will generate self-signed certificate since some browsers like
# Firefox and Chrome automatically do HTTPS requests. this will likely show a warning in
# the browser, which you can ignore
mkdir -p /etc/ssl/private
openssl req -x509 -nodes -days 7305 -newkey rsa:2048 \
	-keyout /etc/ssl/private/nginx-selfsigned.key \
	-out /etc/ssl/certs/nginx-selfsigned.crt \
	-subj "/CN=127.0.0.1"
