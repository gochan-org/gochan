#!/usr/bin/env bash

# used for connecting to gochan's database whether it's using MySQL/MariaDB or PostgreSQL

CFGFILE=""

# check for config file in locations in the order that gochan checks them, starting with the vagrant shared folder
if [ -f "/vagrant/gochan.json" ]; then
	CFGFILE="/vagrant/gochan.json"
elif [ -f "/usr/local/etc/gochan/gochan.json" ]; then
	CFGFILE="/usr/local/etc/gochan/gochan.json"
elif [ -f "/etc/gochan/gochan.json" ]; then
	CFGFILE="/etc/gochan/gochan.json"
else
	echo "No configuration file found, exiting"
	exit 1
fi

if [ "$DBTYPE" = "mysql" ] || [ -z "$DBTYPE" ]; then
	mysql -stu gochan -D gochan -pgochan
elif [ "$DBTYPE" = "postgresql" ]; then
	psql -U gochan -h 127.0.0.1 gochan
elif [ "$DBTYPE" = "sqlite3" ]; then
	DBFILE=$(sed -nr 's/^.*"DBhost":\s*"(.+)".*$/\1/p' $CFGFILE)
	sqlite3 $DBFILE
else
	echo "DB type '$DBTYPE' not supported"
fi
