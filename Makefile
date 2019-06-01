GOCHAN_VERSION=`cat version`
GOCHAN_BUILDTIME=`date +%y%m%d.%H%M`

GCFLAGS=-trimpath=${PWD}
ASMFLAGS=-trimpath=${PWD}
LDFLAGS="-X main.versionStr=${GOCHAN_VERSION} -X main.buildtimeString=${GOCHAN_BUILDTIME}" 

ifeq ($(GOOS), windows)
	GOCHAN_BIN=gochan.exe
else
	GOCHAN_BIN=gochan
endif

# strips debugging info in the gochan executable
release: LDFLAGS=${LDFLAGS} -w -s
release: build

# includes debugging info in the gochan executable
debug: build

build:
ifndef GOPATH
	$(error GOPATH not set. Please run 'export GOPATH=$$PWD/lib' (or wherever you prefer) and run this again.)
endif
	go build -v -gcflags=${GCFLAGS} -asmflags=${ASMFLAGS} -ldflags ${LDFLAGS} -o ${DIRNAME}${GOCHAN_BIN} ./src


# Don't use this, it doesn't work yet
docker-image:
	docker build . -t="eggbertx/gochan"
