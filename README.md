[![Build Status](https://travis-ci.org/root-gg/plik.svg?branch=master)](https://travis-ci.org/root-gg/plik)
[![Go Report](https://img.shields.io/badge/Go_report-A+-brightgreen.svg)](http://goreportcard.com/report/root-gg/plik)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT)

# Plik

Plik is an simple and powerful file uploading system written in golang.

### Main features
   - Multiple data backends : File, OpenStack Swift, WeedFS
   - Multiple metadata backends : File, MongoDB
   - Shorten backends : Shorten upload urls (is.gd && w000t.me available)
   - OneShot : Files are destructed after the first download
   - Stream : Files are streamed from the uploader to the downloader (nothing stored server side)  
   - Removable : Give the ability to the uploader to remove files at any time
   - TTL : Custom expiration date
   - Password : Protect upload with login/password (Auth Basic)
   - Yubikey : Protect upload with your yubikey. (One Time Password)
   - Comments : Add custom message (in Markdown format)

### Version
1.0


### Installation

##### From release
To run plik, it's very simple :
```sh
$ wget https://github.com/root-gg/plik/releases/download/1.0/plik-1.0.tar.gz
$ tar xvf plik-1.0.tar.gz
$ cd plik-1.0/server
$ ./plikd
```
Et voilà ! You now have a fully functional instance of plik running on http://127.0.0.1:8080.  
You can edit server/plikd.cfg to adapt the configuration to your needs (ports, ssl, ttl, backends params,...)

##### From sources
To compile plik from sources, you'll need golang and npm installed on your system.

First, get the project and libs via go get :
```sh
$ go get github.com/root-gg/plik/server
$ cd $GOPATH/github.com/root-gg/plik/
```

As root user you need to install grunt, bower, and setup the golang crosscompilation environnement :
```sh
$ sudo -c "npm install -g bower grunt-cli"
$ sudo -c "client/build.sh env"
```

To build everything :
```sh
$ make build
$ make clients
```

### API
Plik server expose a REST-full API to manage uploads and get files :

To create upload and upload files :
 
   - **POST**        /upload
     - Params (json object in request body) :
      - oneshot (bool)
      - stream (bool)
      - removable (bool)
      - ttl (int)
      - login (string)
      - password (string)
     - Return :
         JSON formatted upload object.
         Important fields :
           - id (required to upload files)
           - uploadToken (required to upload/remove files)

   - **GET** /upload/:uploadid:
     - Get upload metadata (files list, upload date, ttl,...)

   - **POST** /upload/:uploadid:/file
     - Body must be a multipart request with a part named "file" containing file data
   Returns a JSON object of uploaded file metadata
   
   - **POST** /file/:uploadid/:fileid: (same as above)
     - For stream mode you need to know the file id before the upload starts as it will block.  
   To get the file ids pass a files hash param to the previous create upload call with each file you are about to upload.  
   Fill the reference field with an arbitrary string to avoid to match file ids using the fileName field.
   ```
   upload.files : {
     "0" : {
       fileName: "file.txt"
       fileSize: 12345
       fileType: "text/plain"
       reference: "0"
     },...
   }
   ```
   
   - **DELETE** /upload/:uploadid:/file/:fileid:
     - Delete file from the upload. Upload must have "removable" option enabled.
 
Get files :

  - **HEAD** /file/:uploadid/:fileid:/:filename:
    - Returns only HTTP headers. Usefull to know Content-Type and Content-Type without downloading the file. Especially if upload has OneShot option enabled.

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

##### I have an error when uploading from client : "Unable upload file : HTTP error 411 Length Required"

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

##### Why is stream mode broken in multiple instance deployement ?

Beacause stream mode isn't stateless. As the uploader request blocks on only one plik instance the downloader request MUST go to the same server to succeed. Loadbalancing strategy should be aware of this and route stream requests to the same plik instance by hashing the file id. Plik 1.1 API will provide new API paths to distinguish streams and load balancing configuration examples to ease deployement.

### Participate

You are free to implement other data/metadata/shorten backends and submit them via
pull requests. We will be happy to add them in the future releases.
