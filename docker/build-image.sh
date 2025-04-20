#!/bin/sh

set -euo pipefail

apk add ffmpeg python3 git gcc openssl exiftool musl-dev nodejs npm
CFG_DB_TYPE=$DB_TYPE

if [ "$DB_TYPE" = "mariadb" ]; then
	apk add mariadb-client
	CFG_DB_TYPE=mysql
elif [ "$DB_TYPE" = "mysql" ]; then
	apk add mysql-client
elif [ "$DB_TYPE" = "postgres" ]; then
	apk add postgresql16-client
elif [ "$DB_TYPE" = "sqlite3" ]; then
	apk add sqlite sqlite-dev sqlite-libs
elif [ -z "$DB_TYPE" ]; then
	echo "DB_TYPE is not set"
	exit 1
else
	echo "Unrecognized DB_TYPE '$DB_TYPE'"
	exit 1
fi

sed -i /opt/gochan/gochan-init.json \
	-e 's/"ListenAddress": .*/"ListenAddress": "gochan-server",/' \
	-e 's/"UseFastCGI": .*/"UseFastCGI": false,/' \
	-e 's/"DocumentRoot": .*/"DocumentRoot": "\/var\/www\/gochan",/' \
	-e 's/"TemplateDir": .*/"TemplateDir": "\/opt\/gochan\/templates",/' \
	-e 's/"LogDir": .*/"LogDir": "\/var\/log\/gochan",/' \
	-e 's/"SiteHost": .*/"SiteHost": "'$SITE_HOST'",/' \
	-e 's/"Port": .*/"Port": '$GOCHAN_PORT',/' \
	-e 's/"DBhost": .*/"DBhost": "'$DB_HOST'",/' \
	-e 's/"DBtype": .*/"DBtype": "'$CFG_DB_TYPE'",/' \
	-e 's/"DBpassword": .*/"DBpassword": "gochan",/'

rm -f /opt/gochan/gochan.json
mkdir -p /etc/gochan
mkdir -p /var/log/gochan
mkdir -p /var/www/gochan

npm --prefix frontend/ install
npm --prefix frontend/ run build-ts
echo "Building gochan executable"
go mod tidy
./build.py && ./build.py install --symlinks