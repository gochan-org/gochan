#!/bin/bash
# Vagrant provision script

set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
export GO_VERSION=1.16

if [ -z "$DBTYPE" ]; then
	echo "DBTYPE environment variable not set, must be 'mysql' or 'postgresql'."
	echo "sqlite3 support has been dropped but it may come back eventually."
	exit 1
fi
echo "Using DBTYPE $DBTYPE"

add-apt-repository -y ppa:longsleep/golang-backports
apt-get -y update && apt-get -y upgrade

if [ "$DBTYPE" == "mysql" ]; then
	# Using MySQL (stable)
	apt-get -y install mariadb-server mariadb-client 
	mysql -uroot <<- EOF
	CREATE DATABASE IF NOT EXISTS gochan;
	GRANT USAGE ON *.* TO gochan IDENTIFIED BY 'gochan'; \
	GRANT ALL PRIVILEGES ON gochan.* TO gochan; \
	SET PASSWORD FOR 'gochan'@'%' = PASSWORD('gochan');
	FLUSH PRIVILEGES;
	EOF
	systemctl enable mariadb
	systemctl start mariadb &
	wait
	if [ -d /lib/systemd ]; then
		cp /vagrant/sample-configs/gochan-mysql.service /lib/systemd/system/gochan.service
		systemctl daemon-reload
		systemctl enable gochan.service
	fi
elif [ "$DBTYPE" == "postgresql" ]; then
	# using PostgreSQL (mostly stable)
	apt-get -y install postgresql postgresql-contrib sudo

	systemctl start postgresql
	sudo -u postgres psql -f - <<- EOF
	CREATE USER gochan PASSWORD 'gochan';
	CREATE DATABASE gochan;
	GRANT ALL PRIVILEGES ON DATABASE gochan TO gochan;
	EOF

	echo "127.0.0.1:5432:gochan:gochan:gochan" > /home/vagrant/.pgpass
	chown vagrant:vagrant /home/vagrant/.pgpass
	chmod 0600 /home/vagrant/.pgpass
	systemctl enable postgresql
	systemctl start postgresql &
	wait
	if [ -d /lib/systemd ]; then
		cp /vagrant/sample-configs/gochan-postgresql.service /lib/systemd/system/gochan.service
		systemctl daemon-reload
		systemctl enable gochan.service
	fi
elif [ "$DBTYPE" == "sqlite3" ]; then
	apt install sqlite3
elif [ "$DBTYPE" == "mssql" ]; then
	# using Microsoft SQL Server (currently unsupported)
	echo "Microsoft SQL Server not supported yet";
	exit 1
else
	echo "Unsupported DB type: $DBTYPE"
	exit 1
fi

apt-get -y install git subversion mercurial nginx ffmpeg golang-${GO_VERSION}

ln -sf /usr/lib/go-${GO_VERSION}/bin/* /usr/local/bin/

rm -f /etc/nginx/sites-enabled/* /etc/nginx/sites-available/*
ln -sf /vagrant/sample-configs/gochan-fastcgi.nginx /etc/nginx/sites-available/gochan.nginx
ln -sf /etc/nginx/sites-available/gochan.nginx /etc/nginx/sites-enabled/

# VirtualBox shared folders don't play nicely with sendfile.
sed -e 's/sendfile on;/sendfile off;/' -i /etc/nginx/nginx.conf

# Make sure our shared directories are mounted before nginx starts
systemctl disable nginx
sed -i 's/WantedBy=multi-user.target/WantedBy=vagrant.mount/' /lib/systemd/system/nginx.service

# generate self-signed certificate since some browsers like Firefox and Chrome automatically do HTTPS requests
# this will likely show a warning in the browser, which you can ignore
openssl req -x509 -nodes -days 7305 -newkey rsa:2048 \
	-keyout /etc/ssl/private/nginx-selfsigned.key \
	-out /etc/ssl/certs/nginx-selfsigned.crt \
	-subj "/CN=192.168.56.3"

systemctl daemon-reload
systemctl enable nginx
systemctl restart nginx &
wait

mkdir -p /etc/gochan
cp /vagrant/sample-configs/gochan.example.json /etc/gochan/gochan.json

sed -i /etc/gochan/gochan.json \
	-e 's/"Port": 8080/"Port": 9000/' \
	-e 's/"UseFastCGI": false/"UseFastCGI": true/' \
	-e 's#"DocumentRoot": "html"#"DocumentRoot": "/srv/gochan"#' \
	-e 's#"TemplateDir": "templates"#"TemplateDir": "/usr/share/gochan/templates"#' \
	-e 's#"LogDir": "log"#"LogDir": "/var/log/gochan"#' \
	-e "s/\"DBtype\": .*/\"DBtype\": \"$DBTYPE\",/" \
	-e 's/"DBpassword": ""/"DBpassword": "gochan"/' \
	-e 's/"Verbosity": 0/"Verbosity": 1/'

if [ "$DBTYPE" = "postgresql" ]; then
	sed -i /etc/gochan/gochan.json \
		-e 's/"DBhost": ".*"/"DBhost": "127.0.0.1"/'
elif [ "$DBTYPE" = "sqlite3" ]; then
	sed -i /etc/gochan/gochan.json \
		-e 's#"DBhost": ".*"#"DBhost": "/etc/gochan/gochan.db"#'
fi

# a convenient script for connecting to the db, whichever type we're using
ln -s {/vagrant/devtools,/home/vagrant}/dbconnect.sh
chmod +x /home/vagrant/dbconnect.sh

# used for testing migration from the pre-2021 db schema
ln -s {/vagrant/devtools,/home/vagrant}/get_pre2021.sh
chmod +x get_pre2021.sh

cat <<EOF >>/home/vagrant/.bashrc
export DBTYPE=$DBTYPE
export GOPATH=/home/vagrant/go
EOF

su - vagrant <<EOF
mkdir -p /home/vagrant/go
source /home/vagrant/.bashrc
cd /vagrant/devtools
python build_initdb.py
cd ..
mkdir -p $GOPATH/src/github.com/gochan-org/gochan
cp -r pkg $GOPATH/src/github.com/gochan-org/gochan
./build.py
EOF

cd /vagrant
./build.py install
/vagrant/gochan -rebuild all

# if [ -d /lib/systemd ]; then
# 	systemctl start gochan.service
# fi

cat - <<EOF
Server set up. To access the virtual machine, run 'vagrant ssh'.
To start the gochan server, run 'sudo systemctl start gochan.service'.
To have gochan run at startup, run 'sudo systemctl enable gochan.service'

You can access it from a browser at http://192.168.56.3/
The first time gochan is run, it will create a simple /test/ board.

If you want to do frontend development, see frontend/README.md
EOF
