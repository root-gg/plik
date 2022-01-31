package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewUpload(t *testing.T) {
	upload := NewUpload()
	require.NotNil(t, upload)
	require.NotZero(t, upload.ID, "missing upload id")
	require.NotZero(t, upload.UploadToken, "missing upload token")
}

func TestUploadNewFile(t *testing.T) {
	upload := &Upload{}
	upload.NewFile()
	require.NotZero(t, len(upload.Files), "invalid file count")
}

func TestUploadSanitize(t *testing.T) {
	upload := &Upload{}
	upload.NewFile()
	upload.RemoteIP = "ip"
	upload.Login = "login"
	upload.Password = "password"
	upload.UploadToken = "token"
	upload.Token = "token"
	upload.User = "user"

	config := NewConfiguration()
	config.DownloadDomain = "download.domain"
	upload.Sanitize(config)

	require.Zero(t, upload.RemoteIP, "invalid sanitized upload")
	require.Zero(t, upload.Login, "invalid sanitized upload")
	require.Zero(t, upload.Password, "invalid sanitized upload")
	require.Zero(t, upload.UploadToken, "invalid sanitized upload")
	require.Zero(t, upload.Token, "invalid sanitized upload")
	require.Zero(t, upload.UploadToken, "invalid sanitized upload")
	require.Equal(t, config.DownloadDomain, upload.DownloadDomain, "invalid download domain")
}

func TestUploadSanitizeAdmin(t *testing.T) {
	upload := &Upload{}
	upload.NewFile()
	upload.UploadToken = "token"
	upload.IsAdmin = true

	upload.Sanitize(NewConfiguration())

	require.Equal(t, "token", upload.UploadToken, "invalid sanitized upload")
}

func TestUpload_GetFile(t *testing.T) {
	upload := &Upload{}
	file1 := upload.NewFile()
	file1.ID = "id_1"
	file2 := upload.NewFile()
	file2.ID = "id_2"

	f := upload.GetFile(file1.ID)
	require.NotNil(t, f)
	require.Equal(t, file1, f)
}

func TestUpload_GetFileByReference(t *testing.T) {
	upload := &Upload{}
	file1 := upload.NewFile()
	file1.Reference = "1"
	file2 := upload.NewFile()
	file2.Reference = "2"

	f := upload.GetFileByReference(file1.Reference)
	require.NotNil(t, f)
	require.Equal(t, file1, f)
}

func TestUpload_PrepareInsertForTests(t *testing.T) {
	upload := &Upload{}
	upload.NewFile().Name = "file"
	upload.InitializeForTests()

	require.NotZero(t, upload.ID)
	require.NotZero(t, upload.Files[0].ID)
	require.Equal(t, upload.ID, upload.Files[0].UploadID)
}

func TestUpload_IsExpired(t *testing.T) {
	upload := &Upload{}

	deadline := time.Now().Add(time.Hour)
	upload.ExpireAt = &deadline
	require.False(t, upload.IsExpired())

	deadline = time.Now().Add(-time.Hour)
	upload.ExpireAt = &deadline
	require.True(t, upload.IsExpired())
}
