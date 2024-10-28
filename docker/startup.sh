#!/bin/sh

set -euo pipefail

echo "pinging db, DBTYPE: '$DBTYPE'"
/opt/gochan/docker/wait-for.sh "$DATABASE_HOST:$DATABASE_PORT" -t 30
/opt/gochan/gochan