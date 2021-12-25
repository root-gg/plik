package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func createTestUpload(t *testing.T, ctx *context.Context, upload *common.Upload) {
	upload.InitializeForTests()
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
	require.True(t, upload.IsAdmin, "invalid upload admin status") // You are always admin of your own upload

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
	uploadToCreate.TTL = 60

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
	require.Equal(t, 60, upload.TTL, "invalid upload TTL")
	require.Equal(t, len(uploadToCreate.Files), len(upload.Files), "invalid upload password")
	require.True(t, upload.IsAdmin, "invalid upload admin status") // You are always admin of your own upload

	for _, file := range upload.Files {
		require.NotEqual(t, "", file.ID, "missing file id")
		require.Equal(t, fileToUpload.Name, file.Name, "invalid file name")
		require.Equal(t, fileToUpload.Reference, file.Reference, "invalid file reference")
		require.Equal(t, "missing", file.Status, "invalid file status")
	}

	require.Equal(t, "Basic "+common.EncodeAuthBasicHeader("foo", "bar"), rr.Header().Get("Authorization"))
}

func TestCreateWithForbiddenOptions(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	uploadToCreate.ID = "custom"
	uploadToCreate.DownloadDomain = "hack.me"
	uploadToCreate.UploadToken = "token"
	uploadToCreate.IsAdmin = false

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
	require.True(t, upload.IsAdmin, "invalid upload admin status") // You are always admin of your own upload
}

func TestCreateUploadInvalidParameters(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().OneShot = false

	uploadToCreate := &common.Upload{OneShot: true}
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "one shot uploads are not enabled")
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

type NeverEndingReader struct{}

func (r *NeverEndingReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte('x')
	}
	return len(p), nil
}

func TestCreateUpload_BodyTooBig(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/upload", &NeverEndingReader{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	CreateUpload(ctx, rr, req)

	context.TestBadRequest(t, rr, "request body too large")
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
