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

if [ "$DBTYPE" != "sqlite3" ]; then
    echo "pinging database $DBHOST, DBTYPE: '$DBTYPE'"
    ./docker/wait-for.sh "$DBHOST" -t 30
fi
gochan