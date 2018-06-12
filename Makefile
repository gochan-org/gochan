GOCHAN_RELEASE=0
GOCHAN_DEBUG=1
GOCHAN_VERBOSE=2
GOCHAN_VERBOSITY=0 # This is set by "make release/debug/verbose"

GOCHAN_VERSION=1.10.1
GOCHAN_BUILDTIME=$(shell date +%y%m%d.%H%M)
ifeq ($(GOOS), windows)
	GOCHAN_BIN=gochan.exe
else
	GOCHAN_BIN=gochan
endif

CGO_ENABLED=0
GOARCH=amd64

# If you run make without any arguments, this will be used by default. \
It doesn't give any debugging info by println and only prints major errors.
release: GOCHAN_VERBOSITY=${GOCHAN_RELEASE}
release: build

# To give warnings and stuff, run "make debug"
debug: GOCHAN_VERBOSITY=${GOCHAN_DEBUG}
debug: build

# Used in development for benchmarking and finding issues that might not be discovered by "make debug"
verbose: GOCHAN_VERBOSITY=${GOCHAN_VERBOSE}
verbose: build

build:
ifndef GOPATH
	@echo "$ GOPATH not set. Please run 'export GOPATH=\$$PWD/lib' (or wherever you prefer) and run this again."
endif
	@echo ${GOCHAN_VERBOSITY}
	@echo ${GOCHAN_VERBOSE}
	go build -v -ldflags "-w -X main.version=${GOCHAN_VERSION} -X main.buildtimeString=${GOCHAN_BUILDTIME} -X main.verbosityString=${GOCHAN_VERBOSITY}" -o ${DIRNAME}${GOCHAN_BIN} ./src
