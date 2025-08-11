#!/usr/bin/env bash

# Shell script that downloads a previous gochan release for testing gochan database updating
# This should only be used in a development environment

set -eo pipefail

TESTING_VERSION=$1

if [ -z "$TESTING_VERSION" ]; then
	TESTING_VERSION="v3.10.2"
elif [ "$TESTING_VERSION" = "-h" ] || [ "$TESTING_VERSION" = "--help" ]; then
	echo "usage: $(basename $0) [version]"
	echo "If no version is specified, defaults to v3.10.2"
	exit 0
fi

TESTING_VERSION=$(echo $TESTING_VERSION | sed -r 's/^v?(.+)/v\1/')
echo "Using release $TESTING_VERSION"

RELEASE_DIR="gochan-${TESTING_VERSION}_linux"
RELEASE_GZ="$RELEASE_DIR.tar.gz"
RELEASE_URL="https://github.com/gochan-org/gochan/releases/download/$TESTING_VERSION/$RELEASE_GZ"

if [ "$USER" != "vagrant" ]; then
	echo "This must be run in the vagrant VM (expected \$USER to be vagrant, got $USER)"
	exit 1
fi

cd
if [ ! -e "$RELEASE_GZ" ]; then
	echo "Downloading $RELEASE_URL"
	wget -q --show-progress $RELEASE_URL
fi
rm -rf $RELEASE_DIR
echo "Extracting $RELEASE_GZ"
tar -xf gochan-${TESTING_VERSION}_linux.tar.gz
cd $RELEASE_DIR
mkdir -p log

EXAMPLE_CONFIG="examples/configs/gochan.example.json"
if [[ $TESTING_VERSION =~ ^v3\.[0-8]\. ]]; then
	EXAMPLE_CONFIG="sample-configs/gochan.example.json"
fi
echo "Using template config at $EXAMPLE_CONFIG"
if [ ! -f gochan.json ]; then
	cp -f $EXAMPLE_CONFIG gochan.json
	echo "Modifying $PWD/gochan.json for testing migration"
	sed -i gochan.json \
		-e 's/"Port": .*/"Port": 9000,/' \
		-e 's/"UseFastCGI": false/"UseFastCGI": true/' \
		-e "s/\"DBtype\": .*/\"DBtype\": \""$DBTYPE"\",/" \
		-e 's/"DBpassword": ""/"DBpassword": "gochan"/' \
		-e 's/"LogDir": .*/"LogDir": "log",/' \
		-e 's/"TemplateDir": .*/"TemplateDir": "templates",/' \
		-e 's#"DocumentRoot": .*#"DocumentRoot": "/srv/gochan",#' \
		-e 's/"DBname": "gochan"/"DBname": "gochan_prev"/' \
		-e 's/"DBprefix": .*/"DBprefix": "",/' \
		-e 's/"SiteName": "Gochan"/"SiteName": "Gochan Migration Test"/' \
		-e 's/"SiteSlogan": ""/"SiteSlogan": "Gochan instance used for testing gochan migration"/' \
		-e 's/"DebugMode": false/"DebugMode": true/' \
		-e 's/"Verbosity": 0/"Verbosity": 1/' \
		-e 's/"GeoIPType": .*/"GeoIPType": "",/' \
		-e 's/"EnableGeoIP": .*/"EnableGeoIP": false,/'
fi

if [ "$DBTYPE" = "mysql" ]; then
	echo "Creating testing MySQL DB 'gochan_prev' if it doesn't already exist"
	sudo mysql <<- EOF
	DROP DATABASE IF EXISTS gochan_prev;
	CREATE DATABASE IF NOT EXISTS gochan_prev;
	GRANT USAGE ON *.* TO gochan IDENTIFIED BY 'gochan'; \
	GRANT ALL PRIVILEGES ON gochan_prev.* TO gochan; \
	SET PASSWORD FOR 'gochan'@'%' = PASSWORD('gochan'); \
	FLUSH PRIVILEGES;
	EOF
elif [ "$DBTYPE" = "postgresql" ]; then
	echo "Creating testing PostgreSQL DB 'gochan_prev' if it doesn't already exist"
	sed -i gochan.json \
		-e 's/"DBhost": ".*"/"DBhost": "127.0.0.1"/'
	sudo -u postgres psql -f - <<- EOF1
	DROP DATABASE IF EXISTS gochan_prev;
	CREATE DATABASE gochan_prev;
	GRANT ALL PRIVILEGES ON DATABASE gochan_prev TO gochan;
	EOF1
elif [ "$DBTYPE" = "sqlite3" ]; then
	sed -i gochan.json \
		-e 's/"DBhost": ".*"/"DBhost": "gochan.db"/'
	rm -f gochan.db
else
	echo "Currently using unsupported \$DBTYPE: $DBTYPE"
	exit 1
fi

sudo ./gochan