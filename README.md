[![Build Status](https://travis-ci.org/root-gg/plik.svg?branch=master)](https://travis-ci.org/root-gg/plik)
[![Go Report](https://img.shields.io/badge/Go_report-A+-brightgreen.svg)](http://goreportcard.com/report/root-gg/plik)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT)

# Plik

Plik is an simple and powerful file uploading system written in golang.

### Main features
   - Multiple data backends : File, OpenStack Swift, WeedFS
   - Multiple metadata backends : File, MongoDB
   - Shorten backends : Recuce your uploads urls (is.gd && w000t.me available)
   - OneShot : Files are destructed after first download
   - Removable : Give the hability to uploader to remove files from upload
   - TTL : Option to set upload expiration
   - Password : Protect the upload with login/password (Auth Basic)
   - Comments : Add comments to upload (in Markdown format)
   - Yubikey : Protect the upload with your yubikey. You'll need an OTP per download

### Version
1.0-RC4


### Installation

##### From release
To run plik, it's very simple :
```sh
$ wget https://github.com/root-gg/plik/releases/download/1.0-RC4/plik-1.0-RC4.tar.gz
$ tar xvf plik-1.0-RC4.tar.gz
$ cd plik-1.0-RC4/server
$ ./plikd
```
Et voilà ! You have how a fully functionnal instance of plik ruuning on http://127.0.0.1:8080. You can edit server/plikd.cfg to adapt the params to your needs (ports, ssl, ttl, backends params,...)

##### From sources
For compiling plik from sources, you need a functionnal installation of Golang, and npm installed on your system.

First, get the project and libs via go get
```sh
$ go get github.com/root-gg/plik/server
$ cd $GOPATH/github.com/root-gg/plik/
```

As root user you need to install grunt, bower, and setup the golang crosscompilation environnement :
```sh
$ sudo -c "npm install -g bower grunt-cli"
$ sudo -c "client/build.sh env"
```

And now, you just have to compile it
```sh
$ make build
$ make clients
```

### API
Plik server expose a REST-full API to manage uploads and get files :

Creating upload and uploading files :
 
   - **POST**        /upload
     - Params (json object in request body) :
      - oneshot (bool)
      - removable (bool)
      - ttl (int)
      - login (string)
      - password (string)
     - Return :
         JSON formatted upload object.
         Important fields :
           - id (required to upload files)
           - uploadToken (required to upload files)

   - **GET** /upload/:uploadid:
     - Get upload metadatas (files list, upload date, ttl,...)

   - **POST** /upload/:uploadid:/file
     - Body must be a multipart request with a part named "file" containing file data
   Returning a JSON object of newly uploaded file
   
   - **DELETE** /upload/:uploadid:/file/:fileid:
     - Delete file from the upload. Upload must have "removable" option enabled.
 
Get files :

  - **HEAD** /file/:uploadid/:fileid:/:filename:
    - Returning only HTTP headers. Usefull to know Content-Type and Content-Type of file without downloading it. Especially if upload has OneShot option enabled.

  - **GET**  /file/:uploadid/:fileid:/:filename:
    - Download specified file from upload. Filename **MUST** be right. In a browser, it will try to display file (if it's a jpeg for example). You can force download with dl=1 in url.

  - **GET**  /file/:uploadid/:fileid:/:filename:/yubikey/:yubikeyOtp:
    - Same as previous call, except that you can specify a Yubikey OTP in the URL if the upload is Yubikey restricted.


Examples :
```sh
Create an upload (in the json response, you'll have upload id and upload token)
$ curl -X POST 127.0.0.1:8080/upload

Create a OneShot upload
$ curl -X POST -d '{ "OneShot" : true }' 127.0.0.1:8080/upload

Upload a file to upload
$ curl -X POST --header "X-UploadToken: M9PJftiApG1Kqr81gN3Fq1HJItPENMhl" -F "file=@test.txt" 127.0.0.1:8080/upload/IsrIPIsDskFpN12E/file

Get headers
$ curl -I 127.0.0.1:8080/file/IsrIPIsDskFpN12E/sFjIeokH23M35tN4/test.txt
HTTP/1.1 200 OK
Content-Disposition: filename=test.txt
Content-Length: 3486
Content-Type: text/plain; charset=utf-8
Date: Fri, 15 May 2015 09:16:20 GMT

```

### Cli client
Plik is shipped with a golang multiplatform cli client (downloadable in web interface) :
```sh
Simple upload
$ plik file.doc
Multiple files
$ plik file.doc project.doc
Archive and upload directory (using tar+gzip by default)
$ plik -a project/
Secure upload (OpenSSL with aes-256-cbc by deault)
$ plik -s file.doc

```


### FAQ

##### I have an error when uploading from client : "invalid character '<' looking for beginning of value"

Under nginx < 1.3.9, you must enable HttpChunkin module to allow transfer-encoding "chunked".

For debian, this module is present in the "nginx-extras" package

And add in your server configuration :

```sh
        chunkin on;
        error_page 411 = @my_411_error;
        location @my_411_error {
                chunkin_resume;
        }
```


### Participate

You are free to implement other data/metadata/shorten backends and submit them via
pull requests. We will be happy to add them in the future releases.
