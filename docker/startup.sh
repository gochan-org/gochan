#!/bin/sh

set -euo pipefail

sed -i "/etc/gochan/gochan.json" \
	-e "s/\"Port\": 8080/\"Port\": 9000/" \
	-e "s/\"UseFastCGI\": false/\"UseFastCGI\": true/" \
	-e "s/\"Username\": \".*\",//" \
	-e "s#\"DocumentRoot\": \"html\"#\"DocumentRoot\": \"/srv/gochan\"#" \
	-e "s#\"TemplateDir\": \"templates\"#\"TemplateDir\": \"/usr/share/gochan/templates\"#" \
	-e "s#\"LogDir\": \"log\"#\"LogDir\": \"/var/log/gochan\"#" \
	-e "s/\"Verbosity\": 0/\"Verbosity\": 1/" \
	-e "s/\"DBtype\".*/\"DBtype\": \"${DBTYPE}\",/" \
	-e "s/\"DBhost\".*/\"DBhost\": \"tcp(${DATABASE_HOST}:${DATABASE_PORT})\",/" \
	-e "s/\"DBname\".*/\"DBname\": \"${DATABASE_NAME}\",/" \
	-e "s/\"DBusername\".*/\"DBusername\": \"${DATABASE_USER}\",/" \
	-e "s/\"DBpassword\".*/\"DBpassword\": \"${DATABASE_PASSWORD}\",/"

nginx
echo "pinging db, DBTYPE: '$DBTYPE'"

/opt/gochan/docker/wait-for.sh "$DATABASE_HOST:$DATABASE_PORT" -t 30
/opt/gochan/gochan -rebuild all
/opt/gochan/gochan