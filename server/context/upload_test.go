package context

import (
	"github.com/root-gg/utils"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestUpload_AuthenticationDisabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureAuthentication = common.FeatureDisabled

	upload, err := ctx.CreateUpload(&common.Upload{})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Empty(t, upload.User)

	ctx.user = &common.User{ID: "user"}
	upload, err = ctx.CreateUpload(&common.Upload{})
	require.Errorf(t, err, "authentication is disabled")
	require.Nil(t, upload)
}

func TestUpload_AuthenticationEnabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureAuthentication = common.FeatureEnabled

	upload, err := ctx.CreateUpload(&common.Upload{})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Empty(t, upload.User)

	ctx.user = &common.User{ID: "user"}
	upload, err = ctx.CreateUpload(&common.Upload{})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, ctx.user.ID, upload.User)

}

func TestUpload_AuthenticationForced(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureAuthentication = common.FeatureForced

	upload, err := ctx.CreateUpload(&common.Upload{})
	require.Errorf(t, err, "anonymous uploads are disabled")
	require.Nil(t, upload)

	ctx.user = &common.User{ID: "user"}
	upload, err = ctx.CreateUpload(&common.Upload{})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, ctx.user.ID, upload.User)
}

func TestUpload_OneShotDisabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureOneShot = common.FeatureDisabled

	upload, err := ctx.CreateUpload(&common.Upload{OneShot: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.OneShot)

	upload, err = ctx.CreateUpload(&common.Upload{OneShot: true})
	require.Errorf(t, err, "one shot uploads are disabled")
	require.Nil(t, upload)
}

func TestUpload_OneShotEnabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureOneShot = common.FeatureEnabled

	upload, err := ctx.CreateUpload(&common.Upload{OneShot: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.OneShot)

	upload, err = ctx.CreateUpload(&common.Upload{OneShot: true})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.OneShot)

}

func TestUpload_OneShotForced(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureOneShot = common.FeatureForced

	upload, err := ctx.CreateUpload(&common.Upload{OneShot: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.OneShot)

	upload, err = ctx.CreateUpload(&common.Upload{OneShot: true})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.OneShot)
}

func TestUpload_RemovableDisabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureRemovable = common.FeatureDisabled

	upload, err := ctx.CreateUpload(&common.Upload{Removable: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.Removable)

	upload, err = ctx.CreateUpload(&common.Upload{Removable: true})
	require.Errorf(t, err, "removable uploads are disabled")
	require.Nil(t, upload)
}

func TestUpload_RemovableEnabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureRemovable = common.FeatureEnabled

	upload, err := ctx.CreateUpload(&common.Upload{Removable: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.Removable)

	upload, err = ctx.CreateUpload(&common.Upload{Removable: true})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.Removable)

}

func TestUpload_RemovableForced(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureRemovable = common.FeatureForced

	upload, err := ctx.CreateUpload(&common.Upload{Removable: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.Removable)

	upload, err = ctx.CreateUpload(&common.Upload{Removable: true})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.Removable)
}

func TestUpload_StreamDisabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureStream = common.FeatureDisabled

	upload, err := ctx.CreateUpload(&common.Upload{Stream: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.Stream)

	upload, err = ctx.CreateUpload(&common.Upload{Stream: true})
	require.Errorf(t, err, "streaming uploads are disabled")
	require.Nil(t, upload)
}

func TestUpload_StreamEnabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureStream = common.FeatureEnabled

	upload, err := ctx.CreateUpload(&common.Upload{Stream: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.Stream)

	upload, err = ctx.CreateUpload(&common.Upload{Stream: true})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.Stream)

}

func TestUpload_StreamForced(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureStream = common.FeatureForced

	upload, err := ctx.CreateUpload(&common.Upload{Stream: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.Stream)

	upload, err = ctx.CreateUpload(&common.Upload{Stream: true})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.Stream)
}

func TestUpload_PasswordDisabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeaturePassword = common.FeatureDisabled

	upload, err := ctx.CreateUpload(&common.Upload{Password: ""})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.ProtectedByPassword)

	upload, err = ctx.CreateUpload(&common.Upload{Password: "password"})
	require.Errorf(t, err, "upload password protection is disabled")
	require.Nil(t, upload)
}

func TestUpload_PasswordEnabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeaturePassword = common.FeatureEnabled

	upload, err := ctx.CreateUpload(&common.Upload{Password: ""})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.ProtectedByPassword)

	upload, err = ctx.CreateUpload(&common.Upload{Login: "login", Password: "password"})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.ProtectedByPassword)

	md5sum, err := utils.Md5sum(common.EncodeAuthBasicHeader("login", "password"))
	require.NoError(t, err)

	require.Equal(t, "login", upload.Login)
	require.Equal(t, md5sum, upload.Password)
}

func TestUpload_PasswordForced(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeaturePassword = common.FeatureForced

	upload, err := ctx.CreateUpload(&common.Upload{Password: ""})
	require.Errorf(t, err, "server only accept uploads protected by password")
	require.Nil(t, upload)

	upload, err = ctx.CreateUpload(&common.Upload{Password: "password"})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.ProtectedByPassword)
}

func TestUpload_PasswordDefaultLogin(t *testing.T) {
	ctx := newTestContext()

	params := &common.Upload{Password: "bar"}

	upload, err := ctx.CreateUpload(params)
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, "plik", upload.Login)
	require.NotEqual(t, params.Password, upload.Password)
	require.True(t, upload.ProtectedByPassword)
}

func TestUpload_CommentsDisabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureComments = common.FeatureDisabled

	upload, err := ctx.CreateUpload(&common.Upload{Comments: ""})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Empty(t, upload.Comments)

	upload, err = ctx.CreateUpload(&common.Upload{Comments: "comments"})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Empty(t, upload.Comments)
}

func TestUpload_CommentsEnabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureComments = common.FeatureEnabled

	upload, err := ctx.CreateUpload(&common.Upload{Comments: ""})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Empty(t, upload.Comments)

	upload, err = ctx.CreateUpload(&common.Upload{Comments: "comments"})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, "comments", upload.Comments)

}

func TestUpload_CommentsForced(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureComments = common.FeatureForced

	upload, err := ctx.CreateUpload(&common.Upload{Comments: ""})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Empty(t, upload.Comments)

	upload, err = ctx.CreateUpload(&common.Upload{Comments: "comments"})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.Equal(t, "comments", upload.Comments)
}

func TestUpload_ExtendTTLDisabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureExtendTTL = common.FeatureDisabled

	upload, err := ctx.CreateUpload(&common.Upload{ExtendTTL: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.ExtendTTL)

	upload, err = ctx.CreateUpload(&common.Upload{ExtendTTL: true})
	require.Errorf(t, err, "extend TTL is disabled")
	require.Nil(t, upload)
}

func TestUpload_ExtendTTLEnabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureExtendTTL = common.FeatureEnabled

	upload, err := ctx.CreateUpload(&common.Upload{ExtendTTL: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.False(t, upload.ExtendTTL)

	upload, err = ctx.CreateUpload(&common.Upload{ExtendTTL: true})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.ExtendTTL)

}

func TestUpload_ExtendTTLForced(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureExtendTTL = common.FeatureForced

	upload, err := ctx.CreateUpload(&common.Upload{ExtendTTL: false})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.ExtendTTL)

	upload, err = ctx.CreateUpload(&common.Upload{ExtendTTL: true})
	require.NoError(t, err)
	require.NotNil(t, upload)
	require.True(t, upload.ExtendTTL)
}

func TestCreateUpload(t *testing.T) {
	ctx := newTestContext()
	ctx.sourceIP = net.ParseIP("4.2.4.2")
	ctx.config.FeatureExtendTTL = common.FeatureEnabled

	params := &common.Upload{}
	params.ID = "id"
	params.UploadToken = "token"
	params.IsAdmin = true
	params.ProtectedByPassword = true
	params.RemoteIP = "1.3.3.7"
	params.TTL = 42
	params.ExtendTTL = true
	params.User = "h4x0r"
	params.Token = "h4x0r_token"
	params.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	deadline := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	params.ExpireAt = &deadline
	params.Comments = "comment"

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
	require.True(t, upload.ExtendTTL, "invalid extend TTL status")
	require.False(t, upload.IsAdmin, "invalid admin status")
	require.False(t, upload.ProtectedByPassword, "invalid protected by password status")
	require.Equal(t, params.Comments, upload.Comments, "invalid upload comments")

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
	ctx.config.FeatureAuthentication = common.FeatureEnabled
	ctx.user = &common.User{ID: "user"}
	ctx.token = &common.Token{Token: "token"}

	upload, err := ctx.CreateUpload(&common.Upload{})
	require.NoError(t, err, "unable to create default upload")
	require.NotNil(t, upload)
	require.Equal(t, ctx.user.ID, upload.User, "invalid user")
	require.Equal(t, ctx.token.Token, upload.Token, "invalid token")
}

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

func TestSetTTL(t *testing.T) {
	ctx := newTestContext()
	ctx.config.MaxTTL = 0
	upload, err := ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = newTestContext()
	ctx.config.MaxTTL = 10
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	common.RequireError(t, err, "invalid TTL")
	require.Nil(t, upload)
}

func TestSetTTLDisabled(t *testing.T) {
	ctx := newTestContext()
	ctx.config.FeatureSetTTL = common.FeatureDisabled
	ctx.config.DefaultTTL = 10
	upload, err := ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 10, upload.TTL, "invalid TTL")

	ctx = newTestContext()
	ctx.config.FeatureSetTTL = common.FeatureDisabled
	ctx.config.DefaultTTL = 0
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 0, upload.TTL, "invalid TTL")
}

func TestSetDefaultTTL(t *testing.T) {
	ctx := newTestContext()
	ctx.config.DefaultTTL = 60
	upload, err := ctx.CreateUpload(&common.Upload{TTL: 0})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = newTestContext()
	ctx.config.MaxTTL = 0
	ctx.config.DefaultTTL = 0
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 0})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 0, upload.TTL, "invalid TTL")
}

func TestSetTTLInfinite(t *testing.T) {
	ctx := newTestContext()
	ctx.config.MaxTTL = 0
	upload, err := ctx.CreateUpload(&common.Upload{TTL: -1})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, -1, upload.TTL, "invalid TTL")

	ctx = newTestContext()
	ctx.config.MaxTTL = -1
	upload, err = ctx.CreateUpload(&common.Upload{TTL: -1})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, -1, upload.TTL, "invalid TTL")

	ctx = newTestContext()
	ctx.config.MaxTTL = 10
	upload, err = ctx.CreateUpload(&common.Upload{TTL: -1})
	common.RequireError(t, err, "infinite TTL")
}

func TestSetTTLUser(t *testing.T) {
	ctx := newTestContext()
	ctx.config.MaxTTL = 0
	ctx.config.FeatureAuthentication = common.FeatureEnabled
	ctx.user = &common.User{MaxTTL: 0}
	upload, err := ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = newTestContext()
	ctx.config.MaxTTL = 10
	ctx.config.FeatureAuthentication = common.FeatureEnabled
	ctx.user = &common.User{MaxTTL: 0}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	common.RequireError(t, err, "invalid TTL")

	ctx = newTestContext()
	ctx.config.MaxTTL = 10
	ctx.config.FeatureAuthentication = common.FeatureEnabled
	ctx.user = &common.User{MaxTTL: 100}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = newTestContext()
	ctx.config.MaxTTL = 10
	ctx.config.FeatureAuthentication = common.FeatureEnabled
	ctx.user = &common.User{MaxTTL: -1}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	require.NoError(t, err, "unable to set ttl")
	require.Equal(t, 60, upload.TTL, "invalid TTL")

	ctx = newTestContext()
	ctx.config.MaxTTL = -1
	ctx.config.FeatureAuthentication = common.FeatureEnabled
	ctx.user = &common.User{MaxTTL: 10}
	upload, err = ctx.CreateUpload(&common.Upload{TTL: 60})
	common.RequireError(t, err, "invalid TTL")
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
	ctx.config.FeatureAuthentication = common.FeatureEnabled
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

func TestCheckUserFreeSpaceForUploadNoUser(t *testing.T) {
	ctx := newTestContext()

	params := &common.Upload{}
	for i := 0; i < 10; i++ {
		file := &common.File{Name: "foo", Size: 10 * 1e9}
		params.Files = append(params.Files, file)
	}

	err := ctx.CheckUserFreeSpaceForUpload(params)
	require.NoError(t, err)
}

func TestCheckUserFreeSpaceForUploadNoFiles(t *testing.T) {
	ctx := newTestContext()
	defer setupNewMetadataBackend(ctx)()
	ctx.user = &common.User{MaxUserSize: 1024}
	params := &common.Upload{}

	err := ctx.CheckUserFreeSpaceForUpload(params)
	require.NoError(t, err)
}

func TestCheckUserFreeSpaceForUploadUploadTooBig(t *testing.T) {
	ctx := newTestContext()
	ctx.user = &common.User{MaxUserSize: 1024}

	params := &common.Upload{}
	for i := 0; i < 10; i++ {
		file := &common.File{Name: "foo", Size: 10 * 1e9}
		params.Files = append(params.Files, file)
	}

	err := ctx.CheckUserFreeSpaceForUpload(params)
	common.RequireError(t, err, "maximum user upload size reached")
}

func TestCheckUserFreeSpaceForUploadOK(t *testing.T) {
	testUploadSize(t, true)
}

func TestCheckUserFreeSpaceForUploadKO(t *testing.T) {
	testUploadSize(t, false)
}

func testUploadSize(t *testing.T, ok bool) {
	ctx := newTestContext()
	ctx.user = &common.User{ID: "test", MaxUserSize: 1024}
	defer setupNewMetadataBackend(ctx)()

	// Create
	upload := common.NewUpload()
	upload.User = ctx.user.ID
	file := upload.NewFile()
	file.Status = common.FileUploaded

	if ok {
		file.Size = 900 // 900 + 10 * 10 < 1024
	} else {
		file.Size = 925 // 900 + 10 * 10 > 1024
	}

	err := ctx.GetMetadataBackend().CreateUpload(upload)
	require.NoError(t, err)

	params := &common.Upload{User: ctx.user.ID}
	for i := 0; i < 10; i++ {
		file := &common.File{Name: "foo", Size: 10}
		params.Files = append(params.Files, file)
	}

	err = ctx.CheckUserFreeSpaceForUpload(params)

	if ok {
		require.NoError(t, err)
	} else {
		common.RequireError(t, err, "maximum user upload size reached")
	}
}

func TestCheckUserTotalUploadedSizeOK(t *testing.T) {
	testUserTotalSize(t, true)
}

func TestCheckUserTotalUploadedSizeKO(t *testing.T) {
	testUserTotalSize(t, false)
}

func testUserTotalSize(t *testing.T, ok bool) {
	ctx := newTestContext()
	ctx.user = &common.User{ID: "test", MaxUserSize: 1024}
	defer setupNewMetadataBackend(ctx)()

	// Create
	upload := common.NewUpload()
	upload.User = ctx.user.ID
	file := upload.NewFile()
	file.Status = common.FileUploaded

	if ok {
		file.Size = 900 // 900 + 10 * 10 < 1024
	} else {
		file.Size = 1025 // 900 + 10 * 10 > 1024
	}

	err := ctx.GetMetadataBackend().CreateUpload(upload)
	require.NoError(t, err)

	err = ctx.CheckUserTotalUploadedSize()

	if ok {
		require.NoError(t, err)
	} else {
		common.RequireError(t, err, "maximum user upload size reached")
	}
}

func TestGetUserMaxSizeNoUser(t *testing.T) {
	ctx := newTestContext()
	ctx.GetConfig().MaxUserSize = 10000
	maxUserSize := ctx.GetUserMaxSize()
	require.Equal(t, int64(-1), maxUserSize)
}

func TestGetUserMaxSizeUserUnlimited(t *testing.T) {
	ctx := newTestContext()
	ctx.GetConfig().MaxUserSize = 10000
	ctx.SetUser(&common.User{MaxUserSize: -1})
	maxUserSize := ctx.GetUserMaxSize()
	require.Equal(t, int64(-1), maxUserSize)
}

func TestGetUserMaxSizeUserLimited(t *testing.T) {
	ctx := newTestContext()
	ctx.GetConfig().MaxUserSize = 10000
	ctx.SetUser(&common.User{MaxUserSize: 1000})
	maxUserSize := ctx.GetUserMaxSize()
	require.Equal(t, int64(1000), maxUserSize)
}

func TestGetUserMaxSizeUserServerDefault(t *testing.T) {
	ctx := newTestContext()
	ctx.GetConfig().MaxUserSize = 10000
	ctx.SetUser(&common.User{MaxUserSize: 0})
	maxUserSize := ctx.GetUserMaxSize()
	require.Equal(t, int64(10000), maxUserSize)
}

func TestGetUserMaxSizeUserServerDefaultUnlimited(t *testing.T) {
	ctx := newTestContext()
	ctx.GetConfig().MaxUserSize = -1
	ctx.SetUser(&common.User{MaxUserSize: 0})
	maxUserSize := ctx.GetUserMaxSize()
	require.Equal(t, int64(-1), maxUserSize)
}
