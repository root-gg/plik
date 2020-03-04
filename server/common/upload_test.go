package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUploadCreate(t *testing.T) {
	upload := &Upload{}
	upload.PrepareInsertForTests()
	require.NotZero(t, upload.ID, "missing id")
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
	upload.Sanitize()

	require.Zero(t, upload.RemoteIP, "invalid sanitized upload")
	require.Zero(t, upload.Login, "invalid sanitized upload")
	require.Zero(t, upload.Password, "invalid sanitized upload")
	require.Zero(t, upload.UploadToken, "invalid sanitized upload")
	require.Zero(t, upload.Token, "invalid sanitized upload")
	require.Zero(t, upload.UploadToken, "invalid sanitized upload")
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

func TestUpload_PrepareInsertTooManyFiles(t *testing.T) {
	config := NewConfiguration()
	config.MaxFilePerUpload = 1

	upload := &Upload{}
	upload.NewFile()
	upload.NewFile()

	err := upload.PrepareInsert(config)
	require.Errorf(t, err, "too many files")
}

func TestUpload_PrepareInsertNoAnonymousUploads(t *testing.T) {
	config := NewConfiguration()
	config.NoAnonymousUploads = true

	upload := &Upload{}

	err := upload.PrepareInsert(config)
	require.Errorf(t, err, "anonymous uploads are disabled")
}

func TestUpload_PrepareInsertNoAuthentication(t *testing.T) {
	config := NewConfiguration()
	config.NoAnonymousUploads = true

	upload := &Upload{}
	upload.User = "user"

	err := upload.PrepareInsert(config)
	require.Errorf(t, err, "authentication is disabled")

	upload = &Upload{}
	upload.Token = "token"

	err = upload.PrepareInsert(config)
	require.Errorf(t, err, "authentication is disabled")
}

func TestUpload_PrepareInsertNoOneShot(t *testing.T) {
	config := NewConfiguration()
	config.OneShot = false

	upload := &Upload{}
	upload.OneShot = true

	err := upload.PrepareInsert(config)
	require.Errorf(t, err, "one shot uploads are not enabled")
}

func TestUpload_PrepareInsertNoRemovable(t *testing.T) {
	config := NewConfiguration()
	config.Removable = false

	upload := &Upload{}
	upload.Removable = true

	err := upload.PrepareInsert(config)
	require.Errorf(t, err, "removable uploads are not enabled")
}

func TestUpload_PrepareInsertNoStream(t *testing.T) {
	config := NewConfiguration()
	config.Stream = false

	upload := &Upload{}
	upload.Stream = true

	err := upload.PrepareInsert(config)
	require.Errorf(t, err, "stream mode is not enabled")
}

func TestUpload_PrepareInsertNoBasicAuth(t *testing.T) {
	config := NewConfiguration()
	config.ProtectedByPassword = false

	upload := &Upload{}
	upload.Login = "login"
	upload.Password = "password"

	err := upload.PrepareInsert(config)
	require.Errorf(t, err, "password protection is not enabled")
}

func TestUpload_PrepareInsertTTL(t *testing.T) {
	config := NewConfiguration()

	upload := &Upload{}
	upload.TTL = -1
	err := upload.PrepareInsert(config)
	require.Errorf(t, err, "cannot set infinite ttl")

	upload = &Upload{}
	upload.TTL = 2592000 + 1
	err = upload.PrepareInsert(config)
	require.Errorf(t, err, "invalid ttl")

	upload = &Upload{}
	upload.TTL = -10
	err = upload.PrepareInsert(config)
	require.Errorf(t, err, "invalid ttl")
}

func TestUpload_PrepareInsert(t *testing.T) {
	config := NewConfiguration()

	upload := &Upload{}
	upload.NewFile().Name = "file"
	err := upload.PrepareInsert(config)
	require.NoError(t, err)
}

func TestUpload_PrepareInsertForTests(t *testing.T) {
	upload := &Upload{}
	upload.NewFile().Name = "file"
	upload.PrepareInsertForTests()

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
