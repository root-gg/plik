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

all: clean deps build

deps:
	@cd server/public && npm install

build:
	@cd server/public && bower install --allow-root
	@cd server/public && grunt
	@cd server && go get -v
	@sed -i -e "s/##VERSION##/$(RELEASE_VERSION)/g" server/common/config.go
	@cd server && go build -o plikd ./
	@sed -i -e "s/$(RELEASE_VERSION)/##VERSION##/g" server/common/config.go

clean:
	@rm -rf server/public/bower_components
	@rm -rf server/plikd
	@rm -rf clients
	@rm -rf servers
	@rm -rf debs
	@rm -rf release

clients:
	@cd client && go get -v
	@client/build.sh clients
	@mkdir -p clients/bash && cp client/plik.sh clients/bash

servers:
	@cd server && go get -v
	@server/build.sh servers

debs: clients servers debs-client debs-server

debs-client:
	@client/build.sh debs

debs-server:
	@server/build.sh debs

release: clean build clients
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

	@cd release && tar cvf plik-`cat ../VERSION`.tar.gz *

