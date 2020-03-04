SHELL = /bin/bash

RELEASE_VERSION=$(shell version/version.sh)
RELEASE_DIR="release/plik-$(RELEASE_VERSION)"
RELEASE_TARGETS=darwin-386 darwin-amd64 freebsd-386 \
freebsd-amd64 linux-386 linux-amd64 linux-arm openbsd-386 \
openbsd-amd64 windows-amd64 windows-386

GOHOSTOS=$(shell go env GOHOSTOS)
GOHOSTARCH=$(shell go env GOHOSTARCH)

DEBROOT_SERVER=debs/server
DEBROOT_CLIENT=debs/client

race_detector = GORACE="halt_on_error=1" go build -race
ifdef ENABLE_RACE_DETECTOR
	build = $(race_detector)
else
	build = go build
endif
test: build = $(race_detector)

all: clean clean-frontend frontend clients server

###
# Build frontend ressources
###
frontend:
	@if [ ! -d webapp/node_modules ]; then cd webapp && npm install ; fi
	@if [ ! -d webapp/bower_components ]; then cd webapp && node_modules/bower/bin/bower install --allow-root ; fi
	@cd webapp && node_modules/grunt-cli/bin/grunt

###
# Build plik server for the current architecture
###
server:
	@server/gen_build_info.sh $(RELEASE_VERSION)
	@echo "Building Plik server"
	@cd server && $(build) -o plikd ./

###
# Build plik server for all architectures
###
servers: frontend
	@server/gen_build_info.sh $(RELEASE_VERSION)
	@cd server && for target in $(RELEASE_TARGETS) ; do \
		SERVER_DIR=../servers/$$target; \
		SERVER_PATH=$$SERVER_DIR/plikd;  \
		export GOOS=`echo $$target | cut -d "-" -f 1`; 	\
		export GOARCH=`echo $$target | cut -d "-" -f 2`; \
		mkdir -p ../servers/$$target; \
		if [ $$GOOS = "windows" ] ; then SERVER_PATH=$$SERVER_DIR/plikd.exe ; fi ; \
		if [ -e $$SERVER_PATH ] ; then continue ; fi ; \
		echo "Building Plik server for $$target to $$SERVER_PATH"; \
		$(build) -o $$SERVER_PATH ;	\
	done

###
# Build plik client for the current architecture
###
client:
	@server/gen_build_info.sh $(RELEASE_VERSION)
	@echo "Building Plik client"
	@cd client && $(build) -o plik ./


###
# Build plik client for all architectures
###
clients:
	@server/gen_build_info.sh $(RELEASE_VERSION)
	@cd client && for target in $(RELEASE_TARGETS) ; do	\
		CLIENT_DIR=../clients/$$target;	\
		CLIENT_PATH=$$CLIENT_DIR/plik;	\
		CLIENT_MD5=$$CLIENT_DIR/MD5SUM;	\
		export GOOS=`echo $$target | cut -d "-" -f 1`; \
		export GOARCH=`echo $$target | cut -d "-" -f 2`; \
		mkdir -p $$CLIENT_DIR; \
		if [ $$GOOS = "windows" ] ; then CLIENT_PATH=$$CLIENT_DIR/plik.exe ; fi ; \
		if [ -e $$CLIENT_PATH ] ; then continue ; fi ; \
		echo "Building Plik client for $$target to $$CLIENT_PATH"; \
		$(build) -o $$CLIENT_PATH ; \
		md5sum $$CLIENT_PATH | awk '{print $$1}' > $$CLIENT_MD5; \
	done
	@mkdir -p clients/bash && cp client/plik.sh clients/bash

###
# Make server and clients Debian packages
###
debs: debs-client debs-server

###
# Make server Debian packages
###
debs-server: servers clients
	@mkdir -p $(DEBROOT_SERVER)/usr/local/plikd/server
	@mkdir -p $(DEBROOT_SERVER)/etc/init.d
	@cp -R server/build/deb/DEBIAN $(DEBROOT_SERVER)
	@cp -R clients/ $(DEBROOT_SERVER)/usr/local/plikd/clients
	@cp -R server/public/ $(DEBROOT_SERVER)/usr/local/plikd/server/public
	@cp -R server/plikd.cfg $(DEBROOT_SERVER)/etc/plikd.cfg
	@cp -R server/plikd.init $(DEBROOT_SERVER)/etc/init.d/plikd && chmod +x $(DEBROOT_SERVER)/etc/init.d/plikd
	@for arch in amd64 i386 armhf ; do \
		cp -R server/build/deb/DEBIAN/control $(DEBROOT_SERVER)/DEBIAN/control ; \
		sed -i -e "s/##ARCH##/$$arch/g" $(DEBROOT_SERVER)/DEBIAN/control ; \
		sed -i -e "s/##VERSION##/$(RELEASE_VERSION)/g" $(DEBROOT_SERVER)/DEBIAN/control ; \
		if [ $$arch = 'i386' ]; then \
			cp servers/linux-386/plikd $(DEBROOT_SERVER)/usr/local/plikd/server/ ; \
		elif [ $$arch = 'armhf' ]; then  \
			cp servers/linux-arm/plikd $(DEBROOT_SERVER)/usr/local/plikd/server/ ; \
		else \
			cp servers/linux-$$arch/plikd $(DEBROOT_SERVER)/usr/local/plikd/server/ ; \
		fi ; \
		dpkg-deb --build $(DEBROOT_SERVER) debs/plikd-$(RELEASE_VERSION)-$$arch.deb ; \
	done

###
# Make client Debian packages
###
debs-client: clients
	@mkdir -p $(DEBROOT_CLIENT)/usr/local/bin
	@cp -R client/build/deb/DEBIAN $(DEBROOT_CLIENT)
	@for arch in amd64 i386 armhf ; do \
		cp -R client/build/deb/DEBIAN/control $(DEBROOT_CLIENT)/DEBIAN/control ; \
		sed -i -e "s/##ARCH##/$$arch/g" $(DEBROOT_CLIENT)/DEBIAN/control ; \
		sed -i -e "s/##VERSION##/$(RELEASE_VERSION)/g" $(DEBROOT_CLIENT)/DEBIAN/control ; \
		if [ $$arch = 'i386' ]; then \
			cp clients/linux-386/plik $(DEBROOT_CLIENT)/usr/local/bin ; \
		elif [ $$arch = 'armhf' ]; then  \
			cp clients/linux-arm/plik $(DEBROOT_CLIENT)/usr/local/bin ; \
		else \
			cp clients/linux-$$arch/plik $(DEBROOT_CLIENT)/usr/local/bin ; \
		fi ; \
		dpkg-deb --build $(DEBROOT_CLIENT) debs/plik-$(RELEASE_VERSION)-$$arch.deb ; \
	done

###
# Prepare the release base (css, js, ...)
###
release-template: clean frontend clients
	@mkdir -p $(RELEASE_DIR)/webapp
	@mkdir -p $(RELEASE_DIR)/server
	@cp -r clients $(RELEASE_DIR)
	@cp -r changelog $(RELEASE_DIR)
	@cp -r webapp/dist $(RELEASE_DIR)/webapp/dist
	@cp -r server/plikd.cfg $(RELEASE_DIR)/server

###
# Build release archive
###
release: release-template server
	@cp -R server/plikd $(RELEASE_DIR)/server/plikd
	@cd release && tar czvf plik-$(RELEASE_VERSION)-$(GOHOSTOS)-$(GOHOSTARCH).tar.gz plik-$(RELEASE_VERSION)

###
# Build release archives for all architectures
###
releases: release-template servers

	@mkdir -p releases

	@cd release && for target in $(RELEASE_TARGETS) ; do \
		SERVER_PATH=../servers/$$target/plikd;  \
		OS=`echo $$target | cut -d "-" -f 1`; \
		ARCH=`echo $$target | cut -d "-" -f 2`; \
		if [ $$OS = "darwin" ] ; then OS="macos" ; fi ; \
		if [ $$OS = "windows" ] ; then SERVER_PATH=../servers/$$target/plikd.exe ; fi ; \
		if [ $$ARCH = "386" ] ; then ARCH="32bits" ; fi ; \
		if [ $$ARCH = "amd64" ] ; then ARCH="64bits" ; fi ; \
		cp -R $$SERVER_PATH plik-$(RELEASE_VERSION)/server; \
		if [ $$OS = "windows" ] ; then \
			TARBALL_NAME=plik-$(RELEASE_VERSION)-$$OS-$$ARCH.zip; \
			echo "Packaging plik release for $$target to $$TARBALL_NAME"; \
			zip -r ../releases/$$TARBALL_NAME plik-$(RELEASE_VERSION); \
		else \
			TARBALL_NAME=plik-$(RELEASE_VERSION)-$$OS-$$ARCH.tar.gz; \
			echo "Packaging plik release for $$target to $$TARBALL_NAME"; \
			tar czvf ../releases/$$TARBALL_NAME plik-$(RELEASE_VERSION); \
		fi \
	done

	@md5sum releases/* > releases/md5sum.txt

###
# Generate build info
###
build-info:
	@server/gen_build_info.sh $(RELEASE_VERSION)

###
# Run linters
###
lint:
	@FAIL=0 ;echo -n " - go fmt :" ; OUT=`gofmt -l . | grep -v ^vendor` ; \
	if [[ -z "$$OUT" ]]; then echo " OK" ; else echo " FAIL"; echo "$$OUT"; FAIL=1 ; fi ;\
	echo -n " - go vet :" ; OUT=`go vet ./...` ; \
	if [[ -z "$$OUT" ]]; then echo " OK" ; else echo " FAIL"; echo "$$OUT"; FAIL=1 ; fi ;\
	echo -n " - go lint :" ; OUT=`golint ./... | grep -v ^vendor` ; \
	if [[ -z "$$OUT" ]]; then echo " OK" ; else echo " FAIL"; echo "$$OUT"; FAIL=1 ; fi ;\
	test $$FAIL -eq 0

###
# Run fmt
###
fmt:
	@goimports -w -l -local "github.com/root-gg/plik" $(shell find . -type f -name '*.go' -not -path "./vendor/*")

###
# Run tests
###
test:
	@if curl -s 127.0.0.1:8080 > /dev/null ; then echo "Plik server probably already running" && exit 1 ; fi
	@server/gen_build_info.sh $(RELEASE_VERSION)
	@GORACE="halt_on_error=1" go test -race -cover -p 1 ./... 2>&1 | grep -v "no test files"; test $${PIPESTATUS[0]} -eq 0
	@echo "cli client integration tests :" && cd client && ./test.sh

###
# Run integration tests for all available backends
###
test-backends:
	@testing/test_backends.sh

###
# Build docker
###
docker: release
	@cp Dockerfile $(RELEASE_DIR)
	@cd $(RELEASE_DIR) && docker build -t rootgg/plik .

###
# Remove server build files
###
clean:
	@rm -rf server/common/version.go
	@rm -rf server/plikd
	@rm -rf client/plik
	@rm -rf clients
	@rm -rf servers
	@rm -rf debs
	@rm -rf release
	@rm -rf releases

###
# Remove frontend build files
###
clean-frontend:
	@rm -rf webapp/bower_components
	@rm -rf webapp/dist

###
# Remove all build files and node modules
###
clean-all: clean clean-frontend
	@rm -rf webapp/node_modules

###
# Since the client/server/version directories are not generated
# by make, we must declare these targets as phony to avoid :
# "make: `client' is up to date" cases at compile time
###
.PHONY: client server