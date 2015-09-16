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

RELEASE_VERSION="1.0.2"
RELEASE_DIR="release/plik-$(RELEASE_VERSION)"
RELEASE_TARGETS=darwin-386 darwin-amd64 freebsd-386 \
freebsd-amd64 linux-386 linux-amd64 linux-arm openbsd-386 \
openbsd-amd64 windows-386 windows-amd64

GOHOSTOS=`go env GOHOSTOS`
GOHOSTARCH=`go env GOHOSTARCH`

all: clean deps frontend server client

###
# Install npm build dependencies
# ( run this first once )
###
deps:
	@cd server/public && npm install


###
# Build frontend ressources
###
frontend:
	@if [ ! -d server/public/bower_components ]; then cd server/public && bower install --allow-root ; fi ;
	@if [ ! -d server/public/public ]; then cd server/public && grunt ; fi ;


###
# Build plik server for the current architecture
###
server:
	@sed -i -e "s/##VERSION##/$(RELEASE_VERSION)/g" server/common/config.go
	@cd server && go build -o plikd ./
	@sed -i -e "s/$(RELEASE_VERSION)/##VERSION##/g" server/common/config.go

###
# Build plik server for all architectures
###
servers:
	@cd server && for target in $(RELEASE_TARGETS) ; do \
		SERVER_DIR=../servers/$$target; \
		SERVER_PATH=$$SERVER_DIR/plikd;  \
		export GOOS=`echo $$target | cut -d "-" -f 1`; 	\
		export GOARCH=`echo $$target | cut -d "-" -f 2`; \
		mkdir -p ../servers/$$target; \
		if [ $$GOOS = "windows" ] ; then SERVER_PATH=$$SERVER_DIR/plikd.exe ; fi ; \
		echo "Compiling plik server for $$target to $$SERVER_PATH"; \
		go build -o $$SERVER_PATH ;	\
	done

###
# Build plik client for the current architecture
###
client:
	@cd client && go build -o plik ./

###
# Build plik client for all architectures
###
clients:
	@cd client && for target in $(RELEASE_TARGETS) ; do	\
		CLIENT_DIR=../clients/$$target;	\
		CLIENT_PATH=$$CLIENT_DIR/plik;	\
		export GOOS=`echo $$target | cut -d "-" -f 1`; \
		export GOARCH=`echo $$target | cut -d "-" -f 2`; \
		mkdir -p $$CLIENT_DIR; \
		if [ $$GOOS = "windows" ] ; then CLIENT_PATH=$$CLIENT_DIR/plik.exe ; fi ; \
		echo "Compiling plik client for $$target to $$CLIENT_PATH"; \
        go build -o $$CLIENT_PATH ; \
	done
	@mkdir -p clients/bash && cp client/plik.sh clients/bash

##
# Build docker
##
docker: release
	@cp Dockerfile $(RELEASE_DIR)
	@cd $(RELEASE_DIR) && docker build -t plik .

###
# Make server and clients Debian packages
###
debs: debs-client debs-server

###
# Make server Debian packages
###
debs-server: servers clients
	@server/build.sh debs

###
# Make client Debian packages
###
debs-client: clients
	@client/build.sh debs

###
# Prepare the release base (css, js, ...)
###
release-template: frontend clients
	@mkdir -p $(RELEASE_DIR)/server/public

	@cp -R clients $(RELEASE_DIR)
	@cp -R server/plikd.cfg $(RELEASE_DIR)/server
	@cp -R server/public/css $(RELEASE_DIR)/server/public
	@cp -R server/public/img $(RELEASE_DIR)/server/public
	@cp -R server/public/js $(RELEASE_DIR)/server/public
	@cp -R server/public/partials $(RELEASE_DIR)/server/public
	@cp -R server/public/public $(RELEASE_DIR)/server/public
	@cp -R server/public/index.html $(RELEASE_DIR)/server/public


###
# Build release archive
###
release: release-template server
	@cp -R server/plikd $(RELEASE_DIR)/server
	@cd $(RELEASE_DIR) && tar czvf ../plik-$(RELEASE_VERSION)-$(GOHOSTOS)-$(GOHOSTARCH).tar.gz *


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
		TARBALL_NAME=plik-$(RELEASE_VERSION)-$$OS-$$ARCH.tar.gz; \
		echo "Packaging plik release for $$target to $$TARBALL_NAME"; \
		cp -R $$SERVER_PATH plik-$(RELEASE_VERSION)/server; \
		tar czvf ../releases/$$TARBALL_NAME plik-$(RELEASE_VERSION); \
	done

	@md5sum releases/* > releases/md5sum.txt


###
# Remove all build files
###
clean:
	@rm -rf server/public/bower_components
	@rm -rf server/public/public
	@rm -rf server/plikd
	@rm -rf client/plik
	@rm -rf clients
	@rm -rf servers
	@rm -rf debs
	@rm -rf release
	@rm -rf releases


###
# Since the client/server directories are not generated
# by make, we must declare these targets as phony to avoid :
# "make: `client' is up to date" cases at compile time
###
.PHONY: client server
