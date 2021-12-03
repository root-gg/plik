package common

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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

func TestUpload_TooManyFiles(t *testing.T) {
	config := NewConfiguration()
	config.MaxFilePerUpload = 1

	params := &Upload{}
	params.NewFile()
	params.NewFile()

	upload, err := CreateUpload(config, params)
	require.Errorf(t, err, "too many files")
	require.Nil(t, upload)
}

func TestUpload_NoOneShot(t *testing.T) {
	config := NewConfiguration()
	config.OneShot = false

	upload, err := CreateUpload(config, &Upload{OneShot: true})
	require.Errorf(t, err, "one shot uploads are disabled")
	require.Nil(t, upload)
}

func TestUpload_NoRemovable(t *testing.T) {
	config := NewConfiguration()
	config.Removable = false

	upload, err := CreateUpload(config, &Upload{Removable: true})
	require.Errorf(t, err, "removable uploads are disabled")
	require.Nil(t, upload)
}

func TestUpload_NoStream(t *testing.T) {
	config := NewConfiguration()
	config.Stream = false

	upload, err := CreateUpload(config, &Upload{Stream: true})
	require.Errorf(t, err, "streaming uploads are disabled")
	require.Nil(t, upload)
}

func TestUpload_NoBasicAuth(t *testing.T) {
	config := NewConfiguration()
	config.ProtectedByPassword = false

	upload, err := CreateUpload(config, &Upload{Login: "login", Password: "password"})
	require.Errorf(t, err, "basic auth uploads are disabled")
	require.Nil(t, upload)
}

func TestCreateUpload(t *testing.T) {
	config := NewConfiguration()

	params := &Upload{}

	params.ID = "id"
	params.UploadToken = "token"
	params.IsAdmin = true
	params.ProtectedByPassword = true
	params.RemoteIP = "1.3.3.7"
	params.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	deadline := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	params.ExpireAt = &deadline

	file := params.NewFile()
	file.Name = "name"
	file.ID = "id"
	file.Type = "type"
	file.Size = 1234
	file.UploadID = "upload_id"
	file.Reference = "reference"
	file.Status = FileUploaded
	file.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	file.Md5 = "md5sum"
	file.BackendDetails = "details"

	upload, err := CreateUpload(config, params)
	require.NoError(t, err, "unable to create default upload")
	require.NotNil(t, upload)
	require.NotEmpty(t, upload.ID, "missing upload id")
	require.NotEqual(t, params.ID, upload.ID, "invalid upload id")
	require.Emptyf(t, upload.RemoteIP, "invalid remote IP")
	require.NotEqual(t, params.UploadToken, upload.UploadToken, "invalid upload token")
	require.NotEqual(t, params.CreatedAt, upload.CreatedAt, "invalid created at")
	require.NotEqual(t, params.ExpireAt, upload.ExpireAt, "invalid expired at")
	require.False(t, upload.IsAdmin, "invalid admin status")
	require.False(t, upload.ProtectedByPassword, "invalid protected by password status")
	require.Len(t, upload.Files, 1, "invalid file count")

	require.NotEmpty(t, upload.Files[0].ID, "empty file id")
	require.NotEqual(t, file.ID, upload.Files[0].ID, "invalid file id")
	require.Equal(t, upload.ID, upload.Files[0].UploadID, "invalid file id")
	require.Equal(t, file.Name, upload.Files[0].Name, "invalid file name")
	require.Equal(t, file.Type, upload.Files[0].Type, "invalid file type")
	require.Equal(t, file.Size, upload.Files[0].Size, "invalid file size")
	require.Equal(t, file.Reference, upload.Files[0].Reference, "invalid file reference")
	require.Equal(t, FileMissing, upload.Files[0].Status, "invalid file status")
	require.Emptyf(t, upload.Files[0].Md5, "invalid file md5 status")
	require.Emptyf(t, upload.Files[0].BackendDetails, "invalid file md5 status")
	require.NotEqual(t, file.CreatedAt, upload.Files[0].CreatedAt, "invalid file created at")

}

func TestCreateUploadTooManyFiles(t *testing.T) {
	config := NewConfiguration()
	config.MaxFilePerUpload = 2

	params := &Upload{}

	for i := 0; i < 10; i++ {
		fileToUpload := &File{}
		fileToUpload.Reference = strconv.Itoa(i)
		params.Files = append(params.Files, fileToUpload)
	}

	upload, err := CreateUpload(config, params)

	RequireError(t, err, "too many files")
	require.Nil(t, upload)
}

func TestCreateUploadOneShotWhenOneShotIsDisabled(t *testing.T) {
	config := NewConfiguration()
	config.OneShot = false

	params := &Upload{OneShot: true}

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "one shot uploads are not enabled")
	require.Nil(t, upload)
}

func TestCreateUploadOneShotWhenRemovableIsDisabled(t *testing.T) {
	config := NewConfiguration()
	config.Removable = false

	params := &Upload{Removable: true}

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "removable uploads are not enabled")
	require.Nil(t, upload)
}

func TestCreateUploadStreamWhenStreamIsDisabled(t *testing.T) {
	config := NewConfiguration()
	config.Stream = false

	params := &Upload{Stream: true}

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "stream mode is not enabled")
	require.Nil(t, upload)
}

func TestCreateUploadDefaultTTL(t *testing.T) {
	config := NewConfiguration()
	config.DefaultTTL = 60

	params := &Upload{TTL: 0}

	upload, err := CreateUpload(config, params)
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, 60, upload.TTL)

	require.NotNil(t, upload.CreatedAt)
	require.True(t, upload.CreatedAt.After(time.Now().Add(-1*time.Second)))
	require.True(t, upload.CreatedAt.Before(time.Now().Add(1*time.Second)))

	require.NotNil(t, upload.ExpireAt)
	require.True(t, upload.ExpireAt.After(time.Now().Add(59*time.Second)))
	require.True(t, upload.ExpireAt.Before(time.Now().Add(61*time.Second)))
}

func TestCreateTTLTooLong(t *testing.T) {
	config := NewConfiguration()
	config.MaxTTL = 60

	params := &Upload{TTL: 120}

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "invalid TTL. (maximum allowed")
	require.Nil(t, upload)
}

func TestCreateInvalidNegativeTTL(t *testing.T) {
	config := NewConfiguration()

	params := &Upload{TTL: -10}

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "invalid TTL")
	require.Nil(t, upload)
}

func TestCreateInfiniteTTL(t *testing.T) {
	config := NewConfiguration()
	config.MaxTTL = -1

	params := &Upload{TTL: -1}

	upload, err := CreateUpload(config, params)
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Nil(t, upload.ExpireAt)
}

func TestCreateInvalidInfiniteTTL(t *testing.T) {
	config := NewConfiguration()

	params := &Upload{TTL: -1}

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "cannot set infinite TTL")
	require.Nil(t, upload)
}

func TestCreateWithPasswordWhenPasswordIsNotEnabled(t *testing.T) {
	config := NewConfiguration()
	config.ProtectedByPassword = false

	params := &Upload{Login: "foo", Password: "bar"}

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "password protection is not enabled")
	require.Nil(t, upload)
}

func TestCreateWithBasicAuth(t *testing.T) {
	config := NewConfiguration()

	params := &Upload{Login: "foo", Password: "bar"}

	upload, err := CreateUpload(config, params)
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, params.Login, upload.Login)
	require.NotEqual(t, params.Password, upload.Password)
	require.True(t, upload.ProtectedByPassword)
}

func TestCreateWithBasicAuthDefaultLogin(t *testing.T) {
	config := NewConfiguration()

	params := &Upload{Password: "bar"}

	upload, err := CreateUpload(config, params)
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, "plik", upload.Login)
	require.NotEqual(t, params.Password, upload.Password)
	require.True(t, upload.ProtectedByPassword)

}

func TestCreateMissingFilename(t *testing.T) {
	config := NewConfiguration()

	params := &Upload{}
	params.NewFile()

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "missing file name")
	require.Nil(t, upload)
}

func TestCreateWithFilenameTooLong(t *testing.T) {
	config := NewConfiguration()

	params := &Upload{}

	file := &File{}
	params.Files = append(params.Files, file)
	for i := 0; i < 2048; i++ {
		file.Name += "x"
	}

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "too long")
	require.Nil(t, upload)
}

func TestCreateWithFileTooBig(t *testing.T) {
	config := NewConfiguration()
	config.MaxFileSize = 1024

	params := &Upload{}

	file := &File{Name: "foo", Size: 10 * 1024}
	params.Files = append(params.Files, file)

	upload, err := CreateUpload(config, params)
	RequireError(t, err, "is too big")
	require.Nil(t, upload)
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
