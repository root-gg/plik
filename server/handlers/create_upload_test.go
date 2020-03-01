package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func createTestUpload(t *testing.T, ctx *context.Context, upload *common.Upload) {
	upload.PrepareInsertForTests()
	err := ctx.GetMetadataBackend().CreateUpload(upload)
	require.NoError(t, err, "create upload error")
	ctx.SetUpload(upload)
	if len(upload.Files) == 1 {
		ctx.SetFile(upload.Files[0])
	}
}

func TestCreateUploadWithoutOptions(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var upload = &common.Upload{}
	err = json.Unmarshal(respBody, upload)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, "", upload.ID, "missing upload id")
	require.NotEqual(t, "", upload.UploadToken, "missing upload token")
}

func TestCreateUploadWithOptions(t *testing.T) {
	config := common.NewConfiguration()
	config.Authentication = true

	ctx := newTestingContext(config)

	uploadToCreate := &common.Upload{}
	uploadToCreate.OneShot = true
	uploadToCreate.Removable = true
	uploadToCreate.Stream = true
	uploadToCreate.User = "user"
	uploadToCreate.Token = "token"
	uploadToCreate.ProtectedByPassword = true
	uploadToCreate.Login = "foo"
	uploadToCreate.Password = "bar"

	fileToUpload := &common.File{}
	fileToUpload.Name = "file"
	fileToUpload.Reference = "0"
	uploadToCreate.Files = append(uploadToCreate.Files, fileToUpload)

	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var upload = &common.Upload{}
	err = json.Unmarshal(respBody, upload)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, "", upload.ID, "missing upload id")
	require.NotEqual(t, "", upload.UploadToken, "missing upload token")
	require.Equal(t, uploadToCreate.OneShot, upload.OneShot, "invalid upload oneshot status")
	require.Equal(t, uploadToCreate.Removable, upload.Removable, "invalid upload removable status")
	require.Equal(t, uploadToCreate.Stream, upload.Stream, "invalid upload stream status")
	require.Equal(t, "", upload.User, "invalid upload user")
	require.Equal(t, "", upload.Token, "invalid upload token")
	require.Equal(t, uploadToCreate.ProtectedByPassword, upload.ProtectedByPassword, "invalid upload protected by password status")
	require.Equal(t, "", upload.Login, "invalid upload login")
	require.Equal(t, "", upload.Password, "invalid upload password")
	require.Equal(t, len(uploadToCreate.Files), len(upload.Files), "invalid upload password")

	for _, file := range upload.Files {
		require.NotEqual(t, "", file.ID, "missing file id")
		require.Equal(t, fileToUpload.Name, file.Name, "invalid file name")
		require.Equal(t, fileToUpload.Reference, file.Reference, "invalid file reference")
		require.Equal(t, "missing", file.Status, "invalid file status")
	}
}

func TestCreateWithForbiddenOptions(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	uploadToCreate.ID = "custom"
	uploadToCreate.DownloadDomain = "hack.me"
	uploadToCreate.UploadToken = "token"

	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var upload = &common.Upload{}
	err = json.Unmarshal(respBody, upload)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, uploadToCreate.ID, upload.ID, "invalid upload id")
	require.NotEqual(t, uploadToCreate.UploadToken, upload.UploadToken, "invalid upload token")
	require.NotEqual(t, uploadToCreate.DownloadDomain, upload.DownloadDomain, "invalid download domain")
	require.Equal(t, 0, len(upload.Files), "invalid upload files count")
}

func TestCreateWithoutAnonymousUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().NoAnonymousUploads = true

	uploadToCreate := &common.Upload{}
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "anonymous uploads are disabled")
}

func TestCreateNotWhitelisted(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetWhitelisted(false)

	uploadToCreate := &common.Upload{}
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestForbidden(t, rr, "untrusted source IP address")
}

func TestCreateInvalidRequestBody(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer([]byte("invalid request body")))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "unable to deserialize request body")
}

func TestCreateTooManyFiles(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().MaxFilePerUpload = 2

	uploadToCreate := &common.Upload{}

	for i := 0; i < 10; i++ {
		fileToUpload := &common.File{}
		fileToUpload.Reference = strconv.Itoa(i)
		uploadToCreate.Files = append(uploadToCreate.Files, fileToUpload)
	}

	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "too many files. maximum is")
}

func TestCreateOneShotWhenOneShotIsDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().OneShot = false

	uploadToCreate := &common.Upload{}
	uploadToCreate.OneShot = true
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "one shot uploads are not enabled")
}

func TestCreateOneShotWhenRemovableIsDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Removable = false

	uploadToCreate := &common.Upload{}
	uploadToCreate.Removable = true
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "removable uploads are not enabled")
}

func TestCreateStreamWhenStreamIsDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().Stream = false

	uploadToCreate := &common.Upload{}
	uploadToCreate.Stream = true
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "stream mode is not enabled")
}

func TestCreateInvalidTTL(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().MaxTTL = 30

	uploadToCreate := &common.Upload{}
	uploadToCreate.TTL = 365
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid ttl. (maximum allowed is : 30)")
}

func TestCreateInvalidNegativeTTL(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	uploadToCreate.TTL = -365
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid ttl")
}

func TestCreateWithPasswordWhenPasswordIsNotEnabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().ProtectedByPassword = false

	uploadToCreate := &common.Upload{}
	uploadToCreate.Password = "password"
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "password protection is not enabled")
}

func TestCreateWithPasswordAndDefaultLogin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	uploadToCreate.Password = "password"
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var upload = &common.Upload{}
	err = json.Unmarshal(respBody, upload)
	require.NoError(t, err, "unable to unmarshal response body")
}

func TestCreateWithFilenameTooLong(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	file := common.NewFile()
	name := make([]byte, 2000)
	for i := range name {
		name[i] = 'x'
	}
	file.Name = string(name)
	uploadToCreate.Files = append(uploadToCreate.Files, file)

	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "is too long")
}

//func TestCreateWithMetadataBackendError(t *testing.T) {
//	ctx := newTestingContext(common.NewConfiguration())
//	ctx.GetMetadataBackend().(*metadatadata_test.Backend).SetError(errors.New("metadata backend error"))
//
//	uploadToCreate := common.NewUpload()
//	file := common.NewFile()
//	file.Name = "name"
//	uploadToCreate.Files[file.ID] = file
//
//	reqBody, err := json.Marshal(uploadToCreate)
//	require.NoError(t, err, "unable to marshal request body")
//
//	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
//	require.NoError(t, err, "unable to create new request")
//
//	rr := ctx.NewRecorder(req)
//	CreateUpload(ctx, rr, req)
//	context.TestInternalServerError(t, rr, "create upload error : metadata backend error")
//}
