#!/usr/bin/env bash
if [ -z $GOPATH ]
then
	#export GOPATH=$PWD/lib
	echo "\$GOPATH not set. Please run 'export GOPATH=\$PWD/lib' (or wherever you prefer) and run this again."
	exit
fi
GOCHAN_VERBOSE=0
GOCHAN_VERSION="0.9"
GOCHAN_BUILDTIME=$(date +%y%m%d.%H%M)
CGO_ENABLED=0
GOARCH=amd64
SUFFIX=""
if [[ $GOOS == "windows" ]]
then
	SUFFIX=".exe"
fi

go build -v  -ldflags "-w -X main.version=$GOCHAN_VERSION -X main.buildtime_str=$GOCHAN_BUILDTIME -X main.verbose_str=$GOCHAN_VERBOSE" -o gochan$SUFFIX ./src
# the -w ldflag omits debugging stuff
