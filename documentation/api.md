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

Show server details :

   - **GET** /version
     - Show plik server version, and some build information (build host, date, git revision,...)

   - **GET** /config
     - Show plik server configuration (ttl values, max file size, ...)

User authentication :

   - 
   Plik can authenticate users using Google and/or OVH third-party API. Once authenticated 
   the only call Plik will ever make to those API is get the user ID, name and email. It will never forward any
   upload data or metadata.   
   The /auth API is designed for the Plik web application nevertheless if you want to automatize it just provide a valid
   Referrer HTTP header and forward all session cookies.   
   To avoid CSRF attacks the value of the plik-xsrf cookie MUST be copied in the X-XRSFToken HTTP header of each
   authenticated request.   
   Once authenticated a user can generate upload tokens. Those tokens can be used in the X-PlikToken HTTP header used to link
   an upload to the user account. It can be put in the ~/.plikrc file of the Plik command line client.   
   
   
   - **Google** :
      - You'll need to create a new application in the [Google Developper Console](https://console.developers.google.com)
      - You'll be handed a Google API ClientID and a Google API ClientSecret that you'll need to put in the plikd.cfg file.
      - Do not forget to whitelist valid origin and redirect url ( https://yourdomain/auth/google/callback ) for your domain.
   
   - **OVH** :
      - You'll need to create a new application in the OVH API : https://eu.api.ovh.com/createApp/
      - You'll be handed an OVH application key and an OVH application secret key that you'll need to put in the plikd.cfg file.

   - **GET** /auth/google/login
      - Get Google user consent URL. User have to visit this URL to authenticate.

   - **GET** /auth/google/callback
     - Callback of the user consent dialog.
     - The user will be redirected back to the web application with a Plik session cookie at the end of this call.

   - **GET** /auth/ovh/login
     - Get OVH user consent URL. User have to visit this URL to authenticate. 
     - The response will contain a temporary session cookie to forward the API endpoint and OVH consumer key to the callback.

   - **GET** /auth/google/callback
     - Callback of the user consent dialog. 
     - The user will be redirected back to the web application with a Plik session cookie at the end of this call.

   - **GET** /auth/logout
     - Invalidate Plik session cookies.

   - **GET** /me
     - Return basic user info ( ID, name, email ) and tokens.

   - **DELETE** /me
     - Remove user account.

   - **POST** /me/token
     - Create a new upload token.
     - A comment can be passed in the json body.

   - **DELETE** /me/token/{token}
     - Revoke an upload token.

   - **GET** /me/uploads
     - Return all uploads linked to a user account.
     - Params :
        - token : filter by token
        - size : maximum uploads to return ( max : 100 )
        - offset : number of uploads to skip

   - **DELETE** /me/uploads
     - Remove all uploads linked to a user account.
     - Params :
        - token : filter by token

QRCode :

   - **GET** /qrcode
     - Generate a QRCode image from an url
     - Params :
        - url  : The url you want to store in the QRCode
        - size : The size of the generated image in pixels (default: 250, max: 1000)


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
