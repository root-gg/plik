package plik

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestGetServerVersion(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	bi, err := pc.GetServerVersion()
	require.NoError(t, err, "unable to get plik server version")
	require.Equal(t, common.GetBuildInfo().Version, bi.Version, "invalid plik server version")
}

func TestGetServerConfig(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().DownloadDomain = "test test test"
	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	config, err := pc.GetServerConfig()
	require.NoError(t, err, "unable to get plik server config")
	require.NotNil(t, config, "unable to get plik server config")
	require.Equal(t, ps.GetConfig().DownloadDomain, config.DownloadDomain, "invalid config value")
}

func TestDefaultUploadParams(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	pc.OneShot = true

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	require.True(t, upload.OneShot, "upload is not oneshot")

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.True(t, upload.Metadata().OneShot, "upload is not oneshot")

	uploadResult, err := pc.GetUpload(upload.ID())
	require.NoError(t, err, "unable to get upload")
	require.Equal(t, upload.ID(), uploadResult.ID(), "upload has not been created")
	require.True(t, uploadResult.OneShot, "upload is not oneshot")
}

func TestUploadParamsOverride(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	pc.OneShot = true

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.OneShot = false

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.False(t, upload.Metadata().OneShot, "upload is not oneshot")

	uploadResult, err := pc.GetUpload(upload.ID())
	require.NoError(t, err, "unable to get upload")
	require.Equal(t, upload.ID(), uploadResult.ID(), "upload has not been created")
	require.False(t, uploadResult.OneShot, "upload is not oneshot")
}

func TestCreateAndGetUpload(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	err = upload.Create()
	require.NoError(t, err, "unable to upload file")
	require.NotNil(t, upload.Metadata(), "upload has not been created")

	uploadResult, err := pc.GetUpload(upload.ID())
	require.NoError(t, err, "unable to upload file")
	require.Equal(t, upload.ID(), uploadResult.ID(), "upload has not been created")

	err = uploadResult.Create()
	require.NoError(t, err, "unable to upload file")
	require.Equal(t, upload.ID(), uploadResult.ID(), "upload has not been created")
}

func TestAddFileToExistingUpload(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	err = upload.Create()
	require.NoError(t, err, "unable to create upload")

	file := upload.AddFileFromReader("filename", bytes.NewBufferString("data"))

	err = upload.Upload()
	require.NoError(t, err, "unable to upload file")
	require.NoError(t, file.Error(), "invalid file error")
	require.Equal(t, file.Metadata().Status, common.FileUploaded, "invalid file status")
}

func TestAddFileToExistingUpload2(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload, _, err := pc.UploadReader("file 1", bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to create upload")

	uploadToken := upload.Metadata().UploadToken

	upload, err = pc.GetUpload(upload.ID())
	require.NoError(t, err, "unable to get upload")

	file2 := upload.AddFileFromReader("file 2", bytes.NewBufferString("data"))

	upload.Metadata().UploadToken = uploadToken
	err = upload.Upload()
	require.NoError(t, err, "unable to upload file")
	require.NoError(t, file2.Error(), "invalid file error")
	require.Equal(t, file2.Metadata().Status, common.FileUploaded, "invalid file status")

	upload, err = pc.GetUpload(upload.ID())
	require.NoError(t, err, "unable to get upload")
	require.Len(t, upload.files, 2, "invalid file count")
}

func TestUploadReader(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", ioutil.NopCloser(bytes.NewBufferString(data)))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Metadata().Files, 0, "invalid file count")

	reader, err := pc.downloadFile(upload.getParams(), file.getParams())
	require.NoError(t, err, "unable to download file")
	content, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, data, string(content), "invalid file content")
}

func TestUploadReadCloser(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", ioutil.NopCloser(bytes.NewBufferString(data)))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Metadata().Files, 0, "invalid file count")

	reader, err := pc.downloadFile(upload.getParams(), file.getParams())
	require.NoError(t, err, "unable to download file")
	content, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, data, string(content), "invalid file content")
}

func TestUploadFiles(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	tmpFile, err := ioutil.TempFile("", "pliktmpfile")
	require.NoError(t, err, "unable to create tmp file")
	defer os.Remove(tmpFile.Name())

	data := "data data data"
	_, err = tmpFile.Write([]byte(data))
	require.NoError(t, err, "unable to write tmp file")
	err = tmpFile.Close()
	require.NoError(t, err, "unable to close tmp file")

	tmpFile2, err := ioutil.TempFile("", "pliktmpfile")
	require.NoError(t, err, "unable to create tmp file")
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile2.Write([]byte(data))
	require.NoError(t, err, "unable to write tmp file")
	err = tmpFile2.Close()
	require.NoError(t, err, "unable to close tmp file")

	upload := pc.NewUpload()

	_, err = upload.AddFileFromPath(tmpFile.Name())
	require.NoError(t, err, "unable to add file")

	_, err = upload.AddFileFromPath(tmpFile2.Name())
	require.NoError(t, err, "unable to add file")

	err = upload.Upload()
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Metadata().Files, 0, "invalid file count")

	for _, file := range upload.Metadata().Files {
		reader, err := pc.downloadFile(upload.Metadata(), file)
		require.NoError(t, err, "unable to download file")
		content, err := ioutil.ReadAll(reader)
		require.NoError(t, err, "unable to read file")
		require.Equal(t, data, string(content), "invalid file content")
	}
}

func TestUploadMultipleFiles(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	for i := 1; i <= 30; i++ {
		filename := fmt.Sprintf("file_%d", i)
		data := fmt.Sprintf("data data data %s", filename)
		upload.AddFileFromReader(filename, bytes.NewBufferString(data))
	}

	err = upload.Upload()
	require.NoError(t, err, "unable to upload files")

	for _, file := range upload.Files() {
		require.Equal(t, file.Metadata().Status, common.FileUploaded, "invalid file status")
		require.NoError(t, file.Error(), "unexpected file error")

		reader, err := pc.downloadFile(upload.Metadata(), file.Metadata())
		require.NoError(t, err, "unable to download file")
		content, err := ioutil.ReadAll(reader)
		require.NoError(t, err, "unable to read file")
		require.Equal(t, fmt.Sprintf("data data data %s", file.Name), string(content), "invalid file content")
	}
}

func TestCreateAndGetUploadFiles(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	for i := 1; i <= 30; i++ {
		filename := fmt.Sprintf("file_%d", i)
		data := fmt.Sprintf("data data data %s", filename)
		upload.AddFileFromReader(filename, bytes.NewBufferString(data))
	}

	err = upload.Upload()
	require.NoError(t, err, "unable to upload files")

	uploadResult, err := pc.GetUpload(upload.ID())
	require.NoError(t, err, "unable to upload file")
	require.Equal(t, upload.ID(), uploadResult.ID(), "upload has not been created")
	require.Len(t, uploadResult.Files(), len(upload.Files()), "file count mismatch")

	for _, file := range uploadResult.Files() {
		require.Equal(t, file.Metadata().Status, common.FileUploaded, "invalid file status")
		require.NoError(t, file.Error(), "unexpected file error")

		reader, err := file.Download()
		require.NoError(t, err, "unable to download file")
		content, err := ioutil.ReadAll(reader)
		require.NoError(t, err, "unable to read file")
		require.Equal(t, fmt.Sprintf("data data data %s", file.Name), string(content), "invalid file content")
	}
}

func TestUploadFile(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	_, _, err = pc.UploadFile("missing_file_name")
	require.Error(t, err, "unable to upload file")
	require.Contains(t, err.Error(), "not found", "unable to upload file")

	_, _, err = pc.UploadFile(".")
	require.Error(t, err, "unable to upload file")
	require.Contains(t, err.Error(), "unhandled file mode", "unable to upload file")

	dummyFilePath := "/tmp/plik.test.dummy.file"
	_, err = os.Create(dummyFilePath)
	require.NoError(t, err, "unable to create file")

	_, _, err = pc.UploadFile(dummyFilePath)
	require.NoError(t, err, "unable to upload file")
}

func TestRemoveFile(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", ioutil.NopCloser(bytes.NewBufferString(data)))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Metadata().Files, 0, "invalid file count")

	_, err = pc.downloadFile(upload.getParams(), file.getParams())
	require.NoError(t, err, "unable to download file")

	err = pc.removeFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err, "unable to remove file")

	_, err = pc.downloadFile(upload.getParams(), file.getParams())
	common.RequireError(t, err, fmt.Sprintf("file %s (%s) is not available", file.Name, file.metadata.ID))
}

func TestRemoveFileNotFound(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()
	err = pc.removeFile(upload, file)
	common.RequireError(t, err, "not found")
}

func TestRemoveFileNoServer(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()
	err := pc.removeFile(upload, file)
	common.RequireError(t, err, "connection refused")
}

func TestDeleteUpload(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", ioutil.NopCloser(bytes.NewBufferString(data)))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Metadata().Files, 0, "invalid file count")

	_, err = file.Download()
	require.NoError(t, err, "unable to download file")

	err = upload.Delete()
	require.NoError(t, err, "unable to remove upload")

	_, err = pc.GetUpload(upload.ID())
	common.RequireError(t, err, "not found")

	_, err = file.Download()
	common.RequireError(t, err, "not found")
}

func TestDeleteUploadNotFound(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := &common.Upload{}
	upload.InitializeForTests()
	err = pc.removeUpload(upload)
	common.RequireError(t, err, "not found")

	upload2 := pc.NewUpload()
	err = upload2.Delete()
	common.RequireError(t, err, "not found")
}

func TestDeleteUploadNoServer(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	upload := &common.Upload{}
	upload.InitializeForTests()
	err := pc.removeUpload(upload)
	common.RequireError(t, err, "connection refused")
}

func TestDownloadArchive(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, _, err := pc.UploadReader("filename", ioutil.NopCloser(bytes.NewBufferString(data)))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Metadata().Files, 0, "invalid file count")

	reader, err := upload.DownloadZipArchive()
	require.NoError(t, err, "unable to download archive")

	defer reader.Close()
	content, err := ioutil.ReadAll(reader)
	require.NoError(t, err, "unable to read archive")

	require.NotEmpty(t, content, "empty archive")
}

func TestGetArchiveNotFound(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := &common.Upload{}
	upload.InitializeForTests()
	_, err = pc.downloadArchive(upload)
	common.RequireError(t, err, "not found")

	upload2 := pc.NewUpload()
	_, err = upload2.DownloadZipArchive()
	common.RequireError(t, err, "not found")
}

func TestGetArchiveNoServer(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	upload := &common.Upload{}
	upload.InitializeForTests()
	_, err := pc.downloadArchive(upload)
	common.RequireError(t, err, "connection refused")
}
