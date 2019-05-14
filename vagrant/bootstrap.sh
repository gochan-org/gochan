#!/bin/bash
# Vagrant provision script

set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

export DBTYPE=mysql

apt-get -y update && apt-get -y upgrade

if [ "$DBTYPE" == "mysql" ]; then
apt-get -y install mariadb-server mariadb-client 
# Make sure any imported database is utf8mb4
# http://mathiasbynens.be/notes/mysql-utf8mb4
# Put in /etc/mysql/conf.d/local.cnf
cat << EOF >/etc/mysql/conf.d/local.cnf
[client]
default-character-set = utf8mb4

[mysql]
default-character-set = utf8mb4

[mysqld]
character-set-client-handshake = FALSE
character-set-server = utf8mb4
collation-server = utf8mb4_unicode_ci
default-storage-engine = innodb
EOF

mysql -uroot -e "CREATE DATABASE IF NOT EXISTS gochan; \
GRANT USAGE ON *.* TO gochan IDENTIFIED BY ''; \
GRANT ALL PRIVILEGES ON gochan.* TO gochan; \
SET PASSWORD FOR 'gochan'@'%' = PASSWORD('gochan');
FLUSH PRIVILEGES;"

cat << EOF >/etc/mysql/conf.d/open.cnf
[mysqld]
bind-address = 0.0.0.0
EOF
elif [ "$DBTYPE" == "postgresql" ]; then
	# apt-get -y install postgresql postgresql-contrib
	# useradd gochan
	# passwd -d gochan
	# sudo -u postgres createuser -d gochan
	echo "PostgreSQL not supported yet"
	exit 1
elif [ "$DBTYPE" == "mssql" ]; then
	echo "Microsoft SQL Server not supported yet";
	exit 1
elif [ "$DBTYPE" == "sqlite" ]; then
	echo "SQLite not supported yet"
	exit 1
else
	echo "Invalid DB type: $DBTYPE"
	exit 1
fi

apt-get -y install git subversion mercurial golang-1.10 nginx ffmpeg

rm -f /etc/nginx/sites-enabled/* /etc/nginx/sites-available/*
ln -sf /vagrant/gochan-fastcgi.nginx /etc/nginx/sites-available/gochan.nginx
ln -sf /etc/nginx/sites-available/gochan.nginx /etc/nginx/sites-enabled/

# VirtualBox shared folders don't play nicely with sendfile.
sed -e 's/sendfile on;/sendfile off;/' -i /etc/nginx/nginx.conf

# Make sure our shared directories are mounted before nginx starts
systemctl disable nginx
sed -i 's/WantedBy=multi-user.target/WantedBy=vagrant.mount/' /lib/systemd/system/nginx.service
systemctl daemon-reload
systemctl enable nginx

systemctl restart nginx mysql &
wait

mkdir -p /vagrant/lib
cd /vagrant
su - vagrant
export GOCHAN_PATH=/home/vagrant/gochan
export GOPATH=/vagrant/lib
mkdir /home/vagrant/bin
ln -s /usr/lib/go-1.10/bin/* /home/vagrant/bin/ 
export PATH="$PATH:/home/vagrant/bin"

function changePerms {
	chmod -R 755 $1 
	chown -R vagrant:vagrant $1
}

function makeLink {
	ln -sf /vagrant/$1 $GOCHAN_PATH/$1
}

cat << EOF >>/home/vagrant/.bashrc
export GOPATH=/vagrant/lib
export GOCHAN_PATH=/home/vagrant/gochan
EOF

# a couple convenience shell scripts, since they're nice to have
cat << EOF >/home/vagrant/dbconnect.sh
#!/usr/bin/env bash

mysql -s -t -u gochan -D gochan -pgochan
EOF

cat << EOF >/home/vagrant/buildgochan.sh
#!/usr/bin/env bash

cd /vagrant && make debug && cd ~/gochan && ./gochan
EOF

chmod +x /home/vagrant/dbconnect.sh
chmod +x /home/vagrant/buildgochan.sh

go get \
	github.com/disintegration/imaging \
	github.com/nranchev/go-libGeoIP \
	github.com/go-sql-driver/mysql \
	github.com/lib/pq \
	golang.org/x/net/html \
	github.com/aquilax/tripcode \
	golang.org/x/crypto/bcrypt \
	github.com/frustra/bbcode
make debug

rm -f $GOCHAN_PATH/gochan
rm -f $GOCHAN_PATH/initdb.sql

install -m 775 -o vagrant -g vagrant -d $GOCHAN_PATH
makeLink html
makeLink log
makeLink gochan
makeLink templates
makeLink initdb.sql
changePerms $GOCHAN_PATH

mkdir -p /home/vagrant/.config/systemd/user/
ln -s /vagrant/gochan.service /home/vagrant/.config/systemd/user/gochan.service

sed /vagrant/gochan.example.json \
	-e 's/"Port": 8080,/"Port": 9000,/' \
	-e 's/"UseFastCGI": false,/"UseFastCGI": true,/' \
	-e 's/"DomainRegex": ".*",/"DomainRegex": "(https|http):\\\/\\\/(.*)\\\/(.*)",/' \
	-e 's/"DBpassword": ""/"DBpassword": "gochan"/' \
	-e 's/"RandomSeed": ""/"RandomSeed": "abc123"/' \
	-e 's/"Verbosity": 0/"Verbosity": 1/' \
	-e w\ $GOCHAN_PATH/gochan.json

echo
echo "Server set up, please run \"vagrant ssh\" on your host machine and"
echo "(optionally) \"systemctl --user start gochan\" in the guest."
echo "Then browse to http://172.27.0.3/manage to complete installation."
# echo "If you want gochan to run on system startup run \"systemctl --user enable gochan\""
# TODO: add further instructions as default initial announcement or /manage?action=firstrun
