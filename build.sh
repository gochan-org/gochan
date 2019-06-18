#!/bin/bash

set -eo pipefail

BIN=gochan
VERSION=`cat version`
BUILDTIME=`date +%y%m%d.%H%M`

GCFLAGS=-trimpath=$PWD
ASMFLAGS=-trimpath=$PWD
LDFLAGS="-X main.versionStr=$VERSION -w -s"


if [ -z "$GOPATH" ]; then
	echo "GOPATH not set. Please run 'export GOPATH=$PWD/lib' (or wherever you prefer) and run this again."
	exit 1
fi

function usage {
	echo "usage: $0 [command or valid \$GOOS]"
	echo "commands:"
	echo "  clean"
	echo "  dependencies      install necessary dependencies"
	echo "  docker-image      create Docker image (not yet implemented)"
	echo "  help              show this message and exit"
	echo "  release [GOOS]    create release archives for Linux, macOS, and Windows"
	echo "                    or a specific platform, if specified"
	echo ""
	echo "Any other \"command\" will be treated as a GOOS to build gochan"
	echo "If no arguments are given, gochan will be built for the current OS"
	exit 0
}

function build {
	GCOS=$GOOS
	if [ -n "$1" ]; then
		if [ "$1" = "macos" ]; then GCOS="darwin"; else GCOS=$1; fi
	fi

	if [ "$GCOS" = "windows" ]; then BIN=$BIN.exe; fi

	buildCmd="GOOS=$GCOS go build -v -gcflags=$GCFLAGS -asmflags=$ASMFLAGS -ldflags \"$LDFLAGS\" -o $BIN ./src"

	if [ "$GCOS" = "windows" ]; then
		buildCmd="GOARCH=amd64 CC='x86_64-w64-mingw32-gcc -fno-stack-protector -D_FORTIFY_SOURCE=0 -lssp' $buildCmd"
	fi
	if [ -n "$GCOS" ]; then
		echo "Building gochan for '$GCOS'"
	else 
		echo "Building gochan for native system"
	fi
	bash -c "$buildCmd"
}

function release {
	GCOS=$GOOS
	if [ -n "$1" ]; then
		if [ "$1" = "darwin" ]; then GCOS="macos"; else GCOS=$1; fi
	fi

	DIRNAME=releases/gochan-v${VERSION}_${GCOS}64/

	mkdir -p $DIRNAME
	build $1
	if [ "$GCOS" = "darwin" ]; then GCOS="macos"; fi

	cp $BIN $DIRNAME
	if [ "$GCOS" = "linux" ]; then
		cp gochan.service $DIRNAME
	fi
	mkdir -p $DIRNAME/html
	cp -r sass $DIRNAME
	cp -r html/css $DIRNAME/html/css
	cp -r html/error $DIRNAME/html/error
	cp -r html/javascript $DIRNAME/html/javascript
	touch $DIRNAME/html/firstrun.html
	cp html/firstrun.html $DIRNAME/html/firstrun.html
	mkdir -p $DIRNAME/log
	cp -r templates $DIRNAME
	cp initdb_*.sql $DIRNAME
	cp *.nginx $DIRNAME
	cp README.md $DIRNAME
	cp LICENSE $DIRNAME
	cp gochan.example.json $DIRNAME


	cd releases
	if [ "$GCOS" = "windows" ] || [ "$GCOS" = "macos" ]; then
		zip gochan-v${VERSION}_${GCOS}64.zip gochan-v${VERSION}_${GCOS}64/*
	else
		tar -zcvf gochan-v${VERSION}_${GCOS}64.tar.gz gochan-v${VERSION}_${GCOS}64/
	fi
	cd ..
}

if [ $# = 0 ]; then
	build
	exit 0
fi

case "$1" in
	clean)
		rm -f $BIN
		rm -f $BIN.exe
		rm -rf releases
		;;
	dependencies)
		go get -v \
			github.com/disintegration/imaging \
			github.com/nranchev/go-libGeoIP \
			github.com/go-sql-driver/mysql \
			github.com/lib/pq \
			golang.org/x/net/html \
			github.com/aquilax/tripcode \
			golang.org/x/crypto/bcrypt \
			github.com/frustra/bbcode \
			github.com/mattn/go-sqlite3
		;;
	docker-image)
		echo "Docker image creation not yet implemented"
		exit 1
		# docker build . -t="eggbertx/gochan"
		;;
	help)
		usage
		;;
	release)
		if [ -n "$2" ]; then
			release $2
			exit 1
		else
			release linux
			release macos
			release windows
		fi
		;;
	*)
		build $1
		;;
esac
