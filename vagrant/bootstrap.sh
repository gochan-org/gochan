#!/bin/bash
# Vagrant provision script

set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

if [ -z "$DBTYPE" ]; then
	echo "DBTYPE environment variable not set, must be 'mysql', 'postgresql', or 'sqlite3'"
	exit 1
fi

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
	systemctl enable mysql
	systemctl start mysql &
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
	# using SQLite (mostly stable)
	apt-get -y install sqlite3
elif [ "$DBTYPE" == "mssql" ]; then
	# using Microsoft SQL Server (currently unsupported)
	echo "Microsoft SQL Server not supported yet";
	exit 1
else
	echo "Unsupported DB type: $DBTYPE"
	exit 1
fi

apt-get -y install git subversion mercurial nginx ffmpeg golang-1.10
mkdir -p /root/bin
ln -s /usr/lib/go-1.10/bin/* /root/bin/
export PATH=$PATH:/root/bin
echo "export PATH=$PATH:/root/bin" >> /root/.bashrc

rm -f /etc/nginx/sites-enabled/* /etc/nginx/sites-available/*
ln -sf /vagrant/sample-configs/gochan-fastcgi.nginx /etc/nginx/sites-available/gochan.nginx
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

mkdir -p /etc/gochan
cp /vagrant/sample-configs/gochan.example.json /etc/gochan/gochan.json

sed -i /etc/gochan/gochan.json \
	-e 's/"Port": 8080/"Port": 9000/' \
	-e 's/"UseFastCGI": false/"UseFastCGI": true/' \
	-e 's/"DomainRegex": ".*"/"DomainRegex": "(https|http):\\\/\\\/(.*)\\\/(.*)"/' \
	-e 's#"DocumentRoot": "html"#"DocumentRoot": "/srv/gochan"#' \
	-e 's#"TemplateDir": "templates"#"TemplateDir": "/usr/local/share/gochan/templates"#' \
	-e 's#"LogDir": "log"#"LogDir": "/var/log/gochan"#' \
	-e 's/"DBpassword": ""/"DBpassword": "gochan"/' \
	-e 's/"Verbosity": 0/"Verbosity": 1/'

if [ "$DBTYPE" = "postgresql" ]; then
	sed -i /etc/gochan/gochan.json \
		-e 's/"DBtype": ".*"/"DBtype": "postgres"/' \
		-e 's/"DBhost": ".*"/"DBhost": "127.0.0.1"/'
elif [ "$DBTYPE" = "sqlite3" ]; then
	sed -i /etc/gochan/gochan.json \
		-e 's/"DBtype": ".*"/"DBtype": "sqlite3"/' \
		-e 's/"DBhost": ".*"/"DBhost": "gochan.db"/'
fi

# a convenient script for connecting to the db, whichever type we're using
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
chmod +x /home/vagrant/dbconnect.sh

cat <<EOF >>/home/vagrant/.bashrc
export PATH=$PATH:/home/vagrant/bin
export DBTYPE=$DBTYPE
export GOPATH=/vagrant/lib
EOF

cat <<EOF >>/root.bashrc
export GOPATH=/vagrant/lib
EOF
export GOPATH=/vagrant/lib

cd /vagrant
su - vagrant <<EOF
mkdir /home/vagrant/bin
ln -s /usr/lib/go-1.10/bin/* /home/vagrant/bin/ 
mkdir -p /vagrant/lib
source /home/vagrant/.bashrc
export GOPATH=/vagrant/lib
cd /vagrant
make dependencies
make
EOF
make install

# if [ -d /lib/systemd ]; then
# 	systemctl start gochan.service
# fi

cat - <<EOF
Server set up. To access the virtual machine, run 'vagrant ssh'. Then, to start the gochan server,
run 'sudo systemctl start gochan.service'. The virtual machine is set to run gochan on startup, so you
will not need to do this every time you start it. You can access it from a browser at http://172.27.0.3/
The first time gochan is run, it will create a simple /test/ board.
EOF
