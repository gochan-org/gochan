#!/bin/bash
# Vagrant provision script, kinda sorta based on Ponychan's

set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
export GOCHAN_PATH=/home/vagrant/gochan
export GOPATH=/vagrant/lib
export PATH="$PATH:/usr/lib/go-1.10/bin"

apt-get update
apt-get -y upgrade
apt-get -y install git subversion mercurial golang-1.10 nginx redis-server mariadb-server mariadb-client ffmpeg

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

mkdir -p /vagrant/lib || true
cd /vagrant
su vagrant
export GOCHAN_PATH=/home/vagrant/gochan
export GOPATH=/vagrant/lib
export PATH="$PATH:/usr/lib/go-1.10/bin"

function changePerms {
	chmod -R 755 $1 
	chown -R vagrant:vagrant $1
}

function makeLink {
	ln -sf /vagrant/$1 $GOCHAN_PATH/$1
}

cat - <<EOF >>/home/vagrant/.bashrc
export GOPATH=/vagrant/lib
export GOCHAN_PATH=/home/vagrant/gochan
EOF

go get github.com/disintegration/imaging
go get github.com/nranchev/go-libGeoIP
go get github.com/nyarla/go-crypt
go get github.com/go-sql-driver/mysql
go get golang.org/x/crypto/bcrypt
go get github.com/frustra/bbcode
make verbose

rm -f $GOCHAN_PATH/gochan
rm -f $GOCHAN_PATH/initialsetupdb.sql

install -m 775 -o vagrant -g vagrant -d $GOCHAN_PATH
makeLink html
makeLink log
makeLink gochan
makeLink templates
makeLink initialsetupdb.sql
changePerms $GOCHAN_PATH

if [ ! -e "$GOCHAN_PATH/gochan.json" ]; then
	install -m 775 -o vagrant -g vagrant -T /vagrant/gochan.example.json $GOCHAN_PATH/gochan.json
fi

sed -e 's/"Port": 8080,/"Port": 9000,/' -i $GOCHAN_PATH/gochan.json
sed -e 's/"UseFastCGI": false,/"UseFastCGI": true,/' -i /$GOCHAN_PATH/gochan.json
sed -e 's/"DomainRegex": ".*",/"DomainRegex": "(https|http):\\\/\\\/(.*)\\\/(.*)",/' -i $GOCHAN_PATH/gochan.json
sed -e 's/"DBpassword": ""/"DBpassword": "gochan"/' -i /home/vagrant/gochan/gochan.json
sed -e 's/"RandomSeed": ""/"RandomSeed": "abc123"/' -i $GOCHAN_PATH/gochan.json

echo
echo "Server set up, please run \"vagrant ssh\" on your host machine, and"
echo "\"cd /home/vagrant/gochan && ./gochan\" in the guest. Then browse to http://172.27.0.3/manage"
echo "to complete installation (TODO: add further instructions as default initial announcement"
echo "or /manage?action=firstrun)"
