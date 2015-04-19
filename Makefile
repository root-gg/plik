all: clean deps build

deps:
	@cd server/public && npm install

build:
	@cd server/public && bower install
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

release: clean build clients
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

	@cd release && tar cvf plik-`cat ../VERSION`.tar.gz .