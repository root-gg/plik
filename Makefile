#
## Plik global Makefile
#

RELEASE_VERSION=`cat VERSION`
RELEASE_DIR="release/plik-$(RELEASE_VERSION)"

all: clean deps build

deps:
	@cd server/public && npm install

build:
	@cd server/public && bower install
	@cd server/public && grunt
	@cd server && go get -v
	@sed -i -e "s/##VERSION##/$(RELEASE_VERSION)/g" server/common/config.go
	@cd server && go build -o plikd ./
	@sed -i -e "s/$(RELEASE_VERSION)/##VERSION##/g" server/common/config.go

clean:
	@rm -rf server/public/bower_components
	@rm -rf server/plikd
	@rm -rf clients
	@rm -rf release

clients:
	@cd client && go get -v
	@client/build.sh clients
	@mkdir -p clients/bash && cp client/plik.sh clients/bash

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

