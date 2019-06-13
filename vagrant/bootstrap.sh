#!/bin/bash
# Vagrant provision script

set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

export DBTYPE=postgresql

apt-get -y update && apt-get -y upgrade

if [ "$DBTYPE" == "mysql" ]; then
# Using MySQL (stable)
apt-get -y install mariadb-server mariadb-client 
mysql -uroot -e "CREATE DATABASE IF NOT EXISTS gochan; \
GRANT USAGE ON *.* TO gochan IDENTIFIED BY ''; \
GRANT ALL PRIVILEGES ON gochan.* TO gochan; \
SET PASSWORD FOR 'gochan'@'%' = PASSWORD('gochan');
FLUSH PRIVILEGES;"

systemctl enable mysql
systemctl start mysql &
wait
elif [ "$DBTYPE" == "postgresql" ]; then
# using PostgreSQL (mostly stable)
apt-get -y install postgresql postgresql-contrib

sudo -u postgres psql -f - << EOF
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
elif [ "$DBTYPE" == "sqlite3" ]; then
# using SQLite (mostly stable)
apt-get -y install sqlite3
elif [ "$DBTYPE" == "mssql" ]; then
# using Microsoft SQL Server (currently unsupported)
echo "Microsoft SQL Server not supported yet";
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
systemctl restart nginx &
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
	ln -sf /vagrant/$1 $GOCHAN_PATH/
}

cat << EOF >>/home/vagrant/.bashrc
export GOPATH=$GOPATH
export GOCHAN_PATH=$GOCHAN_PATH
export DBTYPE=$DBTYPE
EOF

# a couple convenience shell scripts, since they're nice to have
cat << EOF >/home/vagrant/dbconnect.sh
#!/usr/bin/env bash

if [ "$DBTYPE" = "mysql" ] || [ -z "$DBTYPE" ]; then
	mysql -stu gochan -D gochan -pgochan
elif [ "$DBTYPE" = "postgresql" ]; then
	psql -U gochan -h 127.0.0.1 gochan
elif [ "$DBTYPE" = "sqlite3" ]; then
	sqlite3 ~/gochan/gochan.db
else
	echo "DB type '$DBTYPE' not supported"
fi
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
	github.com/frustra/bbcode \
	github.com/mattn/go-sqlite3
make debug

rm -f $GOCHAN_PATH/gochan
rm -f $GOCHAN_PATH/initdb*.sql

install -m 775 -o vagrant -g vagrant -d $GOCHAN_PATH
makeLink html
makeLink log
makeLink gochan
makeLink templates
ln -sf /vagrant/initdb*.sql $GOCHAN_PATH/
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
	-e "w $GOCHAN_PATH/gochan.json"

if [ "$DBTYPE" = "postgresql" ]; then
	sed \
		-e 's/"DBtype": ".*",/"DBtype": "postgres",/' \
		-e 's/"DBhost": ".*",/"DBhost": "127.0.0.1",/' \
		-i $GOCHAN_PATH/gochan.json
elif [ "$DBTYPE" = "sqlite3" ]; then
	sed \
		-e 's/"DBtype": ".*",/"DBtype": "sqlite3",/' \
		-e 's/"DBhost": ".*",/"DBhost": "gochan.db",/' \
		-i $GOCHAN_PATH/gochan.json
fi

echo
echo "Server set up, please run \"vagrant ssh\" on your host machine and"
echo "(optionally) \"systemctl --user start gochan\" in the guest."
echo "Then browse to http://172.27.0.3/manage to complete installation."
# echo "If you want gochan to run on system startup run \"systemctl --user enable gochan\""
# TODO: add further instructions as default initial announcement or /manage?action=firstrun
