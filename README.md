Plik
=========

Plik, aka "file" in polish, is a fully autonomous uploading system



Installation
---------------

For now, you must compile it yourself : 

##### Get the sources via go get
```sh
go get github.com/root-gg/plik/server
cd $GOPATH/src/github.com/root-gg/plik/
```

#### Install build dependencies
```sh
npm install -g grunt-cli bower
./client/build.sh env
```

##### Build it and run it
```sh
make
cd server && ./plikd
```

#### Run functional test
You have to run instance on localhost port 8080 before start test
```sh
cd test
npm install -g jasmine-node
npm install
jasmine-node ./
```
