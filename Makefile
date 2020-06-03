BIN=gochan
BINEXE=$(BIN)$(shell go env GOEXE)
GCOS=$(shell go env GOOS)
GCOS_NAME=${GCOS}
ifeq (${GCOS_NAME},darwin)
	GCOS_NAME=macos
endif

GOCHAN_PKG=github.com/gochan-org/gochan
DOCUMENT_ROOT=/srv/gochan
RELEASE_NAME=${BIN}-v${VERSION}_${GCOS_NAME}
RELEASE_DIR=releases/${RELEASE_NAME}
PREFIX=/usr/local
VERSION=$(shell cat version)

GCFLAGS=-trimpath=${PWD}
ASMFLAGS=-trimpath=${PWD}
LDFLAGS=-X main.versionStr=${VERSION}
GO_CMD=go build -o ${BINEXE} -v 
NPM_CMD=npm --prefix frontend/ run 

DOCUMENT_ROOT_FILES= \
	css \
	error \
	javascript \
	banned.jpg \
	notbanned.png \
	permabanned.jpg \
	favicon* \
	firstrun.html \
	hittheroad*

build:
	GOOS=${GCOS} ${GO_CMD} -gcflags=${GCFLAGS} -asmflags=${ASMFLAGS} -ldflags="${LDFLAGS} -w -s" ./cmd/gochan

build-debug:
	GOOS=${GCOS} ${GO_CMD} -gcflags="${GCFLAGS} -l -N" -asmflags=${ASMFLAGS} -ldflags="${LDFLAGS}" ./cmd/gochan

clean:
	rm -f ${BIN}
	rm -f ${BIN}.exe
	rm -rf releases/
	rm -rf ${GOPATH}/src/${GOCHAN_PKG}

dependencies:
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
		github.com/mojocn/base64Captcha

install:
	mkdir -p \
		${PREFIX}/share/gochan \
		${DOCUMENT_ROOT} \
		/etc/gochan \
		/var/log/gochan
	cp ${DO_SYMLINKS} -f ./gochan ${PREFIX}/bin/${BINEXE}
	cp ${DO_SYMLINKS} -f ./*.sql ${PREFIX}/share/gochan/
	cp ${DO_SYMLINKS} -rf ./templates ${PREFIX}/share/gochan/
	cd html $(foreach file,${DOCUMENT_ROOT_FILES}, && cp -rf ${file} ${DOCUMENT_ROOT})
	$(info gochan successfully installed. If you haven't already, you should run)
	$(info cp sample-configs/gochan.example.json /etc/gochan/gochan.json)
ifeq (${GCOS_NAME},linux)
	$(info If your distro has systemd, you will also need to run the following commands)
	$(info cp sample-configs/gochan-[mysql|postgresql|sqlite3].service /lib/systemd/system/gochan.service)
	$(info systemctl daemon-reload)
	$(info systemctl enable gochan.service)
	$(info systemctl start gochan.service)
endif

install-symlinks:
	DO_SYMLINKS=-s make install

js:
	${NPM_CMD} build

js-minify:
	${NPM_CMD} build-minify

js-watch:
	${NPM_CMD} build-watch

js-minify-watch:
	${NPM_CMD} build-minify-watch

release-all: 
	GOOS=darwin make release
	GOOS=linux make release
	GOOS=windows make release

release:
	mkdir -p \
		${RELEASE_DIR}/html \
		${RELEASE_DIR}/log \
		${RELEASE_DIR}/sample-configs
	cp LICENSE ${RELEASE_DIR}/
	cp README.md ${RELEASE_DIR}/
	# make js-minify
	cp -rt ${RELEASE_DIR}/html/ $(foreach file,${DOCUMENT_ROOT_FILES},html/${file})
	cp -r docker ${RELEASE_DIR}/
	cp -r sass ${RELEASE_DIR}/
	cp -r templates ${RELEASE_DIR}/
	cp initdb_*.sql ${RELEASE_DIR}/
	cp sample-configs/*.nginx ${RELEASE_DIR}/sample-configs/
	cp sample-configs/gochan.example.json ${RELEASE_DIR}/sample-configs/
	make build
	make sass-minified
	mv ${BINEXE} ${RELEASE_DIR}/
ifeq (${GCOS_NAME},macos)
	cd releases && zip -r ${RELEASE_NAME}.zip ${RELEASE_NAME}
else ifeq (${GCOS_NAME},windows)
	cd releases && zip -r ${RELEASE_NAME}.zip ${RELEASE_NAME}
else
	cp sample-configs/gochan-mysql.service ${RELEASE_DIR}/sample-configs/
	cp sample-configs/gochan-postgresql.service ${RELEASE_DIR}/sample-configs/
	cp sample-configs/gochan-sqlite3.service ${RELEASE_DIR}/sample-configs/
	tar -C releases -zcvf ${RELEASE_DIR}.tar.gz ${RELEASE_NAME}
endif

sass:
	sass --no-source-map sass:html/css

sass-minified:
	sass --style compressed --no-source-map sass:html/css

test:
	go test -v ./src

.PHONY: subpackages ${INTERNALS} sass
