Plik
=========

Plik, aka "file" in polish, is a fully autonomous uploading system



Installation
---------------

For now, you must compile it yourself : 

##### Clone the repository
```sh
git clone git@git.spinoff.ovh.net:plik/plik.git
cd plik/
```

#### Install build dependencies
```sh
npm install -g grunt-cli bower
./client/build.sh env
```

##### Build it and run it
```sh
make
./plikd
```

#### Run functional test
You have to run instance on localhost port 8080 before start test
```sh
cd test
npm install -g jasmine-node
npm install
jasmine-node ./
```
