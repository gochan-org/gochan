#!/bin/sh

cd /opt/gochan

linkwww() {
    if [ ! -L /var/www/gochan/$1 ]; then
        ln -sv /opt/gochan/html/$1 /var/www/gochan/
    fi
}

linkwww css
linkwww error
linkwww js
linkwww static
linkwww favicon.png
linkwww firstrun.html

mkdir -p /etc/gochan
if [ ! -e "/etc/gochan/gochan.json" ]; then
    echo "gochan.json not found in /etc/gochan/, moving /opt/gochan/gochan-init.json to /etc/gochan/gochan.json"
    mv /opt/gochan/gochan-init.json /etc/gochan/gochan.json
fi

if [ "$DB_TYPE" != "sqlite3" ]; then
    echo "pinging database $DB_HOST, DBTYPE: '$DB_TYPE'"
    ./docker/wait-for.sh "$DB_HOST" -t 30
fi

gochan