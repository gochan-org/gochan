#!/usr/bin/env bash

OLD_VERSION="2.12.0"
OLD_GCDIR="gochan-v${OLD_VERSION}_linux"
OLD_URL="https://github.com/gochan-org/gochan/releases/download/v$OLD_VERSION/$OLD_GCDIR.tar.gz"

if [ "$USER" = "root" ]; then
	echo "This testing script isn't intended to be run as root but will still probably run anyway."
	read -p "Press enter to continue anyway or ctrl+c to exit
"
fi

pgrep gochan > /dev/null
if [ "$?" = "0" ]; then
	cat - <<- EOF
	A gochan instance is currently running. This script is intended for testing gochan migration,
	so only one instance should be running at a time
	EOF
	exit 1
fi


if [ -z "$1" ] || [ "$1" = "install" ]; then
	if [ -e ~/$OLD_GCDIR ]; then
		echo "Previous release is already installed, run $0 uninstall && $0"
		exit 1
	fi
	sudo mysql < /vagrant/vagrant/migrationtest/pre2021/gochan-${OLD_VERSION}-db.sql
	cd ~
	wget $OLD_URL
	tar -xvf "$OLD_GCDIR.tar.gz"
	rm "$OLD_GCDIR.tar.gz"
	cd $OLD_GCDIR
	cp sample-configs/gochan.example.json gochan.json

	sed -i gochan.json \
		-e 's/"Port": 8080/"Port": 9000/' \
		-e 's/"UseFastCGI": false/"UseFastCGI": true/' \
		-e "s/\"DBtype\": .*/\"DBtype\": \"mysql\",/" \
		-e "s/\"DBname\": .*/\"DBname\": \"gochan_pre2021_db\",/" \
		-e 's/"DBpassword": ""/"DBpassword": "gochan"/' \
		-e 's/"Verbosity": 0/"Verbosity": 1/' \
		-e 's/"DebugMode": false/"DebugMode": true/'

	mv gochan{,_$OLD_VERSION}
	mkdir -p html/test/{,res,src,thumb}
	echo ""
	echo "gochan v${OLD_VERSION} is ready to go. To start it, run"
	echo "screen -S gochan_$OLD_VERSION"
	echo "cd ~/$OLD_GCDIR"
	echo "./gochan_$OLD_VERSION"
elif [ "$1" = "uninstall" ]; then
	sudo mysqladmin -f DROP gochan_pre2021_db
	killall "gochan_$OLD_VERSION"; rm -rf ~/$OLD_GCDIR
else
	echo "Invalid argument. Usage is $1 [install|uninstall]"
fi