#!/usr/bin/env bash

set -eo pipefail

rootPkg="github.com/gochan-org/gochan"
pkgBase="pkg"
gopath=`go env GOPATH`
pkgDir=$gopath/pkg/linux_amd64/$rootPkg

if [ -z "$1" ]||[ "$1" == "--help" ]; then
cat - <<EOF
Usage:
$0 --all
	Build/install all subpackages
$0 --clean
	Removes the source and compiled package from the GOPATH
$0 [--help]
	Show this message
$0 $pkgBase/path/to/subpkg
	Compiles and installes the package to the GOPATH
EOF
elif [ "$1" == "--all" ]; then
	for f in pkg/*; do
		$0 $f
	done
	echo
	find $pkgDir
elif [ "$1" == "--clean" ]; then
	rm -rf ${pkgDir%/gochan}
else
	target=${1#*$pkgBase/}
	target=${target%/}

	echo "Building/installing $target in $rootPkg/$pkgBase/$target"
	go install $rootPkg/$pkgBase/$target
	
	if [ "$2" == "-v" ]; then
		find $pkgDir
	fi
fi