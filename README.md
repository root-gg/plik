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

##### From root.gg Debian repository

Configure root.gg repository and install server and/or client
```
    wget -O - http://mir.root.gg/gg.key | apt-key add -
    echo "deb http://mir.root.gg/ $(lsb_release --codename --short) main" > /etc/apt/sources.list.d/root.gg.list
    apt-get update
    apt-get install plikd plik
```

Edit server configuration at /etc/plikd.cfg and start the server 
```
    service plikd start
```

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
$ make server
$ make clients
```

To make debian packages :
```
$ make debs-server debs-client
```

To make release archives :
```
$ make releases
```

### API
Plik server expose a REST-full API to manage uploads and get files :

Get and create upload :
 
   - **POST**        /upload
     - Params (json object in request body) :
      - oneshot (bool)
      - stream (bool)
      - removable (bool)
      - ttl (int)
      - login (string)
      - password (string)
      - files (see below)
     - Return :
         JSON formatted upload object.
         Important fields :
           - id (required to upload files)
           - uploadToken (required to upload/remove files)
           - files (see below)

   For stream mode you need to know the file id before the upload starts as it will block.
   File size and/or file type also need to be known before the upload starts as they have to be printed 
   in HTTP response headers.
   To get the file ids pass a "files" json object with each file you are about to upload.
   Fill the reference field with an arbitrary string to avoid matching file ids using the fileName field.
   This is also used to notify of MISSING files when file upload is not yet finished or has failed.
  ```
  "files" : {
    "0" : {
      "fileName": "file.txt",
      "fileSize": 12345,
      "fileType": "text/plain",
      "reference": "0"
    },...
  }
  ```
   - **GET** /upload/:uploadid:
     - Get upload metadata (files list, upload date, ttl,...)

Upload file :

   - **POST** /$mode/:uploadid:/:fileid:/:filename:
     - Request body must be a multipart request with a part named "file" containing file data.

   - **POST** /file/:uploadid:
     - Same as above without passing file id, won't work for stream mode.

Get file :

  - **HEAD** /$mode/:uploadid:/:fileid:/:filename:
    - Returns only HTTP headers. Usefull to know Content-Type and Content-Type without downloading the file. Especially if upload has OneShot option enabled.

  - **GET**  /$mode/:uploadid:/:fileid:/:filename:
    - Download file. Filename **MUST** match. A browser, might try to display the file if it's a jpeg for example. You may try to force download with ?dl=1 in url.

  - **GET**  /$mode/:uploadid:/:fileid:/:filename:/yubikey/:yubikeyOtp:
    - Same as previous call, except that you can specify a Yubikey OTP in the URL if the upload is Yubikey restricted.

Remove file :

   - **DELETE** /$mode/:uploadid:/:fileid:/:filename:
     - Delete file. Upload **MUST** have "removable" option enabled.

$mode can be "file" or "stream" depending if stream mode is enabled. See FAQ for more details.

Examples :
```sh
Create an upload (in the json response, you'll have upload id and upload token)
$ curl -X POST http://127.0.0.1:8080/upload

Create a OneShot upload
$ curl -X POST -d '{ "OneShot" : true }' http://127.0.0.1:8080/upload

Upload a file to upload
$ curl -X POST --header "X-UploadToken: M9PJftiApG1Kqr81gN3Fq1HJItPENMhl" -F "file=@test.txt" http://127.0.0.1:8080/file/IsrIPIsDskFpN12E

Get headers
$ curl -I http://127.0.0.1:8080/file/IsrIPIsDskFpN12E/sFjIeokH23M35tN4/test.txt
HTTP/1.1 200 OK
Content-Disposition: filename=test.txt
Content-Length: 3486
Content-Type: text/plain; charset=utf-8
Date: Fri, 15 May 2015 09:16:20 GMT

```

### Cli client
Plik is shipped with a powerful golang multiplatform cli client (downloadable in web interface) :  

```
Usage:
  plik [options] [FILE] ...

Options:
  -h --help                 Show this help
  -d --debug                Enable debug mode
  -q --quiet                Enable quiet mode
  -o, --oneshot             Enable OneShot ( Each file will be deleted on first download )
  -r, --removable           Enable Removable upload ( Each file can be deleted by anyone at anymoment )
  -S, --stream              Enable Streaming ( It will block until remote user starts downloading )
  -t, --ttl TTL             Time before expiration (Upload will be removed in m|h|d)
  -n, --name NAME           Set file name when piping from STDIN
  --server SERVER           Overrides plik url
  --comments COMMENT        Set comments of the upload ( MarkDown compatible )
  --archive-options OPTIONS [tar|zip] Additional command line options
  -p                        Protect the upload with login and password
  --password PASSWD         Protect the upload with login:password ( if omitted default login is "plik" )
  -y, --yubikey             Protect the upload with a Yubikey OTP
  -a                        Archive upload using default archive params ( see ~/.plikrc )
  --archive MODE            Archive upload using specified archive backend : tar|zip
  --compress MODE           [tar] Compression codec : gzip|bzip2|xz|lzip|lzma|lzop|compress|no
  -s                        Encrypt upload usnig default encrypt params ( see ~/.plikrc )
  --secure MODE             Archive upload using specified archive backend : openssl|pgp
  --cipher CIPHER           [openssl] Openssl cipher to use ( see openssl help )
  --passphrase PASSPHRASE   [openssl] Passphrase or '-' to be prompted for a passphrase
  --secure-options OPTIONS  [openssl|pgp] Additional command line options
  --recipient RECIPIENT     [pgp] Set recipient for pgp backend ( example : --recipient Bob )
  --update                  Update client
```

For example to create directory tar.gz archive and encrypt it with openssl :
```
$plik -a -s mydirectory/
Passphrase : 30ICoKdFeoKaKNdnFf36n0kMH
Upload successfully created : 
    https://127.0.0.1:8080/#/?id=0KfNj6eMb93ilCrl

mydirectory.tar.gz : 15.70 MB 5.92 MB/s

Commands :
curl -s 'https://127.0.0.1:8080/file/0KfNj6eMb93ilCrl/q73tEBEqM04b22GP/mydirectory.tar.gz' | openssl aes-256-cbc -d -pass pass:30ICoKdFeoKaKNdnFf36n0kMH | tar xvf - --gzip
```

Client configuration and preferences are stored at ~/.plikrc ( overridable with PLIKRC environement variable )

### FAQ

##### I have an error when uploading from client : "Unable upload file : HTTP error 411 Length Required"

Under nginx < 1.3.9, you must enable HttpChunkin module to allow transfer-encoding "chunked".  
You might want to install the "nginx-extras" Debian package with built-in HttpChunkin module.

And add in your server configuration :

```sh
        chunkin on;
        error_page 411 = @my_411_error;
        location @my_411_error {
                chunkin_resume;
        }
```

##### Why is stream mode broken in multiple instance deployement ?

Beacause stream mode isn't stateless. As the uploader request will block on one plik instance the downloader request **MUST** go to the same instance to succeed.
The load balancing strategy **MUST** be aware of this and route stream requests to the same instance by hashing the file id.

Here is an example of how to achieve this using nginx and a little piece of LUA.
Make sur your nginx server is built with LUA scripting support.
You might want to install the "nginx-extras" Debian package (>1.7.2) with built-in LUA support.
```
upstream plik {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
}

upstream stream {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    hash $hash_key;
}

server {
    listen 9000;

    location / {
        set $upstream "";
        set $hash_key "";
        access_by_lua '
            _,_,file_id = string.find(ngx.var.request_uri, "^/stream/[a-zA-Z0-9]+/([a-zA-Z0-9]+)/.*$")
            if file_id == nil then
                ngx.var.upstream = "plik"
            else
                ngx.var.upstream = "stream"
                ngx.var.hash_key = file_id
            end
        ';
        proxy_pass http://$upstream;
    }
}
```

##### Is "file" metadata backend compatible with multi-instance ?

Unfortunately, you may experience some weird behaviour using file metadata backend with multiple instances of plik.

The lock used in this backend is specific to a given instance, so the metadata file could be corrupted on concurrent requests.

You can set a 'sticky' on the source ip but we recommend using the MongoDB metadata backend, when deploying a high available plik installation.


##### How to disable nginx buffering ?

By default nginx buffers large HTTP requests and reponses to a temporary file. This behaviour leads to unnecessary disk load and slower transfers. This should be turned off (>1.7.12) for /file and /stream paths. You might also want to increase buffers size.

Detailed documentation : http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffering
```
   proxy_buffering off;
   proxy_request_buffering off;
   proxy_http_version 1.1;
   proxy_buffer_size 1M;
   proxy_buffers 8 1M;
   client_body_buffer_size 1M;
```
##### How to take and upload screenshots like a boss ?

```
alias pshot="scrot -s -e 'plik -q \$f | xclip ; xclip -o ; rm \$f'"
```

Requires you to have plik, scrot and xclip installed in your $PATH.  
scroot -s allow you to "Interactively select a window or rectangle with the mouse" then
Plik will upload the screenshot and the url will be directly copied to your clipboard and displayed by xclip.
The screenshot is then removed of your home directory to avoid garbage.

##### How to contribute to the project ?

Contributions are welcome, you are free to implement other data/metadata/shorten backends and submit them via
pull requests. We will be happy to add them in the future releases.
