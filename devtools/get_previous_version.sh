#!/usr/bin/env bash

# Shell script that downloads a previous gochan release for testing gochan-migration -updatedb
# This should only be used in a development environment

TESTING_VERSION="v3.7.0"
RELEASE_DIR="gochan-${TESTING_VERSION}_linux"
RELEASE_GZ="$RELEASE_DIR.tar.gz"
RELEASE_URL="https://github.com/gochan-org/gochan/releases/download/$TESTING_VERSION/$RELEASE_GZ"

if [ "$USER" != "vagrant" ]; then
	echo "This must be run in the vagrant VM (expected \$USER to be vagrant, got $USER)"
	exit 1
fi

cd ~
rm -f $RELEASE_GZ
echo "Downloading $RELEASE_GZ"
wget -q --show-progress $RELEASE_URL
echo "Extracting $RELEASE_GZ"
tar -xf gochan-${TESTING_VERSION}_linux.tar.gz
cd $RELEASE_DIR

cp examples/configs/gochan.example.json gochan.json
echo "Modifying $PWD/gochan.json for testing migration"
sed -i gochan.json \
	-e 's/"Port": .*/"Port": 9000,/' \
	-e 's/"UseFastCGI": false/"UseFastCGI": true/' \
	-e "s/\"DBtype\": .*/\"DBtype\": \""$DBTYPE"\",/" \
	-e 's/"DBpassword": ""/"DBpassword": "gochan"/' \
	-e 's/"DBname": "gochan"/"DBname": "gochan_37"/' \
	-e 's/"SiteName": "Gochan"/"SiteName": "Gochan Migration Test"/' \
	-e 's/"SiteSlogan": ""/"SiteSlogan": "Gochan instance used for testing gochan-migrate -updatedb"/' \
	-e 's/"DebugMode": false/"DebugMode": true/' \
	-e 's/"Verbosity": 0/"Verbosity": 1/'

if [ "$DBTYPE" = "mysql" ]; then
	echo "Creating testing MySQL DB 'gochan_37' if it doesn't already exist"
	sudo mysql <<- EOF1
	CREATE DATABASE IF NOT EXISTS gochan_37;
	GRANT USAGE ON *.* TO gochan IDENTIFIED BY 'gochan'; \
	GRANT ALL PRIVILEGES ON gochan_37.* TO gochan; \
	SET PASSWORD FOR 'gochan'@'%' = PASSWORD('gochan');
	FLUSH PRIVILEGES;
	EOF1
elif [ "$DBTYPE" = "postgresql" ]; then
	echo "Creating testing PostgreSQL DB 'gochan_37' if it doesn't already exist"
	sed -i /etc/gochan/gochan.json \
		-e 's/"DBhost": ".*"/"DBhost": "127.0.0.1"/'
	sudo -u postgres psql -f - <<- EOF1
	CREATE DATABASE gochan_37;
	GRANT ALL PRIVILEGES ON DATABASE gochan_37 TO gochan;
	EOF1
else
	echo "Currently using unsupported \$DBTYPE: $DBTYPE"
	exit 1
fi

sudo ./gochan