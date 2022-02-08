#!/usr/bin/env bash

# Shell script that downloads a pre-migration gochan release for testing gochan-migration
# This should only be used in a development environment

TESTING_VERSION="v2.12.0"
RELEASE_DIR="gochan-${TESTING_VERSION}_linux"
RELEASE_GZ="$RELEASE_DIR.tar.gz"
RELEASE_URL="https://github.com/gochan-org/gochan/releases/download/$TESTING_VERSION/$RELEASE_GZ"

if [ -z "$STY" ]; then
	echo "This command should be run from a screen instance"
	echo "Example: screen -S get_pre2021 $0"
	exit 1
fi

if [ "$USER" != "vagrant" ]; then
	echo "This must be run in the vagrant VM (expected \$USER to be vagrant, got $USER)"
	exit 1
fi

cd ~
rm -f $RELEASE_GZ
echo "Downloading $RELEASE_GZ"
wget -q --show-progress $RELEASE_URL
echo "Extracting $RELEASE_GZ"
tar -xf gochan-v2.12.0_linux.tar.gz
cd $RELEASE_DIR

cp sample-configs/gochan.example.json gochan.json
echo "Modifying $PWD/gochan.json for testing migration"
sed -i gochan.json \
	-e 's/"Port": .*/"Port": 9000,/' \
	-e 's/"UseFastCGI": false/"UseFastCGI": true/' \
	-e "s/\"DBtype\": .*/\"DBtype\": \""$DBTYPE"\",/" \
	-e 's/"DBpassword": ""/"DBpassword": "gochan"/' \
	-e 's/"DBname": "gochan"/"DBname": "gochan_pre2021"/' \
	-e 's/"SiteName": "Gochan"/"SiteName": "Gochan pre-2021"/' \
	-e 's/"SiteSlogan": ""/"SiteSlogan": "Gochan instance used for testing migrating pre-2021"/' \
	-e 's/"DebugMode": false/"DebugMode": true/' \
	-e 's/"Verbosity": 0/"Verbosity": 1/'

if [ "$DBTYPE" = "mysql" ]; then
	echo "Creating pre-2021 MySQL DB 'gochan_pre2021' if it doesn't already exist"
	sudo mysql <<- EOF1
	CREATE DATABASE IF NOT EXISTS gochan_pre2021;
	GRANT USAGE ON *.* TO gochan IDENTIFIED BY 'gochan'; \
	GRANT ALL PRIVILEGES ON gochan_pre2021.* TO gochan; \
	SET PASSWORD FOR 'gochan'@'%' = PASSWORD('gochan');
	FLUSH PRIVILEGES;
	EOF1
elif [ "$DBTYPE" = "postgresql" ]; then
	echo "Creating pre-2021 PostgreSQL DB 'gochan_pre2021' if it doesn't already exist"
	sed -i /etc/gochan/gochan.json \
		-e 's/"DBhost": ".*"/"DBhost": "127.0.0.1"/'
	sudo -u postgres psql -f - <<- EOF1
	CREATE DATABASE gochan_pre2021;
	GRANT ALL PRIVILEGES ON DATABASE gochan_pre2021 TO gochan;
	EOF1
else
	echo "Currently using unsupported \$DBTYPE: $DBTYPE"
	exit 1
fi

sudo ./gochan