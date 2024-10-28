#!/bin/sh

set -euo pipefail

cd /opt/gochan

if [ ! -L /var/www/gochan/js ]; then
    echo "Build stage didn't create links in /var/www/gochan"
    exit 1
fi

if [ ! -f /etc/gochan/.installed ]; then
    git config --global --add safe.directory /opt/gochan
    echo "Creating gochan executable"
    ./build.py
    touch /etc/gochan/.installed
fi

echo "pinging db, DBTYPE: '$DBTYPE'"
./docker/wait-for.sh "$DATABASE_HOST:$DATABASE_PORT" -t 30
./gochan