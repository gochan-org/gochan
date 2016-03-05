GOCHAN_VERBOSE=0
GOCHAN_VERSION=0.9
GOCHAN_BUILDTIME=$(shell date +%y%m%d.%H%m)
GOCHAN_EXT=""

CGO_ENABLED=0
GOARCH=amd64


all:

ifndef GOPATH
	@echo ${date +%y%m%d.%H%m}
	@echo "$ GOPATH not set. Please run 'export GOPATH=\$$PWD/lib' (or wherever you prefer) and run this again."
endif
ifeq (GOOS, "windows")
	GOCHAN_EXT=".exe"
endif
	go build -v  -ldflags "-w -X main.version=${GOCHAN_VERSION} -X main.buildtime_str=${GOCHAN_BUILDTIME} -X main.verbose_str=${GOCHAN_VERBOSE}" -o gochan${SUFFIX} ./src
