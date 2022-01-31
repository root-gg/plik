package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// CreateUpload create a new upload
func CreateUpload(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	log := ctx.GetLogger()
	config := ctx.GetConfig()

	if !ctx.IsWhitelisted() {
		ctx.Forbidden("untrusted source IP address")
		return
	}

	// Read request body
	defer func() { _ = req.Body.Close() }()
	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ctx.BadRequest("unable to read request body : %s", err)
		return
	}

	// Deserialize json body
	uploadParams := &common.Upload{}
	version := 0
	if len(body) > 0 {
		version, err = common.UnmarshalUpload(body, uploadParams)
		if err != nil {
			ctx.BadRequest("unable to deserialize request body : %s", err)
			return
		}
	}

	// Create upload from user params
	upload, err := ctx.CreateUpload(uploadParams)
	if err != nil {
		ctx.BadRequest("unable to create upload : %s", err)
		return
	}

	// Update request logger prefix
	prefix := fmt.Sprintf("%s[%s]", log.Prefix, upload.ID)
	log.SetPrefix(prefix)
	ctx.SetUpload(upload)

	// Save the upload to the metadata database
	err = ctx.GetMetadataBackend().CreateUpload(upload)
	if err != nil {
		ctx.InternalServerError("create upload error", err)
		return
	}

	// You are admin of your own uploads
	upload.IsAdmin = true

	// Hide private information (IP, data backend details, User ID, Login/Password, ...)
	upload.Sanitize(config)

	if upload.ProtectedByPassword {
		// Add Authorization header to the response for convenience
		// So clients can just copy this header into the next request
		// The Authorization header will contain the base64 version of "login:password"
		header := common.EncodeAuthBasicHeader(uploadParams.Login, uploadParams.Password)
		resp.Header().Add("Authorization", "Basic "+header)
	}

	// Print upload metadata in the json response.
	bytes, err := common.MarshalUpload(upload, version)
	if err != nil {
		ctx.InternalServerError("unable to serialize upload", err)
	}

	_, _ = resp.Write(bytes)
}
