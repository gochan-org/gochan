#!/usr/bin/env bash

VERSION=`cat version`
GOOS_ORIG=$GOOS

function copyStuff {
	mkdir $DIRNAME
	make release
	mkdir $DIRNAME/html
	cp -r html/css $DIRNAME/html/css
	cp -r html/error $DIRNAME/html/error
	cp -r html/javascript $DIRNAME/html/javascript
	touch $DIRNAME/html/firstrun.html
	cp html/firstrun.html $DIRNAME/html/firstrun.html
	mkdir $DIRNAME/log
	cp -r templates $DIRNAME
	cp initialsetupdb.sql $DIRNAME
	cp *.nginx $DIRNAME
	cp README.md $DIRNAME
	cp LICENSE $DIRNAME
	cp gochan.example.json $DIRNAME
}

export GOOS=linux
export DIRNAME=releases/gochan-v${VERSION}_${GOOS}64/
copyStuff
cp gochan.service $DIRNAME
cd releases
tar -zcvf gochan-v${VERSION}_${GOOS}-64.tar.gz gochan-v${VERSION}_${GOOS}64/
cd ..

export GOOS=darwin
export DIRNAME=releases/gochan-v${VERSION}_macos64/
copyStuff
cd releases
tar -zcvf gochan-v${VERSION}_macos64.tar.gz gochan-v${VERSION}_macos64/
cd ..

export GOOS=windows
export DIRNAME=releases/gochan-v${VERSION}_${GOOS}64/
copyStuff
cd releases
zip gochan-v${VERSION}_${GOOS}-64.zip gochan-v${VERSION}_${GOOS}64/*
cd ..

export GOOS=$GOOS_ORIG
