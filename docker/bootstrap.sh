#!/bin/bash
# Docker boostrap script

set -euo pipefail

if [ -z "$DBTYPE" ]; then
	echo "DBTYPE environment variable not set, must be 'mysql', 'postgresql', or 'sqlite3'"
	exit 1
fi

if [ -z "$GCVERSION"]; then
	echo "Gochan version not set in Dockerfile, required to download from the repo"
	exit 1
fi

if [ -f /usr/local/bin/gochan ]; then
	/usr/local/bin/gochan
	exit $?
fi

COMMUNITY_REPO=`grep -Eo '[^#]+community' -m 1 /etc/apk/repositories`

echo $COMMUNITY_REPO >> /etc/apk/repositories
GCURL="https://github.com/Eggbertx/gochan/releases/download/$GCVERSION/gochan-${GCVERSION}_linux64.tar.gz"

apk update && apk upgrade

if [ "$DBTYPE" == "postgresql" ]; then
	# using PostgreSQL (mostly stable)
	apk add postgresql postgresql-contrib sudo
	rc-update add postgresql default
	/etc/init.d/postgresql start
	echo "127.0.0.1:5432:gochan:gochan:gochan" > /root/.pgpass
	chmod 0600 /root/.pgpass
	sudo -u postgres psql -f - <<- EOF
	CREATE USER gochan PASSWORD 'gochan';
	CREATE DATABASE gochan;
	GRANT ALL PRIVILEGES ON DATABASE gochan TO gochan;
	EOF
	wait
else
	echo "Unsupported DB type: $DBTYPE (currently only PostgreSQL is supported for Docker containers"
	exit 1
fi

apk add git subversion libc-dev mercurial nginx ffmpeg

rm -f /etc/nginx/sites-enabled/* /etc/nginx/sites-available/*
ln -sf /opt/gochan/gochan-fastcgi.nginx /etc/nginx/sites-available/gochan.nginx
ln -sf /etc/nginx/sites-available/gochan.nginx /etc/nginx/sites-enabled/

mkdir -p /opt/gochan/lib
cd /opt/gochan
export GOPATH=/opt/gochan/lib
# mkdir /root/bin
# ln -s /usr/lib/go-1.10/bin/* /root/bin/ 
# export PATH="$PATH:/home/vagrant/bin"

cat << EOF >>/root/.bashrc
export GOPATH=$GOPATH
export DBTYPE=$DBTYPE
EOF

./build.sh dependencies
./build.sh
./build.sh install -s
echo "Done installing"

if [ -d /lib/systemd ]; then
	ln -s /opt/gochan/gochan.service /lib/systemd/system/gochan.service
	systemctl enable gochan.service
fi

cp gochan.example.json /etc/gochan/gochan.json

sed -i /etc/gochan/gochan.json \
	-e 's/"Port": 8080/"Port": 9000/' \
	-e 's/"UseFastCGI": false/"UseFastCGI": true/' \
	-e 's/"DomainRegex": ".*"/"DomainRegex": "(https|http):\\\/\\\/(.*)\\\/(.*)"/' \
	-e 's#"DocumentRoot": "html"#"DocumentRoot": "/srv/gochan"#' \
	-e 's#"TemplateDir": "templates"#"TemplateDir": "/usr/local/share/gochan/templates"#' \
	-e 's#"LogDir": "log"#"LogDir": "/var/log/gochan"#' \
	-e 's/"DBpassword": ""/"DBpassword": "gochan"/' \
	-e 's/"RandomSeed": ""/"RandomSeed": "abc123"/' \
	-e 's/"Verbosity": 0/"Verbosity": 1/'

if [ "$DBTYPE" = "postgresql" ]; then
	sed -i /etc/gochan/gochan.json \
		-e 's/"DBtype": ".*"/"DBtype": "postgres"/' \
		-e 's/"DBhost": ".*"/"DBhost": "127.0.0.1"/'
elif [ "$DBTYPE" = "sqlite3" ]; then
	sed -i /etc/gochan/gochan.json \
		-e 's/"DBtype": ".*"/"DBtype": "sqlite3"/' \
		-e 's/"DBhost": ".*"/"DBhost": "/usr/local/share/gochan/gochan.db"/'
fi

echo
echo "Container set up, please browse to http://172.27.0.3/manage to complete installation."
