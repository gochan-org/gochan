GOCHAN_RELEASE=0
GOCHAN_DEBUG=1
GOCHAN_VERBOSE=2
GOCHAN_VERBOSITY=0

GOCHAN_VERSION=1.2
GOCHAN_BUILDTIME=$(shell date +%y%m%d.%H%M)
GOCHAN_EXT=""

CGO_ENABLED=0
GOARCH=amd64

release: GOCHAN_VERBOSITY=${GOCHAN_RELEASE}
release: build

debug: GOCHAN_VERBOSITY=${GOCHAN_DEBUG}
debug: build

verbose: GOCHAN_VERBOSITY=${GOCHAN_VERBOSE}
verbose: build

build:
ifndef GOPATH
	@echo "$ GOPATH not set. Please run 'export GOPATH=\$$PWD/lib' (or wherever you prefer) and run this again."
endif
ifeq (GOOS, "windows")
	GOCHAN_EXT=".exe"
endif
	go build -v  -ldflags "-w -X main.version=${GOCHAN_VERSION} -X main.buildtime_str=${GOCHAN_BUILDTIME} -X main.verbosity_str=${GOCHAN_VERBOSITY}" -o gochan${SUFFIX} ./src
