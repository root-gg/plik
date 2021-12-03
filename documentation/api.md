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
  "files" : [
    {
      "fileName": "file.txt",
      "fileSize": 12345,
      "fileType": "text/plain",
      "reference": "0"
    },...
  ]
  ```
  
   - **GET** /upload/:uploadid:
     - Get upload metadata (files list, upload date, ttl,...)

Upload file :

   - **POST** /$mode/:uploadid:/:fileid:/:filename:
     - Request body must be a multipart request with a part named "file" containing file data.

   - **POST** /file/:uploadid:
     - Same as above without passing file id, won't work for stream mode.
     
   - **POST** /:
     - Quick mode, automatically create an upload with default parameters and add the file to it.

Get file :

  - **HEAD** /$mode/:uploadid:/:fileid:/:filename:
    - Returns only HTTP headers. Useful to know Content-Type and Content-Length without downloading the file. Especially if upload has OneShot option enabled.

  - **GET**  /$mode/:uploadid:/:fileid:/:filename:
    - Download file. Filename **MUST** match. A browser, might try to display the file if it's a jpeg for example. You may try to force download with ?dl=1 in url.

  - **GET**  /archive/:uploadid:/:filename:
    - Download uploaded files in a zip archive. :filename: must end with .zip

Remove file :

   - **DELETE** /$mode/:uploadid:/:fileid:/:filename:
     - Delete file. Upload **MUST** have "removable" option enabled.

Show server details :

   - **GET** /version
     - Show plik server version, and some build information (build host, date, git revision,...)

   - **GET** /config
     - Show plik server configuration (ttl values, max file size, ...)

   - **GET** /stats
     - Get server statistics ( upload/file count, user count, total size used )
     - Admin only

User authentication :

   - 
   Plik can authenticate users using Google and/or OVH third-party API.   
   The /auth API is designed for the Plik web application nevertheless if you want to automatize it be sure to provide a valid
   Referrer HTTP header and forward all session cookies.   
   Plik session cookies have the "secure" flag set, so they can only be transmitted over secure HTTPS connections.   
   To avoid CSRF attacks the value of the plik-xsrf cookie MUST be copied in the X-XSRFToken HTTP header of each
   authenticated request.   
   Once authenticated a user can generate upload tokens. Those tokens can be used in the X-PlikToken HTTP header used to link
   an upload to the user account. It can be put in the ~/.plikrc file of the Plik command line client.   
   
   - **Local** :
      - You'll need to create users using the server command line
   
   - **Google** :
      - You'll need to create a new application in the [Google Developper Console](https://console.developers.google.com)
      - You'll be handed a Google API ClientID and a Google API ClientSecret that you'll need to put in the plikd.cfg file
      - Do not forget to whitelist valid origin and redirect url ( https://yourdomain/auth/google/callback ) for your domain
   
   - **OVH** :
      - You'll need to create a new application in the OVH API : https://eu.api.ovh.com/createApp/
      - You'll be handed an OVH application key and an OVH application secret key that you'll need to put in the plikd.cfg file

   - **GET** /auth/google/login
      - Get Google user consent URL. User have to visit this URL to authenticate

   - **GET** /auth/google/callback
     - Callback of the user consent dialog
     - The user will be redirected back to the web application with a Plik session cookie at the end of this call

   - **GET** /auth/ovh/login
     - Get OVH user consent URL. User have to visit this URL to authenticate
     - The response will contain a temporary session cookie to forward the API endpoint and OVH consumer key to the callback

   - **GET** /auth/ovh/callback
     - Callback of the user consent dialog. 
     - The user will be redirected back to the web application with a Plik session cookie at the end of this call

   - **POST** /auth/local/login
     - Params :
       - login : user login
       - password : user password

   - **GET** /auth/logout
     - Invalidate Plik session cookies

   - **GET** /me
     - Return basic user info ( ID, name, email ) and tokens

   - **DELETE** /me
     - Remove user account.

   - **GET** /me/token
     - List user tokens
      - This call use pagination

   - **POST** /me/token
     - Create a new upload token
     - A comment can be passed in the json body

   - **DELETE** /me/token/{token}
     - Revoke an upload token

   - **GET** /me/uploads
     - List user uploads
     - Params :
        - token : filter by token
      - This call use pagination

   - **DELETE** /me/uploads
     - Remove all uploads linked to a user account
     - Params :
        - token : filter by token

   - **GET** /me/stats
     - Get user statistics ( upload/file count, total size used )

   - **GET** /users
     - List all users
     - This call use pagination
     - Admin only 

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
