RELEASE_FILE = plik-`cat ../VERSION`.tar.gz
DEST_DIR = /opt/plik

all: clean deps build

install-devtools:
	@npm install -g grunt-cli bower
	@client/build.sh env

deps:
	@cd server/public && npm install
	@cd server/public && bower install --allow-root

build:
	@cd server/public && grunt
	@cd server && go get -v
	@cd server && go build -o plikd ./

clean:
	@rm -rf server/public/bower_components
	@rm -rf server/plikd
	@rm -rf clients
	@rm -rf release

clients:
	@cd client && go get -v
	@client/build.sh clients

release: clean deps build clients
	@mkdir release
	@mkdir -p release/server/public

	@cp -R clients release

	@cp -R server/plikd release/server
	@cp -R server/plikd.cfg release/server

	@cp -R server/public/css release/server/public
	@cp -R server/public/img release/server/public
	@cp -R server/public/js release/server/public
	@cp -R server/public/partials release/server/public
	@cp -R server/public/public release/server/public
	@cp -R server/public/index.html release/server/public

	@cd release && tar cvf $(RELEASE_FILE) .

install: clean deps build clients
	@mkdir -p $(DEST_DIR)/server/public

	@cp -R clients $(DEST_DIR)

	@cp -R server/plikd $(DEST_DIR)/server
	@cp -R server/plikd.cfg $(DEST_DIR)/server

	@cp -R server/public/css $(DEST_DIR)/server/public
	@cp -R server/public/img $(DEST_DIR)/server/public
	@cp -R server/public/js $(DEST_DIR)/server/public
	@cp -R server/public/partials $(DEST_DIR)/server/public
	@cp -R server/public/public $(DEST_DIR)/server/public
	@cp -R server/public/index.html $(DEST_DIR)/server/public


