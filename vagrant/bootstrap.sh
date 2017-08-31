#!/bin/bash
# Vagrant provision script, kinda sorta based on Ponychan's

set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

function changePerms {
	chmod -R 755 $1 
	chown -R ubuntu:ubuntu $1
}

function makeLink {
	ln -sf /vagrant/$1 $GOCHAN_PATH/$1
}

export GOCHAN_PATH=/home/ubuntu/gochan
apt-get update
apt-get -y install git subversion mercurial golang nginx redis-server mariadb-server mariadb-client #gifsicle

# Make sure any imported database is utf8mb4
# http://mathiasbynens.be/notes/mysql-utf8mb4
# Put in /etc/mysql/conf.d/local.cnf
cat - <<EOF123 >/etc/mysql/conf.d/local.cnf
[client]
default-character-set = utf8mb4

[mysql]
default-character-set = utf8mb4

[mysqld]
character-set-client-handshake = FALSE
character-set-server = utf8mb4
collation-server = utf8mb4_unicode_ci
default-storage-engine = innodb
EOF123

mysql -uroot -e "CREATE DATABASE IF NOT EXISTS gochan; \
GRANT USAGE ON *.* TO gochan IDENTIFIED BY ''; \
GRANT ALL PRIVILEGES ON gochan.* TO gochan; \
SET PASSWORD FOR 'gochan'@'%' = PASSWORD('gochan');
FLUSH PRIVILEGES;"

cat - <<EOF123 >/etc/mysql/conf.d/open.cnf
[mysqld]
bind-address = 0.0.0.0
EOF123

service mysql restart &
wait
rm -f /etc/nginx/sites-enabled/* /etc/nginx/sites-available/*
cp -f /vagrant/gochan-fastcgi.nginx /etc/nginx/sites-available/gochan.nginx
ln -sf /etc/nginx/sites-available/gochan.nginx /etc/nginx/sites-enabled/

# VirtualBox shared folders don't play nicely with sendfile.
sed -e 's/sendfile on;/sendfile off;/' -i /etc/nginx/nginx.conf
service nginx restart


mkdir -p /vagrant/lib || true
export GOPATH=/vagrant/lib
cd /vagrant
su ubuntu
go get github.com/disintegration/imaging
go get github.com/nranchev/go-libGeoIP
go get github.com/nyarla/go-crypt
go get github.com/go-sql-driver/mysql
go get golang.org/x/crypto/bcrypt
make verbose

rm -f $GOCHAN_PATH/gochan
rm -f $GOCHAN_PATH/initialsetupdb.sql

install -m 775 -o ubuntu -g ubuntu -d $GOCHAN_PATH
makeLink html
makeLink log
makeLink gochan
makeLink templates
makeLink initialsetupdb.sql
changePerms $GOCHAN_PATH

if [ ! -e "$GOCHAN_PATH/gochan.json" ]; then
	install -m 775 -o ubuntu -g ubuntu -T /vagrant/gochan.example.json $GOCHAN_PATH/gochan.json
fi

sed -e 's/"Port": 8080,/"Port": 9000,/' -i $GOCHAN_PATH/gochan.json
sed -e 's/"UseFastCGI": false,/"UseFastCGI": true,/' -i /$GOCHAN_PATH/gochan.json
sed -e 's/"DomainRegex": ".*",/"DomainRegex": "(https|http):\\\/\\\/(.*)\\\/(.*)",/' -i $GOCHAN_PATH/gochan.json
sed -e 's/"DBpassword": ""/"DBpassword": "gochan"/' -i /home/ubuntu/gochan/gochan.json
sed -e 's/"RandomSeed": ""/"RandomSeed": "abc123"/' -i $GOCHAN_PATH/gochan.json

echo
echo "Server set up, please run \"vagrant ssh\" on your host machine, and"
echo "\"cd ~/gochan && ./gochan\" in the guest. Then browse to http://172.27.0.3/manage"
echo "to complete installation (TODO: add further instructions as default initial announcement"
echo "or /manage?action=firstrun)"
