#!/usr/bin/env bash

# used after vagrant provisioning to skip gochan-install steps and use generic configuration

set -euo pipefail

if [ ! -d /vagrant ]; then
	echo "This script is intended to be run in a Vagrant-provisioned VM"
	exit 1
fi

mkdir -p /etc/gochan
cp /vagrant/examples/configs/gochan.example.json /vagrant/gochan.json
ln -s /vagrant/gochan.json /etc/gochan/gochan.json
sed -i /vagrant/gochan.json \
	-e 's/"Port": 8080/"Port": 9000/' \
	-e 's/"UseFastCGI": false/"UseFastCGI": true/' \
	-e 's#"DocumentRoot": "html"#"DocumentRoot": "/srv/gochan"#' \
	-e 's#"TemplateDir": "templates"#"TemplateDir": "/usr/share/gochan/templates"#' \
	-e 's#"LogDir": "log"#"LogDir": "/var/log/gochan"#' \
	-e "s/\"DBtype\": .*/\"DBtype\": \"$DBTYPE\",/" \
	-e 's/"SiteHost": .*/"SiteHost": "192.168.56.3",/' \
	-e 's/"DBpassword": .*/"DBpassword": "gochan",/' \
	-e 's/"Verbosity": 0/"Verbosity": 1/'

if [ "$DBTYPE" = "postgresql" ]; then
	sed -i /etc/gochan/gochan.json \
		-e 's/"DBhost": ".*"/"DBhost": "127.0.0.1"/'
elif [ "$DBTYPE" = "sqlite3" ]; then
	sed -i /etc/gochan/gochan.json \
		-e 's#"DBhost": ".*"#"DBhost": "/etc/gochan/gochan.db"#'
fi