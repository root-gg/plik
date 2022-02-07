package context

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestUpload_TooManyFiles(t *testing.T) {
	ctx := newTestContext()
	ctx.config.MaxFilePerUpload = 1

	params := &common.Upload{}
	params.NewFile()
	params.NewFile()

	upload, err := ctx.CreateUpload(params)
	require.Errorf(t, err, "too many files")
	require.Nil(t, upload)
}

func TestUpload_NoOneShot(t *testing.T) {
	ctx := newTestContext()
	ctx.config.OneShot = false

	upload, err := ctx.CreateUpload(&common.Upload{OneShot: true})
	require.Errorf(t, err, "one shot uploads are are not enabled")
	require.Nil(t, upload)
}

func TestUpload_NoRemovable(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Removable = false

	upload, err := ctx.CreateUpload(&common.Upload{Removable: true})
	require.Errorf(t, err, "removable uploads are are not enabled")
	require.Nil(t, upload)
}

func TestUpload_NoStream(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Stream = false

	upload, err := ctx.CreateUpload(&common.Upload{Stream: true})
	require.Errorf(t, err, "streaming uploads are are not enabled")
	require.Nil(t, upload)
}

func TestUpload_NoBasicAuth(t *testing.T) {
	ctx := newTestContext()
	ctx.config.ProtectedByPassword = false

	upload, err := ctx.CreateUpload(&common.Upload{Login: "login", Password: "password"})
	require.Errorf(t, err, "basic auth uploads are disabled")
	require.Nil(t, upload)
}

func TestCreateUpload(t *testing.T) {
	ctx := newTestContext()
	ctx.sourceIP = net.ParseIP("4.2.4.2")

	params := &common.Upload{}

	params.ID = "id"
	params.UploadToken = "token"
	params.IsAdmin = true
	params.ProtectedByPassword = true
	params.RemoteIP = "1.3.3.7"
	params.TTL = 42
	params.User = "h4x0r"
	params.Token = "h4x0r_token"
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
	file.Status = common.FileUploaded
	file.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	file.Md5 = "md5sum"
	file.BackendDetails = "details"

	upload, err := ctx.CreateUpload(params)
	require.NoError(t, err, "unable to create default upload")
	require.NotNil(t, upload)
	require.NotEmpty(t, upload.ID, "missing upload id")
	require.NotEqual(t, params.ID, upload.ID, "invalid upload id")
	require.Equal(t, params.TTL, upload.TTL, "invalid upload ttl")
	require.Equal(t, "4.2.4.2", upload.RemoteIP, "invalid remote IP")
	require.Empty(t, upload.User, "invalid upload user")
	require.Empty(t, upload.Token, "invalid upload user")
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
	require.Equal(t, common.FileMissing, upload.Files[0].Status, "invalid file status")
	require.Emptyf(t, upload.Files[0].Md5, "invalid file md5 status")
	require.Emptyf(t, upload.Files[0].BackendDetails, "invalid file md5 status")
	require.NotEqual(t, file.CreatedAt, upload.Files[0].CreatedAt, "invalid file created at")

}

func TestCreateUploadCtx(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Authentication = true
	ctx.user = &common.User{ID: "user"}
	ctx.token = &common.Token{Token: "token"}

	upload, err := ctx.CreateUpload(&common.Upload{})
	require.NoError(t, err, "unable to create default upload")
	require.NotNil(t, upload)
	require.Equal(t, ctx.user.ID, upload.User, "invalid user")
	require.Equal(t, ctx.token.Token, upload.Token, "invalid token")
}

func TestCreateUploadAuthenticationDisabled(t *testing.T) {
	ctx := newTestContext()
	ctx.user = &common.User{ID: "user"}
	ctx.token = &common.Token{Token: "token"}

	upload, err := ctx.CreateUpload(&common.Upload{})
	common.RequireError(t, err, "authentication is disabled")
	require.Nil(t, upload)
}

func TestCreateUploadNoAnonymousUpload(t *testing.T) {
	ctx := newTestContext()
	ctx.config.NoAnonymousUploads = true

	upload, err := ctx.CreateUpload(&common.Upload{})
	common.RequireError(t, err, "anonymous uploads are disabled")
	require.Nil(t, upload)
}

func TestCreateUploadTooManyFiles(t *testing.T) {
	ctx := newTestContext()
	ctx.config.MaxFilePerUpload = 2

	params := &common.Upload{}

	for i := 0; i < 10; i++ {
		fileToUpload := &common.File{}
		fileToUpload.Reference = strconv.Itoa(i)
		params.Files = append(params.Files, fileToUpload)
	}

	upload, err := ctx.CreateUpload(params)

	common.RequireError(t, err, "too many files")
	require.Nil(t, upload)
}

func TestSetTTL(t *testing.T) {
	ctx := &Context{config: &common.Configuration{MaxTTL: 0}}
	upload, err := ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = &Context{config: &common.Configuration{MaxTTL: 10}}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	common.RequireError(t, err, "invalid TTL")
	require.Nil(t, upload)
}

func TestSetDefaultTTL(t *testing.T) {
	ctx := &Context{config: &common.Configuration{DefaultTTL: 60}}
	upload, err := ctx.CreateUpload(&common.Upload{TTL: 0})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = &Context{config: &common.Configuration{DefaultTTL: 0}}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 0})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 0, upload.TTL, "invalid TTL")
}

func TestSetTTLInfinite(t *testing.T) {
	ctx := &Context{config: &common.Configuration{MaxTTL: 0}}
	upload, err := ctx.CreateUpload(&common.Upload{TTL: -1})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, -1, upload.TTL, "invalid TTL")

	ctx = &Context{config: &common.Configuration{MaxTTL: -1}}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: -1})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, -1, upload.TTL, "invalid TTL")

	ctx = &Context{config: &common.Configuration{MaxTTL: 10}}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: -1})
	common.RequireError(t, err, "infinite TTL")
}

func TestSetTTLUser(t *testing.T) {
	ctx := &Context{config: &common.Configuration{MaxTTL: 0, Authentication: true}, user: &common.User{MaxTTL: 0}}
	upload, err := ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = &Context{config: &common.Configuration{MaxTTL: 10, Authentication: true}, user: &common.User{MaxTTL: 0}}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	common.RequireError(t, err, "invalid TTL")

	ctx = &Context{config: &common.Configuration{MaxTTL: 10, Authentication: true}, user: &common.User{MaxTTL: 100}}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = &Context{config: &common.Configuration{MaxTTL: 10, Authentication: true}, user: &common.User{MaxTTL: -1}}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = &Context{config: &common.Configuration{MaxTTL: -1, Authentication: true}, user: &common.User{MaxTTL: 10}}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	common.RequireError(t, err, "invalid TTL")
}

func TestCreateWithPasswordWhenPasswordIsNotEnabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.ProtectedByPassword = false

	params := &common.Upload{Login: "foo", Password: "bar"}

	upload, err := ctx.CreateUpload(params)
	common.RequireError(t, err, "password protection is not enabled")
	require.Nil(t, upload)
}

func TestCreateWithBasicAuth(t *testing.T) {
	ctx := newTestContext()

	params := &common.Upload{Login: "foo", Password: "bar"}

	upload, err := ctx.CreateUpload(params)
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, params.Login, upload.Login)
	require.NotEqual(t, params.Password, upload.Password)
	require.True(t, upload.ProtectedByPassword)
}

func TestCreateWithBasicAuthDefaultLogin(t *testing.T) {
	ctx := newTestContext()

	params := &common.Upload{Password: "bar"}

	upload, err := ctx.CreateUpload(params)
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, "plik", upload.Login)
	require.NotEqual(t, params.Password, upload.Password)
	require.True(t, upload.ProtectedByPassword)

}

func TestCreateMissingFilename(t *testing.T) {
	ctx := newTestContext()

	params := &common.Upload{}
	params.NewFile()

	upload, err := ctx.CreateUpload(params)
	common.RequireError(t, err, "missing file name")
	require.Nil(t, upload)
}

func TestCreateWithFilenameTooLong(t *testing.T) {
	ctx := newTestContext()

	params := &common.Upload{}

	file := &common.File{}
	params.Files = append(params.Files, file)
	for i := 0; i < 2048; i++ {
		file.Name += "x"
	}

	upload, err := ctx.CreateUpload(params)
	common.RequireError(t, err, "too long")
	require.Nil(t, upload)
}

func TestCreateWithFileTooBig(t *testing.T) {
	ctx := newTestContext()
	ctx.config.MaxFileSize = 1024

	params := &common.Upload{}

	file := &common.File{Name: "foo", Size: 10 * 1024}
	params.Files = append(params.Files, file)

	upload, err := ctx.CreateUpload(params)
	common.RequireError(t, err, "is too big")
	require.Nil(t, upload)
}

func TestCreateWithFileTooBigUser(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Authentication = true
	ctx.user = &common.User{MaxFileSize: 0}
	ctx.config.MaxFileSize = 1024

	params := &common.Upload{}

	file := &common.File{Name: "foo", Size: 10 * 1024}
	params.Files = append(params.Files, file)

	upload, err := ctx.CreateUpload(params)
	common.RequireError(t, err, "is too big")
	require.Nil(t, upload)

	ctx.user.MaxFileSize = 100 * 1024
	upload, err = ctx.CreateUpload(params)
	require.NoError(t, err, "unable to create upload")

	ctx.config.MaxFileSize = 100 * 1024
	ctx.user.MaxFileSize = 1024
	upload, err = ctx.CreateUpload(params)
	common.RequireError(t, err, "is too big")
	require.Nil(t, upload)
}

func TestCreateFile(t *testing.T) {
	ctx := newTestContext()
	file, err := ctx.CreateFile(&common.Upload{}, &common.File{Name: "foo"})
	common.RequireError(t, err, "upload not initialized")
	require.Nil(t, file)
}
