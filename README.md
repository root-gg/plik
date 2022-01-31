[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)
[![Build](https://github.com/root-gg/plik/actions/workflows/master.yaml/badge.svg)](https://github.com/root-gg/plik/actions/workflows/master.yaml)
[![Go Report](https://img.shields.io/badge/Go_report-A+-brightgreen.svg)](http://goreportcard.com/report/root-gg/plik)
[![Docker Pulls](https://img.shields.io/docker/pulls/rootgg/plik.svg)](https://hub.docker.com/r/rootgg/plik)
[![GoDoc](https://godoc.org/github.com/root-gg/plik?status.svg)](https://godoc.org/github.com/root-gg/plik)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT)

Want to chat with us ? Telegram channel : https://t.me/plik_rootgg

# Plik

Plik is a scalable & friendly temporary file upload system (Wetransfer like) in golang.

### Main features
   - Powerful command line client
   - Easy to use web UI
   - Multiple data backend : File, OpenStack Swift, S3, Google Cloud Storage
   - Multiple metadata backend : Sqlite3, PostgreSQL, MySQL
   - OneShot : Files are destructed after the first download
   - Stream : Files are streamed from the uploader to the downloader (nothing stored server side)  
   - Removable : Give the ability to the uploader to remove files at any time
   - TTL : Custom expiration date
   - Password : Protect upload with login/pasgisword (Auth Basic)
   - Comments : Add custom message (in Markdown format)
   - User authentication : Local / Google / OVH
   - Upload restriction : Source IP / Token
   - Administrator CLI and web UI
   - Server side encryption (with S3 data backend)
   - [ShareX](https://getsharex.com/) Uploader : Directly integrated into ShareX
   - [plikSharp](https://github.com/iss0/plikSharp) : A .NET API client for Plik
   - [Filelink for Plik](https://gitlab.com/joendres/filelink-plik) : Thunderbird Addon to upload attachments to Plik

### Content Table
1. [Installation](#installation) 
2. [Configuration](#configuration)
3. [Data Backends](#data-backends)
4. [Metadata Backends](#metadata-backends)
5. [Client CLI](#cli-client)
6. [Go Client](#go-client)
7. [HTTP API](#api)
8. [Admin CLI](#admin-cli)
9. [Authentication](#authentication)
10. [Security](#security)
11. [Cross Compilation](#cross-compilation)
12. [FAQ](#faq)
13. [How To Contribute](#how-to-contribute)

### Installation <a name="installation"></a>

##### From release
To run plik, it's very simple :
```sh
$ wget https://github.com/root-gg/plik/releases/download/1.3.5/plik-1.3.5-linux-amd64.tar.gz
$ tar xzvf plik-1.3.5-linux-amd64.tar.gz
$ cd plik-1.3.5-linux-amd64/server
$ ./plikd
```
Et voilà ! You now have a fully functional instance of Plik running on http://127.0.0.1:8080.  
You can edit server/plikd.cfg to adapt the configuration to your needs (ports, ssl, ttl, backend params,...)

##### From sources
To compile plik from sources, you'll need golang and npm installed on your system

Git clone or go get the project and simply run make :
```sh
$ make
$ cd server && ./plikd
```

##### Docker <a name="docker"></a>
Plik comes with multiarch docker images built for linux amd64/i386/arm/arm64: 
 - rootgg/plik:latest (latest release)
 - rootgg/plik:{version} (release)
 - rootgg/plik:dev (latest commit of master)

See the [Plik Docker reference](documentation/docker.md)

Plik also comes with some useful scripts to test backend in standalone docker instances :

See the [Plik Docker backend testing](testing)

### Configuration <a name="configuration"></a>

The configuration is managed using a TOML file [plikd.cfg](server/plikd.cfg)

###### Defining configuration parameters using environment variables

One can specify configuration parameters using env variable with the configuration parameter in screaming snake case  
```
    PLIKD_DEBUG_REQUESTS=true ./plikd
```

For Arrays and config maps they must be provided in json format.
Arrays are overridden but maps are merged

```
    PLIKD_DATA_BACKEND_CONFIG='{"Directory":"/var/files"}' ./plikd
```

### Data backends <a name="data-backends"></a>

Plik is shipped with multiple data backend for uploaded files and metadata backend for the upload metadata.

 - File databackend :

Store uploaded files in a local or mounted file system directory.

 - Openstack Swift databackend : http://docs.openstack.org/developer/swift/

Openstack Swift is a highly available, distributed, eventually consistent object/blob store which supports Server Side Encryption  

 - Amazon S3

 - Google Cloud Storage

### Metadata backends <a name="metadata-backends"></a>

 - Sqlite3

Suitable for standalone deployment.

 - PostgreSQL / Mysql

Suitable for distributed / High Availability deployment.

### Cli client <a name="cli-client"></a>
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
  --token TOKEN             Specify an upload token
  --comments COMMENT        Set comments of the upload ( MarkDown compatible )
  -p                        Protect the upload with login and password
  --password PASSWD         Protect the upload with login:password ( if omitted default login is "plik" )
  -a                        Archive upload using default archive params ( see ~/.plikrc )
  --archive MODE            Archive upload using specified archive backend : tar|zip
  --compress MODE           [tar] Compression codec : gzip|bzip2|xz|lzip|lzma|lzop|compress|no
  --archive-options OPTIONS [tar|zip] Additional command line options
  -s                        Encrypt upload usnig default encrypt params ( see ~/.plikrc )
  --not-secure              Do not encrypt upload regardless of ~/.plikrc configurations
  --secure MODE             Archive upload using specified archive backend : openssl|pgp
  --cipher CIPHER           [openssl] Openssl cipher to use ( see openssl help )
  --passphrase PASSPHRASE   [openssl] Passphrase or '-' to be prompted for a passphrase
  --recipient RECIPIENT     [pgp] Set recipient for pgp backend ( example : --recipient Bob )
  --secure-options OPTIONS  [openssl|pgp] Additional command line options
  --update                  Update client
  -v --version              Show client version
```

For example to create directory tar.gz archive and encrypt it with openssl :
```bash
$ plik -a -s mydirectory/
Passphrase : 30ICoKdFeoKaKNdnFf36n0kMH
Upload successfully created : 
    https://127.0.0.1:8080/#/?id=0KfNj6eMb93ilCrl

mydirectory.tar.gz : 15.70 MB 5.92 MB/s

Commands :
curl -s 'https://127.0.0.1:8080/file/0KfNj6eMb93ilCrl/q73tEBEqM04b22GP/mydirectory.tar.gz' | openssl aes-256-cbc -d -pass pass:30ICoKdFeoKaKNdnFf36n0kMH | tar xvf - --gzip
```

Client configuration and preferences are stored at ~/.plikrc or /etc/plik/plikrc ( overridable with PLIKRC environement variable )

### Quick upload using curl only

```bash
curl --form 'file=@/path/to/file' http://127.0.0.1:8080
```
When Authentication is used and NoAnonymousUploads are enabled you can quick upload using user tokens:
```bash
curl --form 'file=@/path/to/file' --header 'X-PlikToken: xxxx-xxx-xxxx-xxxxx-xxxxxxxx' http://127.0.0.1:8080
```

DownloadDomain configuration option must be set for this to properly work.

### Go client <a name="go-client"></a>

Plik now comes with a golang library above which the cli client is built

See the [Plik library reference](plik/README.md)

### API <a name="api"></a>
Plik server expose a HTTP API to manage uploads and get files :

See the [Plik API reference](documentation/api.md)

### Admin CLI <a name="admin-cli"></a>

Using the ./plikd server binary it's possible to :
  - create/list/delete local accounts
  - create/list/delete user CLI tokens
  - create/list/delete files and uploads
  - import / export metadata

See help for more details
   
### Authentication <a name="authentication"></a>

Plik can authenticate users using Local accounts or using Google or OVH APIs.

If source IP address restriction is enabled, user accounts can only be created from trusted IPs and then 
authenticated users can upload files without source IP restriction.

It possible to deny unauthenticated uploads totally ( NoAnonymousUploads ).  

Admin users can access the admin dashboard and manipulate every uploads.

   - **Local** :
      - You can manipulate local users with the server command line
      
      ```sh
      $ ./plikd --config ./plikd.cfg user create --login root --name Admin --admin    
      Generated password for user root is 08ybEyh2KkiMho8dzpdQaJZm78HmvWGC
      ```
      
   - **Google** :
      - You'll need to create a new application in the [Google Developper Console](https://console.developers.google.com)
      - You'll be handed a Google API ClientID and a Google API ClientSecret that you'll need to put in the plikd.cfg file.
      - Do not forget to whitelist valid origin and redirect url ( https://yourdomain/auth/google/callback ) for your domain.
      - It is possible to whitelist only one or more email domains.
   
   - **OVH** :
      - You'll need to create a new application in the OVH API : https://eu.api.ovh.com/createApp/
      - You'll be handed an OVH application key and an OVH application secret key that you'll need to put in the plikd.cfg file.

Once authenticated a user can generate upload tokens that can be specified in the ~/.plikrc file to authenticate
the command line client.

```
Token = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

### Security <a name="security"></a>
Plik allow users to upload and serve any content as-is, but hosting untrusted HTML raises some well known security concerns.

Plik will try to avoid HTML rendering by overriding Content-Type to "text-plain" instead of "text/html".
  
By default Plik sets a couple of security HTTP headers like **X-Content-Type-Options, X-XSS-Protection, X-Frame-Options, Content-Security-Policy** to disable sensible features of most recent browsers like resource loading, xhr requests, iframes,...
This will however break features like audio/video playback, pdf rendering so it's possible to disable this behavior by setting the EnhancedWebSecurity configuration parameter to false
  
Along with that it is also strongly advised to serve uploaded files on a separate (sub-)domain to fight against phishing links and to protect Plik's session cookie with the DownloadDomain configuration parameter.  

### Cross compilation <a name="cross-compilation"></a>

All binary are now statically linked. Clients can be safely cross-compiled for all os/architectures as they do not rely on GCO (sqlite)
Servers rely on CGO/sqlite need a cross-compilation ready environment.

 `make release` will build release archives for `amd64,i386,arm,arm64`

To build a release with only specific architectures of the client
```
    CLIENT_TARGETS="linux/amd64" releaser/release.sh
```

To build only specific for only specific architectures
```
    TARGETS="linux/amd64" releaser/release.sh
```

To build with a specific cross compiler toolchain
```
    TARGETS="linux/arm/v6" CC=arm-linux-gnueabihf-gcc releaser/release.sh
```

### FAQ <a name="faq"></a>

* Why is stream mode broken in multiple instance deployement ?

Beacause stream mode isn't stateless. As the uploader request will block on one plik instance the downloader request **MUST** go to the same instance to succeed.
The load balancing strategy **MUST** be aware of this and route stream requests to the same instance by hashing the file id.

Here is an example of how to achieve this using nginx and a little piece of LUA.
Make sure your nginx server is built with LUA scripting support.
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

* Redirection loops with DownloadDomain enforcement and reverse proxy

```
Invalid download domain 127.0.0.1:8080, expected plik.root.gg
```

DownloadDomain check the Host header of the incoming HTTP request, by default reverse proxies like
Nginx or Apache mod_proxy does not forward this Header. Check the following configuration directive :

```
Apache mod_proxy : ProxyPreserveHost On
Nginx : proxy_set_header Host $host;
```

* I have an error when uploading from client : "Unable to upload file : HTTP error 411 Length Required"

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

* How to disable nginx buffering ?

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

* Why authentication does not work with HTTP connections when EnhancedWebSecurity is set ?

Plik session cookies have the "secure" flag set when EnhancedWebSecurity is set so they can only be transmitted over secure HTTPS connections.

* Build failure "/usr/bin/env: ‘node’: No such file or directory"

Debian users might need to install the nodejs-legacy package.

```
This package contains a symlink for legacy Node.js code requiring
binary to be /usr/bin/node (not /usr/bin/nodejs as provided in Debian).
```

* How to take and upload screenshots like a boss ?

```
alias pshot="scrot -s -e 'plik -q \$f | xclip ; xclip -o ; rm \$f'"
```

Requires you to have plik, scrot and xclip installed in your $PATH.  
scrot -s allow you to "Interactively select a window or rectangle with the mouse" then
Plik will upload the screenshot and the url will be directly copied to your clipboard and displayed by xclip.
The screenshot is then removed of your home directory to avoid garbage.

### How to contribute to the project ? <a name="how-to-contribute"></a>

Contributions are welcome, feel free to open issues and/or submit pull requests.
Please be sure to also run/update the test suite :

```
    make fmt
    make lint
    make test
    make test-backends
```
