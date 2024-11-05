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

echo "pinging database $DATABASE_HOST:$DATABASE_PORT, DBTYPE: '$DBTYPE'"
./docker/wait-for.sh "$DATABASE_HOST:$DATABASE_PORT" -t 30
gochan