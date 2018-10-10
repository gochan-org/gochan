GOCHAN_VERSION=`cat version`
GOCHAN_BUILDTIME=`date +%y%m%d.%H%M`
ifeq ($(GOOS), windows)
	GOCHAN_BIN=gochan.exe
else
	GOCHAN_BIN=gochan
endif

# strips debugging info in the gochan executable
release: LDFLAGS=-w -s
release: build

# includes debugging info in the gochan executable
debug: LDFLAGS=
debug: build

build:
ifndef GOPATH
	@echo "$ GOPATH not set. Please run 'export GOPATH=\$$PWD/lib' (or wherever you prefer) and run this again."
endif
	go build -v -ldflags "${LDFLAGS} -X main.version=${GOCHAN_VERSION} -X main.buildtimeString=${GOCHAN_BUILDTIME}" -o ${DIRNAME}${GOCHAN_BIN} ./src

