package metadata

import (
	"github.com/root-gg/plik/server/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestGenerateTestData(t *testing.T) {
	b := newTestMetadataBackend()
	defer b.Shutdown()

	setting := &common.Setting{Key: "key1", Value: "val1"}
	err := b.CreateSetting(setting)
	require.NoError(t, err, "unable to create setting")

	admin := common.NewUser(common.ProviderLocal, "admin")
	admin.IsAdmin = true
	admin.Login = "admin"
	admin.Password = "$2a$14$s103BdAMxYV96BunH9hefOEpXnmMzHBmif6tcsQHZkioFeoeHiuRu" // p@ssw0rd
	admin.Name = "Plik Admin"
	admin.Email = "admin@root.gg"
	err = b.CreateUser(admin)
	require.NoError(t, err, "unable to create admin user")

	adminToken := admin.NewToken()
	adminToken.Token = "e78415ed-883e-4d0b-5d0e-fe2d03757520"
	adminToken.Comment = "admin token"
	err = b.CreateToken(adminToken)
	require.NoError(t, err, "unable to create admin token")

	user := common.NewUser(common.ProviderGoogle, "googleuser")
	user.Email = "user@root.gg"
	user.Name = "Plik User"
	err = b.CreateUser(user)
	require.NoError(t, err, "unable to create admin user")

	userToken := user.NewToken()
	userToken.Token = "8cbaeacd-6a3e-4636-4200-607a6e240688"
	userToken.Comment = "user token"
	err = b.CreateToken(userToken)
	require.NoError(t, err, "unable to create admin token")

	// Anonymous Upload
	upload := &common.Upload{}
	upload.ID = "UPLOAD1XXXXXXXXX"
	upload.UploadToken = "UPLOADTOKENXXXXXXXXXXXXXXXXXXXXX"
	upload.RemoteIP = "1.3.3.7"
	upload.DownloadDomain = "https://download.domain"
	upload.IsAdmin = true
	upload.OneShot = true
	upload.Removable = true
	upload.Comments = "愛 الحب 사랑 αγάπη любовь प्यार Սեր माया"
	upload.Login = "foo"
	upload.Password = "bar"
	upload.TTL = 3600
	upload.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	deadline := time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC)
	upload.ExpireAt = &deadline

	file := upload.NewFile()
	file.ID = "FILE1XXXXXXXXXXX"
	file.Size = 42
	file.Md5 = "ccea80b85af4f156af9d4d3b94e91a5e"
	file.Name = "愛愛愛"
	file.BackendDetails = "{foo:\"bar\"}"
	file.Reference = ""
	file.Type = "application/awesome"

	err = b.CreateUpload(upload)
	require.NoError(t, err, "unable to save upload metadata")

	// User Upload
	upload2 := &common.Upload{}
	upload2.ID = "UPLOAD2XXXXXXXXX"
	upload2.UploadToken = "UPLOADTOKENXXXXXXXXXXXXXXXXXXXXX"
	upload2.User = user.ID
	upload.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	file2 := upload2.NewFile()
	file2.ID = "FILE2XXXXXXXXXXX"
	file2.Name = "filename"

	err = b.CreateUpload(upload2)
	require.NoError(t, err, "unable to save upload metadata")

	// User Token Upload
	upload3 := &common.Upload{}
	upload3.ID = "UPLOAD3XXXXXXXXX"
	upload3.UploadToken = "UPLOADTOKENXXXXXXXXXXXXXXXXXXXXX"
	upload3.User = user.ID
	upload3.Token = userToken.Token
	file3 := upload3.NewFile()
	file3.ID = "FILE3XXXXXXXXXXX"
	file3.Name = "filename"

	err = b.CreateUpload(upload3)
	require.NoError(t, err, "unable to save upload metadata")

	// Deleted upload
	upload4 := &common.Upload{}
	upload4.ID = "UPLOAD4XXXXXXXXX"
	upload4.UploadToken = "UPLOADTOKENXXXXXXXXXXXXXXXXXXXXX"
	upload.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC)
	upload.DeletedAt = gorm.DeletedAt{Time: deletedAt, Valid: true}
	file4 := upload2.NewFile()
	file4.ID = "FILE4XXXXXXXXXXX"
	file4.Name = "filename"

	err = b.CreateUpload(upload4)
	require.NoError(t, err, "unable to save upload metadata")

	err = b.Export("/tmp/1.3.2.dump")
	require.NoError(t, err, "unable to export metadata")
}

func TestImportTestData(t *testing.T) {
	b := newTestMetadataBackend()
	defer b.Shutdown()

	err := b.Import("dumps/1.3.2.dump", &ImportOptions{})
	require.NoError(t, err, "import error")
}
