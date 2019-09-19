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
elif [ "$DBTYPE" == "postgresql" ]; then
	# using PostgreSQL (mostly stable)
	apt-get -y install postgresql postgresql-contrib sudo

	# if [ -n "$FROMDOCKER" ]; then
	# 	su -s /bin/sh postgres
	# fi
	if [ -n "$FROMDOCKER" ]; then
		service postgresql start
	else
		systemctl start postgresql
	fi
	sudo -u postgres psql -f - <<- EOF
	CREATE USER gochan PASSWORD 'gochan';
	CREATE DATABASE gochan;
	GRANT ALL PRIVILEGES ON DATABASE gochan TO gochan;
	EOF
	if [ -z "$FROMDOCKER" ]; then
		echo "127.0.0.1:5432:gochan:gochan:gochan" > /home/vagrant/.pgpass
		chown vagrant:vagrant /home/vagrant/.pgpass
		chmod 0600 /home/vagrant/.pgpass
		systemctl enable postgresql
		systemctl start postgresql &
	else
		update-rc.d postgresql enable
	fi
	wait
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

apt-get -y install git subversion mercurial nginx ffmpeg
if [ -z "$FROMDOCKER" ]; then
	apt-get -y install golang-1.10
fi

rm -f /etc/nginx/sites-enabled/* /etc/nginx/sites-available/*
ln -sf /vagrant/gochan-fastcgi.nginx /etc/nginx/sites-available/gochan.nginx
ln -sf /etc/nginx/sites-available/gochan.nginx /etc/nginx/sites-enabled/

# VirtualBox shared folders don't play nicely with sendfile.
sed -e 's/sendfile on;/sendfile off;/' -i /etc/nginx/nginx.conf

# Make sure our shared directories are mounted before nginx starts
# service nginx disable
update-rc.d nginx enable
sed -i 's/WantedBy=multi-user.target/WantedBy=vagrant.mount/' /lib/systemd/system/nginx.service
# systemctl daemon-reload
# service nginx enable
# service nginx restart &
wait

mkdir -p /vagrant/lib
cd /opt/gochan
export GOPATH=/opt/gochan/lib
# mkdir /home/vagrant/bin
# ln -s /usr/lib/go-1.10/bin/* /home/vagrant/bin/ 
# export PATH="$PATH:/home/vagrant/bin"

function changePerms {
	chmod -R 755 $1 
	chown -R vagrant:vagrant $1
}

cat << EOF >>/root/.bashrc
export GOPATH=$GOPATH
export DBTYPE=$DBTYPE
EOF

# a couple convenience shell scripts, since they're nice to have
cat << EOF >/root/dbconnect.sh
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

chmod +x /root/dbconnect.sh

./build.sh dependencies
./build.sh
./build.sh install -s
echo "Done installing"

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
		-e 's/"DBhost": ".*"/"DBhost": "gochan.db"/'
fi

# if [ -d /lib/systemd ]; then
# 	cp gochan.service /lib/systemd/system/gochan.service
# 	systemctl daemon-reload
# 	systemctl enable gochan.service
# 	systemctl start gochan.service
# fi

echo
echo "Server set up, please run \"vagrant ssh\" on your host machine."
echo "Then browse to http://172.27.0.3/manage to complete installation."
