#!/usr/bin/env pwsh
param (
	[string]$action = "build",
	[bool]$minify = $TRUE,
	[string]$platform,
	[string]$releaseFor
)
$ErrorActionPreference = "Stop"

if($releaseFor -ne "*" -And $releaseFor -ne "") {
	$platform = $releaseFor
}

$BIN = "gochan"
$BINEXE = "gochan.exe"
if( $releaseFor -ne "*" ) {
	$BINEXE = "$BIN$($env:GOOS=$platform; go env GOEXE)"
}
$GCOS = go env GOOS
$GCOS_NAME = $GCOS
if ($GCOS_NAME -eq "darwin") {
	$GCOS_NAME = "macos"
}
$VERSION = Get-Content version
$RELEASE_NAME = "$BIN-v${VERSION}_$GCOS_NAME"
$RELEASE_DIR = "releases/${RELEASE_NAME}"

$LDFLAGS = "-X main.versionStr=${VERSION} -s"
$DOCUMENT_ROOT_FILES = @"
banned.jpg
notbanned.png
permabanned.jpg
favicon*
firstrun.html
hittheroad*
"@ -split "`n"

for ($l = 0; $l -lt $DOCUMENT_ROOT_FILES.Count; $l++) {
	$line = "html/" + $DOCUMENT_ROOT_FILES[$l]
	$DOCUMENT_ROOT_FILES[$l] = Get-ChildItem $line
}
$DOCUMENT_ROOT_FILES += @"
html/css
html/error
html/javascript
"@

function build {
	$cmd = "& go build -v -gcflags=-trimpath=$PWD -asmflags=-trimpath=$PWD -ldflags=`"$LDFLAGS`" -o $BINEXE ./src "
	$env:GOOS=$platform; Invoke-Expression $cmd
}

function clean {
	Remove-Item $BIN*
	Remove-Item releases/* -Force -Recurse
}

function dependencies {
	go get -v `
		github.com/disintegration/imaging `
		github.com/nranchev/go-libGeoIP `
		github.com/go-sql-driver/mysql `
		github.com/lib/pq `
		golang.org/x/net/html `
		github.com/aquilax/tripcode `
		golang.org/x/crypto/bcrypt `
		github.com/frustra/bbcode `
		github.com/mattn/go-sqlite3 `
		github.com/tdewolff/minify `
		gopkg.in/mojocn/base64Captcha.v1
}

function dockerImage {
	throw "Docker image creation not yet implemented"
	# docker build . -t="eggbertx/gochan"
}

function release {
	clean
	mkdir -p `
		$RELEASE_DIR/html `
		$RELEASE_DIR/log `

		cp LICENSE $RELEASE_DIR
		cp README.md $RELEASE_DIR
		cp -r sample-configs $RELEASE_DIR
		cp -r templates $RELEASE_DIR
	
	foreach($line in $DOCUMENT_ROOT_FILES) {
		$arr = $line.Split()
		foreach($word in $arr) {
			Copy-Item -Recurse $word $RELEASE_DIR/html
		}
	}

	$env:GOOS=$platform; ./build.ps1
}

function doSass {
	if($minify) {
		sass --style compressed --no-source-map sass:html/css
	} else {
		sass --no-source-map sass:html/css
	}
}

switch ($action) {
	"build" { 
		build
	}
	"clean" {
		clean
	}
	"dependencies" {
		dependencies
	}
	"js" {
		throw "Frontend transpilation coming soon"
	}
	"release" {
		if($releaseFor -eq "*") {
			./build.ps1 -action release -platform darwin -releaseFor darwin
			./build.ps1 -action release -platform linux -releaseFor linux
			./build.ps1 -action release -platform windows -releaseFor windows
		} else {
			release
		}
	}
	"sass" {
		doSass
	}
	"test" {
		go test -v ./src
	}
	Default {
		Write-Output "Invalid or unsupported command"
	}
}
