#!/bin/sh

set -euo pipefail

apk add ffmpeg python3 git gcc openssl exiftool musl-dev

CFG_DBTYPE=$DBTYPE

if [ "$DBTYPE" = "mariadb" ]; then
	apk add mariadb-client
	CFG_DBTYPE=mysql
elif [ "$DBTYPE" = "mysql" ]; then
	apk add mysql-client
elif [ "$DBTYPE" = "postgres" ]; then
	apk add postgresql16-client
elif [ "$DBTYPE" = "sqlite3" ]; then
	apk add sqlite sqlite-dev sqlite-libs
elif [ -z "$DBTYPE" ]; then
	echo "DBTYPE is not set"
	exit 1
else
	echo "Unrecognized DBTYPE '$DBTYPE'"
	exit 1
fi

sed -i /etc/gochan/gochan.json \
	-e 's/"DBhost": .*/"DBhost": "'$DBHOST'",/' \
	-e 's/"DBtype": .*/"DBtype": "'$CFG_DBTYPE'",/'

mkdir -p /var/www/gochan
echo "Building gochan executable"
go mod tidy
./build.py && ./build.py install --symlinks
