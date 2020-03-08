#!/bin/bash

set -eo pipefail
if [ -e "version" ]; then version="v`cat version` "; fi
echo "Gochan ${version}build script"
echo ""
BIN=gochan
VERSION=`cat version`
BUILDTIME=`date +%y%m%d.%H%M`

GCFLAGS=-trimpath=$PWD
ASMFLAGS=-trimpath=$PWD
LDFLAGS="-X main.versionStr=$VERSION -w -s"
export CGO_ENABLED=1


if [ -z "$GOPATH" ]; then
	echo "GOPATH not set. Please run 'export GOPATH=$PWD/lib' (or wherever you prefer) and run this again."
	exit 1
fi

function usage {
	if [ -z "$2" ]; then
		cat - << EOF
Usage:
	$0 [command or valid GOOS] [command arguments]

Commands:
	clean		remove any built binaries and releases
	dependencies	install necessary gochan dependencies
	docker-image	create Docker image (not yet implemented)
	help		show this help message
	install		install gochan to the system or specified location
	release		create release archives for deployment
	sass		use Sass to transpile the sass source files

Any other "command" will be treated as a GOOS to build gochan. If no commands
are given, gochan will be built for the current OS
EOF
		exit
	fi
	case "$2" in
		clean)
			cat - <<- EOF
				Usage: $0 clean
				remove any built binaries and releases
			EOF
			;;
		dependencies)
			cat - <<- EOF
				Usage: $0 dependencies
				install necessary gochan dependencies
			EOF
			;;
		docker-image)
			cat - <<- EOF
				Usage: $0 docker-image
				create a Docker image (not yet implemented)
			EOF
			;;
		help)
			cat - <<- EOF
				Usage: $0 help
				show the help message and quits
			EOF
			;;
		install)
			cat - << EOF
Usage:
$0 install [--document-root /path/to/html] [--symlinks] [destination]
Installs gochan on the current system
Arguments:
	--document-root|--html /path/to/html
		install document root resources to specified path, otherwise
		they are installed to ./html/
	--symlinks|-s
		create symbolic links instead of copying, useful for testing

Install locations if no destination is provided:
	./gochan		=>	/usr/local/bin/gochan
	./gochan[.example].json	=>	/etc/gochan/gochan.json
	./templates/		=>	/usr/share/gochan/templates/
	./log =>		=>	/var/log/gochan/
/etc/gochan/gochan.json will only be created if it doesn't already exist
EOF
			;;
		release)
			cat - <<- EOF
				Usage: $0 release [GOOS]
				create release archives for Linux, macOS, and Windows or the specified platform
				for deployment
			EOF
			;;
		sass)
			cat - <<- EOF
			Usage: $0 sass [/path/to/html]
			use sass to transpile the sass source files to ./html/css or the specified
			document root
			EOF
			;;
		*)
			echo "Invalid command"
			;;
	esac
	exit
}

function build {
	GCOS=$GOOS
	if [ -n "$1" ]; then
		if [ "$1" = "macos" ]; then GCOS="darwin"; else GCOS=$1; fi
	fi

	if [ "$GCOS" = "windows" ]; then
		BIN=$BIN.exe
	fi
	
	if [ "$GCOS" = "darwin" ]; then
		echo "Cross-compilation to macOS has been temporarily disabled because of a cgo issue."
		echo "If you really need macOS support, build gochan from a macOS system."
		exit 1
	fi

	if [ -z "$GCOS" ]; then
		GCOS=`go env GOOS`
	fi

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
	mkdir -p $DIRNAME/sample-configs
	if [ "$GCOS" = "linux" ]; then
		strip $DIRNAME/$BIN
		cp sample-configs/gochan-mysql.service $DIRNAME/sample-configs
		cp sample-configs/gochan-postgresql.service $DIRNAME/sample-configs
		cp sample-configs/gochan-sqlite3.service $DIRNAME/sample-configs
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
	cp sample-configs/*.nginx $DIRNAME/sample-configs/
	cp README.md $DIRNAME
	cp LICENSE $DIRNAME
	cp sample-configs/gochan.example.json $DIRNAME/sample-configs/


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

while [ -n "$1" ]; do 
	case "$1" in
		clean)
			echo "Deleting $BIN(.exe)"
			rm -f $BIN
			rm -f $BIN.exe
			echo "Deleting release builds"
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
				github.com/mattn/go-sqlite3 \
				github.com/tdewolff/minify \
				gopkg.in/mojocn/base64Captcha.v1
				# github.com/mojocn/base64Captcha
			;;
		docker-image)
			# echo "Docker image creation not yet implemented"
			# exit 1
			docker build . -t="eggbertx/gochan"
			;;
		help|-h|--help)
			usage $@
			;;
		install)
			shift
			symarg=""
			documentroot=""
			installdir=""
			configpath=""
			while [ -n "$1" ]; do
				case "$1" in
					--symlinks|-s)
						symarg="-s"
						;;
					--document-root|--html)
						if [ -n "$2" ] && [ "$2" != "--symlinks" ] && [ "$2" != "-s" ]; then
							shift
							documentroot="$1"
						fi
						;;
					*)
						installdir="$1"
						;;
				esac
				shift
			done
			if [ "$symarg" = "-s" ]; then
				echo "Creating symlinks"
			fi
		
			if [ -n "$installdir" ]; then
				echo "Install location: '$installdir'"
				if [ -z "$documentroot" ]; then
					documentroot=$installdir/html
				fi
				
				cp $symarg -f $PWD/gochan $installdir/gochan
				cp $symarg -f $PWD/*.sql $installdir/
				cp $symarg -rf $PWD/templates $installdir/templates/

				# cp -f gochan.example.json $installdir/
				if [ -f gochan.json ]; then
					echo "Copying config file to $installdir/gochan.json"
					cp $symarg -f $PWD/gochan.json $installdir/gochan.json
				fi
				mkdir -p $installdir/log

			else
				echo "Installing gochan globally"
				if [ -z "$documentroot" ]; then
					documentroot=/srv/gochan
				fi
				cp $symarg -f $PWD/gochan /usr/local/bin/gochan
				mkdir -p /usr/local/share/gochan
				cp $symarg -f $PWD/*.sql /usr/local/share/gochan/
				cp $symarg -rf $PWD/templates /usr/local/share/gochan/templates/
				
				echo "Creating /etc/gochan/ (if it doesn't already exist)"
				mkdir -p /etc/gochan
				echo "/etc/gochan created, you should run 'cp sample-configs/gochan.example.json /etc/gochan/gochan.json'"
				# cp -f gochan.example.json /etc/gochan/
				# if [ ! -f /etc/gochan/gochan.json ] && [ -f gochan.json ]; then
				# 	echo "Copying gochan.json to /etc/gochan/gochan.json"
				# 	cp $symarg -f $PWD/gochan.json /etc/gochan/gochan.json
				# fi
				echo "Creating /var/log/gochan (if it doesn't already exist)"
				mkdir -p /var/log/gochan
			fi

			echo "Installing document root files and directories"
			mkdir -p $documentroot
			cp $symarg -rf $PWD/html/css/ $documentroot/css/
			cp $symarg -rf $PWD/html/javascript/ $documentroot/javascript/
			files=$PWD/html/*
			for f in $files; do
				if [ -f $f ]; then
					destfile=$documentroot/$(basename $f)
					echo "Installing $f to $destfile"
					cp $symarg -f $f $destfile
				fi
			done

			if [ -d /lib/systemd/system ]; then
				cat - <<-EOF
					It looks like your distribution has systemd. Gochan no longer automatically installs
					the service for you, but you can install it yourself by copying one of the following:
					sample-configs/gochan-mysql.service
					sample-configs/gochan-postgresql.service
					sample-configs/gochan-sqlite3.service
					to /lib/systemd/system/gochan.service then running the following commands
					systemctl daemon-reload
					systemctl enable gochan.service
					systemctl start gochan.service
				EOF
			fi

			echo "Installation complete. Make sure to set the following values in gochan.json:"
			echo "DocumentRoot => $documentroot"
			echo "TemplateDir => /usr/local/share/gochan/templates"

			echo "LogDir => /var/log/gochan"
			exit 0
			;;
		release)
			if [ -n "$2" ]; then
				release $2
			else
				release linux
				#release macos
				release windows
			fi
			;;
		sass)
			if [ -z `which sass` ]; then 
				echo "Sass is not installed, exiting."
				exit 1
			fi
			shift
			sassdir="html"
			if [ -n "$1" ]; then sassdir=$1; fi
			mkdir -p $sassdir
			sass --style expanded --no-source-map  sass:$sassdir/css
			;;
		*)
			build $1
			shift
			;;
	esac
	shift
done
