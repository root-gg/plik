package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

var content = "data data data"
var contentMD5 = "2b1ef3928c85db885c68ff4f47fe9b33"

func getMultipartFormData(name string, in io.Reader) (out io.Reader, contentType string, err error) {
	return getMultipartFormDataWithField("file", name, in)
}

func getMultipartFormDataWithField(fieldname string, name string, in io.Reader) (out io.Reader, contentType string, err error) {
	buffer := new(bytes.Buffer)
	multipartWriter := multipart.NewWriter(buffer)

	writer, err := multipartWriter.CreateFormFile(fieldname, name)
	if err != nil {
		return nil, "", fmt.Errorf("unable to create multipartWriter : %s", err)
	}

	_, err = io.Copy(writer, in)
	if err != nil {
		return nil, "", err
	}

	err = multipartWriter.Close()
	if err != nil {
		return nil, "", err
	}

	return buffer, multipartWriter.FormDataContentType(), nil
}

func getUploadRequest(t *testing.T, upload *common.Upload, file *common.File, reader io.Reader, contentType string) (req *http.Request) {
	req, err := http.NewRequest("POST", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID": file.ID,
	}
	req = mux.SetURLVars(req, vars)

	return req
}

func TestAddFileWithID(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var fileResult = &common.File{}
	err = json.Unmarshal(respBody, fileResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, file.ID, fileResult.ID, "invalid file id")
	require.Equal(t, file.Name, fileResult.Name, "invalid file name")
	require.Equal(t, common.FileUploaded, fileResult.Status, "invalid file status")
	require.Equal(t, contentMD5, fileResult.Md5, "invalid file md5")
	require.Equal(t, "text/plain; charset=utf-8", fileResult.Type, "invalid file type")
	require.True(t, fileResult.IsText, "plain text should be detected as text")
	require.Equal(t, int64(len(content)), fileResult.Size, "invalid file size")
}

func TestAddStreamFileWithID(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	upload.Stream = true
	file := upload.NewFile()
	file.Name = "file"
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var fileResult = &common.File{}
	err = json.Unmarshal(respBody, fileResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, file.ID, fileResult.ID, "invalid file id")
	require.Equal(t, file.Name, fileResult.Name, "invalid file name")
	require.Equal(t, common.FileDeleted, fileResult.Status, "invalid file status")
	require.Equal(t, contentMD5, fileResult.Md5, "invalid file md5")
	require.Equal(t, "text/plain; charset=utf-8", fileResult.Type, "invalid file type")
	require.Equal(t, int64(len(content)), fileResult.Size, "invalid file size")
}

func TestAddFileWithoutID(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)
	ctx.SetUpload(upload)

	name := "file"
	reader, contentType, err := getMultipartFormData(name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var fileResult = &common.File{}
	err = json.Unmarshal(respBody, fileResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, "", fileResult.ID, "invalid file id")
	require.Equal(t, contentMD5, fileResult.Md5, "invalid file md5")
	require.Equal(t, name, fileResult.Name, "invalid file name")
	require.Equal(t, common.FileUploaded, fileResult.Status, "invalid file status")
	require.Equal(t, "text/plain; charset=utf-8", fileResult.Type, "invalid file type")
	require.Equal(t, int64(len(content)), fileResult.Size, "invalid file size")
}

func TestAddFileWithoutUploadInContext(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/file/uploadID", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	context.TestPanic(t, rr, "missing upload from context", func() {
		AddFile(ctx, rr, req)
	})
}

func TestAddFileNoUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	upload.InitializeForTests()

	name := "file"
	reader, contentType, err := getMultipartFormData(name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	rr := ctx.NewRecorder(req)
	context.TestPanic(t, rr, "missing upload form context", func() {
		AddFile(ctx, rr, req)
	})
}

func TestAddFileStatusUploading(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file name"
	file.Status = common.FileUploading
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, fmt.Sprintf("nvalid file status %s, expected missing", common.FileUploading))
}

func TestAddFileStatusUploaded(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file name"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, fmt.Sprintf("invalid file status %s, expected missing", common.FileUploaded))
}

func TestAddFileStatusRemoved(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file name"
	file.Status = common.FileRemoved
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, fmt.Sprintf("invalid file status %s, expected missing", common.FileRemoved))
}

func TestAddFileStatusDeleted(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file name"
	file.Status = common.FileDeleted
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, fmt.Sprintf("invalid file status %s, expected missing", common.FileDeleted))
}

func TestAddFileNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: false}
	createTestUpload(t, ctx, upload)

	req, err := http.NewRequest("POST", "/file/uploadID", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestForbidden(t, rr, "you are not allowed to add file to this upload")
}

func TestAddFileNoMultipartForm(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	req, err := http.NewRequest("POST", "/file/"+upload.ID, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid multipart form : request Content-Type isn't multipart/form-data")
}

func TestAddFileTooManyFiles(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().MaxFilePerUpload = 2

	upload := &common.Upload{IsAdmin: true}

	for range 5 {
		upload.NewFile()
	}
	createTestUpload(t, ctx, upload)

	name := "file"
	reader, contentType, err := getMultipartFormData(name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, "maximum number file per upload reached")
}

func TestAddFileWithFilenameTooLong(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	createTestUpload(t, ctx, upload)
	ctx.SetFile(nil)

	name := make([]byte, 2000)
	for i := range name {
		name[i] = 'x'
	}

	reader, contentType, err := getMultipartFormData(string(name), bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, "is too long")
}

func TestAddFileWithInvalidFileName(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file name"
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormData("blah", bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid file name")
}

func TestAddFileWithEmptyName(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing file name from multipart form")
}

func TestAddFileWithInvalidFieldName(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormDataWithField("blah", file.Name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing file from multipart form")
}

func TestAddFileWithNoFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	buffer := new(bytes.Buffer)
	multipartWriter := multipart.NewWriter(buffer)

	_, err := multipartWriter.CreateFormFile("invalid_form_field", "filename")
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid multipart form")
}

func TestAddFileInvalidMultipartData(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	req, err := http.NewRequest("POST", "/file/"+upload.ID, bytes.NewBuffer([]byte("invalid multipart data")))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid multipart form")
}

//func TestAddFileWithDataBackendError(t *testing.T) {
//	ctx := newTestingContext(common.NewConfiguration())
//	ctx.GetDataBackend().(*data_test.Backend).SetError(errors.New("data backend error"))
//	ctx.SetUploadAdmin(true)
//
//	upload := &common.Upload{}
//	file := upload.NewFile()
//	file.Name = "name"
//
//	createTestUpload(t, ctx, upload)
//	ctx.SetUpload(upload)
//
//	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(content)))
//	require.NoError(t, err, "unable get multipart form data")
//
//	req := getUploadRequest(t, upload, file, reader, contentType)
//
//	rr := ctx.NewRecorder(req)
//	AddFile(ctx, rr, req)
//	context.TestInternalServerError(t, rr, "unable to save file : data backend error")
//}
//
//func TestAddFileWithMetadataBackendError(t *testing.T) {
//	ctx := newTestingContext(common.NewConfiguration())
//	ctx.GetMetadataBackend().(*metadata_test.Backend).SetError(errors.New("metadata backend error"))
//	ctx.SetUploadAdmin(true)
//
//	upload := &common.Upload{}
//	file := upload.NewFile()
//	file.Name = "name"
//
//	createTestUpload(t, ctx, upload)
//	ctx.SetUpload(upload)
//
//	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(content)))
//	require.NoError(t, err, "unable get multipart form data")
//
//	req := getUploadRequest(t, upload, file, reader, contentType)
//
//	rr := ctx.NewRecorder(req)
//	AddFile(ctx, rr, req)
//	context.TestInternalServerError(t, rr, "metadata backend error")
//}

func TestAddFileQuick(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetQuick(true)

	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	name := "file"
	reader, contentType, err := getMultipartFormData(name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	files, err := ctx.GetMetadataBackend().GetFiles(upload.ID)
	require.NoError(t, err, "unable to get upload files")
	require.Len(t, files, 1, "missing file")

	url := fmt.Sprintf("http://127.0.0.1:8080/file/%s/%s/%s\n", upload.ID, files[0].ID, name)

	require.Equal(t, url, string(respBody), "invalid url")
}

func TestAddFileQuickDownloadDomain(t *testing.T) {
	config := common.NewConfiguration()
	config.DownloadDomain = "https://plik.root.gg"
	err := config.Initialize(nil)
	require.NoError(t, err, "config initialization error")

	ctx := newTestingContext(config)
	ctx.SetQuick(true)

	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	name := "file"
	reader, contentType, err := getMultipartFormData(name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	files, err := ctx.GetMetadataBackend().GetFiles(upload.ID)
	require.NoError(t, err, "unable to get upload files")
	require.Len(t, files, 1, "missing file")

	url := fmt.Sprintf("https://plik.root.gg/file/%s/%s/%s\n", upload.ID, files[0].ID, name)

	require.Equal(t, url, string(respBody), "invalid url")
}

func TestAddFileTooBig(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().MaxFileSize = 5

	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	name := "file"
	reader, contentType, err := getMultipartFormData(name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestBadRequest(t, rr, "file too big")
}

func TestAddFileUnlimitedSize(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().MaxFileSize = -1

	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	name := "file"
	reader, contentType, err := getMultipartFormData(name, bytes.NewBuffer([]byte(content)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	context.TestOK(t, rr)
}

func testAddFileMaxUserSize(t *testing.T, ok bool) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser("local", "test")

	if ok {
		user.MaxUserSize = 100000
	} else {
		user.MaxUserSize = 1000
	}

	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	ctx.SetUser(user)

	upload := common.NewUpload()
	upload.User = user.ID
	f := upload.NewFile()
	f.Size = 1000
	f.Status = common.FileUploaded
	err = ctx.GetMetadataBackend().CreateUpload(upload)
	require.NoError(t, err)

	upload = &common.Upload{IsAdmin: true}
	upload.User = user.ID
	f = upload.NewFile()
	f.Name = "file"
	createTestUpload(t, ctx, upload)

	reader, contentType, err := getMultipartFormData(f.Name, bytes.NewBuffer([]byte(content)))
	req, err := http.NewRequest("POST", "/file/"+upload.ID, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)

	if ok {
		context.TestOK(t, rr)
		require.Equal(t, common.FileUploaded, f.Status)
	} else {
		context.TestBadRequest(t, rr, "maximum user upload size reached")
		require.Equal(t, common.FileDeleted, f.Status)
	}
}

func TestAddFileMaxUserSizeOK(t *testing.T) {
	testAddFileMaxUserSize(t, true)
}

func TestAddFileMaxUserSizeKO(t *testing.T) {
	testAddFileMaxUserSize(t, false)
}

func TestAddFileE2EEMimeType(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true, E2EE: "age"}
	file := upload.NewFile()
	file.Name = "secret.txt"
	createTestUpload(t, ctx, upload)

	// Even with the E2EE flag, the preprocessor relies on age header sniffing
	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte("age-encryption.org/v1\n-> scrypt ...\nencrypted payload")))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var fileResult = &common.File{}
	err = json.Unmarshal(respBody, fileResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, "application/octet-stream", fileResult.Type, "E2EE uploads should always be application/octet-stream")
	require.False(t, fileResult.IsText, "E2EE uploads should not be detected as text")
}

func TestAddFilePNGMimeType(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "image.png"
	createTestUpload(t, ctx, upload)

	// PNG magic bytes
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer(pngData))
	require.NoError(t, err, "unable get multipart form data")

	req := getUploadRequest(t, upload, file, reader, contentType)

	rr := ctx.NewRecorder(req)
	AddFile(ctx, rr, req)
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err)

	var fileResult = &common.File{}
	err = json.Unmarshal(respBody, fileResult)
	require.NoError(t, err)

	require.Equal(t, "image/png", fileResult.Type, "PNG should be detected")
	require.False(t, fileResult.IsText, "PNG should not be detected as text")
}

func TestAddFileIsTextDetection(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		content  []byte
		wantMime string
		wantText bool
	}{
		{
			name:     "JSON is text (application/* with text ancestor)",
			fileName: "config.json",
			content:  []byte(`{"name": "plik", "version": "1.0"}`),
			wantMime: "application/json",
			wantText: true,
		},
		{
			name:     "plain text is text",
			fileName: "readme.txt",
			content:  []byte("Hello, world! This is a plain text file."),
			wantMime: "text/plain; charset=utf-8",
			wantText: true,
		},
		{
			name:     "Perl script is text",
			fileName: "script.pl",
			content:  []byte("#!/usr/bin/perl\nuse strict;\nuse warnings;\nprint \"hello\\n\";\n"),
			wantMime: "text/x-perl",
			wantText: true,
		},
		{
			name:     "Shell script is text",
			fileName: "deploy.sh",
			content:  []byte("#!/bin/bash\nset -euo pipefail\necho \"deploying\"\n"),
			wantMime: "text/x-shellscript",
			wantText: true,
		},
		{
			name:     "PNG is not text",
			fileName: "photo.png",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			wantMime: "image/png",
			wantText: false,
		},
		{
			name:     "ZIP is not text",
			fileName: "archive.zip",
			content:  []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00},
			wantMime: "application/zip",
			wantText: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestingContext(common.NewConfiguration())

			upload := &common.Upload{IsAdmin: true}
			file := upload.NewFile()
			file.Name = tt.fileName
			createTestUpload(t, ctx, upload)

			reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer(tt.content))
			require.NoError(t, err, "unable get multipart form data")

			req := getUploadRequest(t, upload, file, reader, contentType)

			rr := ctx.NewRecorder(req)
			AddFile(ctx, rr, req)
			context.TestOK(t, rr)

			respBody, err := io.ReadAll(rr.Body)
			require.NoError(t, err)

			var fileResult = &common.File{}
			err = json.Unmarshal(respBody, fileResult)
			require.NoError(t, err)

			require.Equal(t, tt.wantMime, fileResult.Type, "unexpected MIME type")
			require.Equal(t, tt.wantText, fileResult.IsText, "unexpected IsText value")
		})
	}
}
