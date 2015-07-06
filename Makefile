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

RELEASE_VERSION=`cat VERSION`
RELEASE_DIR="release/plik-$(RELEASE_VERSION)"

all: clean deps server client

###
# Install npm build dependencies
# ( run this first once )
###
deps:
	@cd server/public && npm install

###
# Build plik server for the current architecture
###
server:
	@cd server/public && bower install --allow-root
	@cd server/public && grunt
	@cd server && go get -v
	@sed -i -e "s/##VERSION##/$(RELEASE_VERSION)/g" server/common/config.go
	@cd server && go build -o plikd ./
	@sed -i -e "s/$(RELEASE_VERSION)/##VERSION##/g" server/common/config.go

###
# Build plik server for all architectures
###
servers:
	@cd server/public && bower install --allow-root
	@cd server/public && grunt
	@cd server && go get -v
	@server/build.sh servers

###
# Build plik client for the current architecture
###
client:
	@cd client && go get -v
	@cd client && go build -o plik ./

###
# Build plik client for all architectures
###
clients:
	@cd client && go get -v
	@client/build.sh clients
	@mkdir -p clients/bash && cp client/plik.sh clients/bash

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
# Build release archive
###
release: clean server clients
	@mkdir -p $(RELEASE_DIR)/server/public

	@cp -R clients $(RELEASE_DIR)

	@cp -R server/plikd $(RELEASE_DIR)/server
	@cp -R server/plikd.cfg $(RELEASE_DIR)/server

	@cp -R server/public/css $(RELEASE_DIR)/server/public
	@cp -R server/public/img $(RELEASE_DIR)/server/public
	@cp -R server/public/js $(RELEASE_DIR)/server/public
	@cp -R server/public/partials $(RELEASE_DIR)/server/public
	@cp -R server/public/public $(RELEASE_DIR)/server/public
	@cp -R server/public/index.html $(RELEASE_DIR)/server/public


###
# Build release archives for all architectures
###
releases: release servers

	@mkdir -p releases

	@cp -R servers/linux-amd64/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-linux-64bits.tar.gz *
	@cp -R servers/linux-386/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-linux-32bits.tar.gz *
	@cp -R servers/linux-arm/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-linux-arm.tar.gz *

	@cp -R servers/freebsd-amd64/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-freebsd-64bits.tar.gz *
	@cp -R servers/freebsd-386/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-freebsd-32bits.tar.gz *
	@cp -R servers/freebsd-arm/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-freebsd-arm.tar.gz *

	@cp -R servers/openbsd-amd64/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-openbsd-64bits.tar.gz *
	@cp -R servers/openbsd-386/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-openbsd-32bits.tar.gz *

	@rm $(RELEASE_DIR)/server/plikd
	@cp -R servers/windows-amd64/plikd.exe $(RELEASE_DIR)/server && cd release && zip -r ../releases/plik-`cat ../VERSION`-windows-64bits.zip .
	@cp -R servers/windows-386/plikd.exe $(RELEASE_DIR)/server && cd release && zip -r ../releases/plik-`cat ../VERSION`-windows-32bits.zip .

	@cp -R servers/darwin-amd64/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-macos-64bits.tar.gz *
	@cp -R servers/darwin-386/plikd $(RELEASE_DIR)/server && cd release && tar cvf ../releases/plik-`cat ../VERSION`-macos-32bits.tar.gz *

	@md5sum releases/* > releases/md5sum.txt


###
# Remove all build files
###
clean:
	@rm -rf server/public/bower_components
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
