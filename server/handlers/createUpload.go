/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadataBackend"
	"github.com/root-gg/plik/server/shortenBackend"
)

// CreateUpload create a new upload
func CreateUpload(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	user := common.GetUser(ctx)
	if user == nil && !common.IsWhitelisted(ctx) {
		log.Warning("Unable to create upload from untrusted source IP address")
		common.Fail(ctx, req, resp, "Unable to create upload from untrusted source IP address. Please login or use a cli token.", 403)
		return
	}

	upload := common.NewUpload()
	// Save upload in the request context
	ctx.Set("upload", upload)

	// Read request body
	defer req.Body.Close()
	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warningf("Unable to read request body : %s", err)
		common.Fail(ctx, req, resp, "Unable to read request body", 500)
		return
	}

	// Deserialize json body
	if len(body) > 0 {
		err = json.Unmarshal(body, upload)
		if err != nil {
			log.Warningf("Unable to deserialize request body : %s", err)
			common.Fail(ctx, req, resp, "Unable to deserialize json request bodyy", 500)
			return
		}
	}

	// Set upload id, creation date, upload token, ...
	upload.Create()

	// Update request logger prefix
	prefix := fmt.Sprintf("%s[%s]", log.Prefix, upload.ID)
	log.SetPrefix(prefix)
	ctx.Set("upload", upload)

	// Set upload remote IP
	upload.RemoteIP = common.GetSourceIP(ctx).String()

	// Set upload user and token
	if user != nil {
		upload.User = user.ID
		token := common.GetToken(ctx)
		if token != nil {
			upload.Token = token.Token
		}
	}

	if upload.Stream {
		if !common.Config.StreamMode {
			log.Warning("Stream mode is not enabled")
			common.Fail(ctx, req, resp, "Stream mode is not enabled", 403)
			return
		}
		upload.OneShot = true
	}

	// TTL = Time in second before the upload expiration
	// 0 	-> No ttl specified : default value from configuration
	// -1	-> No expiration : checking with configuration if that's ok
	switch upload.TTL {
	case 0:
		upload.TTL = common.Config.DefaultTTL
	case -1:
		if common.Config.MaxTTL != -1 {
			log.Warningf("Cannot set infinite ttl (maximum allowed is : %d)", common.Config.MaxTTL)
			common.Fail(ctx, req, resp, fmt.Sprintf("Cannot set infinite ttl (maximum allowed is : %d)", common.Config.MaxTTL), 400)
			return
		}
	default:
		if upload.TTL <= 0 {
			log.Warningf("Invalid value for ttl : %d", upload.TTL)
			common.Fail(ctx, req, resp, fmt.Sprintf("Invalid value for ttl : %d", upload.TTL), 400)
			return
		}
		if common.Config.MaxTTL > 0 && upload.TTL > common.Config.MaxTTL {
			log.Warningf("Cannot set ttl to %d (maximum allowed is : %d)", upload.TTL, common.Config.MaxTTL)
			common.Fail(ctx, req, resp, fmt.Sprintf("Cannot set ttl to %d (maximum allowed is : %d)", upload.TTL, common.Config.MaxTTL), 400)
			return
		}
	}

	// Protect upload with HTTP basic auth
	// Add Authorization header to the response for convenience
	// So clients can just copy this header into the next request
	if upload.Password != "" {
		upload.ProtectedByPassword = true
		if upload.Login == "" {
			upload.Login = "plik"
		}

		// The Authorization header will contain the base64 version of "login:password"
		// Save only the md5sum of this string to authenticate further requests
		b64str := base64.StdEncoding.EncodeToString([]byte(upload.Login + ":" + upload.Password))
		upload.Password, err = utils.Md5sum(b64str)
		if err != nil {
			log.Warningf("Unable to generate password hash : %s", err)
			common.Fail(ctx, req, resp, common.NewResult("Unable to generate password hash", nil).ToJSONString(), 500)
			return
		}
		resp.Header().Add("Authorization", "Basic "+b64str)
	}

	// Check the token validity with api.yubico.com
	// Only the Yubikey id part of the token is stored
	// The yubikey id is the 12 first characters of the token
	// The 32 lasts characters are the actual OTP
	if upload.Yubikey != "" {
		upload.ProtectedByYubikey = true

		if !common.Config.YubikeyEnabled {
			log.Warningf("Got a Yubikey upload but Yubikey backend is disabled")
			common.Fail(ctx, req, resp, "Yubikey are disabled on this server", 403)
			return
		}

		_, ok, err := common.Config.YubiAuth.Verify(upload.Yubikey)
		if err != nil {
			log.Warningf("Unable to validate yubikey token : %s", err)
			common.Fail(ctx, req, resp, "Unable to validate yubikey token", 500)
			return
		}

		if !ok {
			log.Warningf("Invalid yubikey token")
			common.Fail(ctx, req, resp, "Invalid yubikey token", 400)
			return
		}

		upload.Yubikey = upload.Yubikey[:12]
	}

	// A short url is created for each upload if a shorten backend is specified in the configuration.
	// Referer header is used to get the url of incoming request, clients have to set it in order
	// to get this feature working
	if shortenBackend.GetShortenBackend() != nil {
		if req.Header.Get("Referer") != "" {
			u, err := url.Parse(req.Header.Get("Referer"))
			if err != nil {
				log.Warningf("Unable to parse referer url : %s", err)
			}
			longURL := u.Scheme + "://" + u.Host + "#/?id=" + upload.ID
			shortURL, err := shortenBackend.GetShortenBackend().Shorten(ctx, longURL)
			if err == nil {
				upload.ShortURL = shortURL
			} else {
				log.Warningf("Unable to shorten url %s : %s", longURL, err)
			}
		}
	}

	// Create files
	for i, file := range upload.Files {

		// Check file name length
		if len(file.Name) > 1024 {
			log.Warning("File name is too long")
			common.Fail(ctx, req, resp, "File name is too long. Maximum length is 1024 characters", 400)
			return
		}

		file.GenerateID()
		file.Status = "missing"
		delete(upload.Files, i)
		upload.Files[file.ID] = file
	}

	// Save the metadata
	err = metadataBackend.GetMetaDataBackend().Create(ctx, upload)
	if err != nil {
		log.Warningf("Create new upload error : %s", err)
		common.Fail(ctx, req, resp, "Unable to create new upload", 500)
		return
	}

	// Remove all private information (ip, data backend details, ...) before
	// sending metadata back to the client
	uploadToken := upload.UploadToken
	upload.Sanitize()

	// Show upload token since its an upload creation
	upload.UploadToken = uploadToken
	upload.IsAdmin = true

	// Print upload metadata in the json response.
	var json []byte
	if json, err = utils.ToJson(upload); err != nil {
		log.Warningf("Unable to serialize json response : %s", err)
		common.Fail(ctx, req, resp, "Unable to serialize json response", 500)
		return
	}

	resp.Write(json)
}
