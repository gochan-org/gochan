#!/usr/bin/env bash
if [ -z $GOPATH ]
then
	#export GOPATH=$PWD/lib
	echo "\$GOPATH not set. Please run 'export GOPATH=\$PWD/lib' (or wherever you prefer) and run this again."
	exit
fi
CGO_ENABLED=0
GOARCH=amd64
SUFFIX=""
if [[ $GOOS == "windows" ]]
then
	SUFFIX=".exe"
fi

go build -v  -ldflags "-w" -o gochan$SUFFIX ./src
# the -w ldflag omits debugging stuff
