###
# The MIT License (MIT)
#
# Copyright (c) <2015>
# - Mathieu Bodjikian <mathieu@bodjikian.fr>
# - Charles-Antoine Mathieu <skatkatt@root.gg>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.
###

RELEASE_VERSION="1.2"
RELEASE_DIR="release/plik-$(RELEASE_VERSION)"
RELEASE_TARGETS=darwin-386 darwin-amd64 freebsd-386 \
freebsd-amd64 linux-386 linux-amd64 linux-arm openbsd-386 \
openbsd-amd64 windows-amd64 windows-386

GOHOSTOS=`go env GOHOSTOS`
GOHOSTARCH=`go env GOHOSTARCH`

DEBROOT_SERVER=debs/server
DEBROOT_CLIENT=debs/client

all: clean clean-frontend frontend clients server

###
# Build frontend ressources
###
frontend:
	@if [ ! -d server/public/node_modules ]; then cd server/public && npm install ; fi
	@if [ ! -d server/public/bower_components ]; then cd server/public && node_modules/bower/bin/bower install --allow-root ; fi
	@if [ ! -d server/public/public ]; then cd server/public && node_modules/grunt-cli/bin/grunt ; fi


###
# Build plik server for the current architecture
###
server:
	@server/gen_build_info.sh $(RELEASE_VERSION)
	@cd server && go build -o plikd ./

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
		echo "Compiling plik server for $$target to $$SERVER_PATH"; \
		go build -o $$SERVER_PATH ;	\
	done


###
# Build plik utils for all architectures
###
utils: servers
	@cd utils && for util in `ls *.go` ; do \
        for target in $(RELEASE_TARGETS) ; do \
            UTIL_DIR=../servers/$$target/utils; \
            UTIL_BASE=`basename $$util .go`; \
            UTIL_PATH=$$UTIL_DIR/$$UTIL_BASE;  \
            mkdir -p $$UTIL_DIR;  \
            export GOOS=`echo $$target | cut -d "-" -f 1`; 	\
            if [ $$GOOS = "windows" ] ; then UTIL_PATH=$$UTIL_DIR/$$UTIL_BASE.exe ; fi ; \
            if [ -e $$UTIL_PATH ] ; then continue ; fi ; \
            echo "Compiling plik util file2bolt for $$target to $$UTIL_PATH"; \
            go build -o $$UTIL_PATH $$util ; \
        done ; \
	done


###
# Build plik client for the current architecture
###
client:
	@server/gen_build_info.sh $(RELEASE_VERSION)
	@cd client && go build -o plik ./

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
		echo "Compiling plik client for $$target to $$CLIENT_PATH"; \
		go build -o $$CLIENT_PATH ; \
		md5sum $$CLIENT_PATH | awk '{print $$1}' > $$CLIENT_MD5; \
	done
	@mkdir -p clients/bash && cp client/plik.sh clients/bash

##
# Build docker
##
docker: release
	@cp Dockerfile $(RELEASE_DIR)
	@cd $(RELEASE_DIR) && docker build -t rootgg/plik .

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
	@mkdir -p $(RELEASE_DIR)/server/public
	@mkdir -p $(RELEASE_DIR)/server/utils

	@cp -R clients $(RELEASE_DIR)
	@cp -R server/plikd.cfg $(RELEASE_DIR)/server
	@cp -R server/public/css $(RELEASE_DIR)/server/public
	@cp -R server/public/fonts $(RELEASE_DIR)/server/public
	@cp -R server/public/img $(RELEASE_DIR)/server/public
	@cp -R server/public/js $(RELEASE_DIR)/server/public
	@cp -R server/public/partials $(RELEASE_DIR)/server/public
	@cp -R server/public/public $(RELEASE_DIR)/server/public
	@cp -R server/public/index.html $(RELEASE_DIR)/server/public
	@cp -R server/public/favicon.ico $(RELEASE_DIR)/server/public


###
# Build release archive
###
release: release-template server
	@cp -R server/plikd $(RELEASE_DIR)/server
	@cd release && tar czvf plik-$(RELEASE_VERSION)-$(GOHOSTOS)-$(GOHOSTARCH).tar.gz plik-$(RELEASE_VERSION)


###
# Build release archives for all architectures
###
releases: release-template servers utils

	@mkdir -p releases

	@cd release && for target in $(RELEASE_TARGETS) ; do \
		SERVER_PATH=../servers/$$target/plikd;  \
		UTIL_DIR=../servers/$$target/utils;  \
		OS=`echo $$target | cut -d "-" -f 1`; \
		ARCH=`echo $$target | cut -d "-" -f 2`; \
		if [ $$OS = "darwin" ] ; then OS="macos" ; fi ; \
		if [ $$OS = "windows" ] ; then SERVER_PATH=../servers/$$target/plikd.exe ; fi ; \
		if [ $$ARCH = "386" ] ; then ARCH="32bits" ; fi ; \
		if [ $$ARCH = "amd64" ] ; then ARCH="64bits" ; fi ; \
		cp -R $$SERVER_PATH plik-$(RELEASE_VERSION)/server; \
		cp -R $$UTIL_DIR plik-$(RELEASE_VERSION)/server; \
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
# Run tests and sanity checks
###
test:

	@server/gen_build_info.sh $(RELEASE_VERSION)
	@ERR="" ; for directory in server client ; do \
		cd $$directory; \
		echo -n "go test $$directory : "; \
		TEST=`go test ./... 2>&1`; \
		if [ $$? = 0 ] ; then echo "OK" ; else echo "$$TEST" | grep -v "no test files" | grep -v "^\[" && ERR="1"; fi ; \
		echo "go fmt $$directory : "; \
		for file in $$(find -name "*.go" | grep -v Godeps ); do \
			echo -n " - file $$file : " ; \
			FMT=`gofmt -l $$file` ; \
			if [ "$$FMT" = "" ] ; then echo "OK" ; else echo "FAIL" && ERR="1" ; fi ; \
		done; \
		echo -n "go vet $$directory : "; \
		VET=`go vet ./... 2>&1`; \
		if [ $$? = 0 ] ; then echo "OK" ; else echo "FAIL" && echo "$$VET" && ERR="1" ; fi ; \
		echo -n "go lint $$directory : "; \
		LINT=`golint ./...`; \
		if [ "$$LINT" = "" ] ; then echo "OK" ; else echo "FAIL" && echo "$$LINT" && ERR="1" ; fi ; \
		cd - 2>&1 > /dev/null; \
	done ; if [ "$$ERR" = "1" ] ; then exit 1 ; fi
	@echo "cli client integration tests :\n" && cd client && ./test.sh

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
	@rm -rf server/public/bower_components
	@rm -rf server/public/public

###
# Remove all build files and node modules
###
clean-all: clean clean-frontend
	@rm -rf server/public/node_modules

###
# Since the client/server directories are not generated
# by make, we must declare these targets as phony to avoid :
# "make: `client' is up to date" cases at compile time
###
.PHONY: client server
