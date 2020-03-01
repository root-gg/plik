package metadata

import (
	"fmt"
	"testing"
	"time"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func createUpload(t *testing.T, b *Backend, upload *common.Upload) {
	upload.PrepareInsertForTests()
	err := b.CreateUpload(upload)
	require.NoError(t, err, "create upload error : %s", err)
}

func TestBackend_CreateUpload(t *testing.T) {
	b := newTestMetadataBackend()

	upload := &common.Upload{}
	file := upload.NewFile()

	createUpload(t, b, upload)

	require.NotZero(t, upload.ID, "missing upload id")
	require.NotZero(t, upload.CreatedAt, "missing creation date")
	require.NotZero(t, file.ID, "missing file id")
	require.Equal(t, upload.ID, file.UploadID, "missing file id")
	require.NotZero(t, file.CreatedAt, "missing creation date")
}

func TestBackend_GetUpload(t *testing.T) {
	b := newTestMetadataBackend()

	upload := &common.Upload{}
	_ = upload.NewFile()

	createUpload(t, b, upload)

	result, err := b.GetUpload(upload.ID)
	require.NoError(t, err, "get upload error")

	require.Equal(t, upload.ID, result.ID, "invalid upload id")
	require.Zero(t, result.Files, "invalid upload files")
	require.Equal(t, upload.UploadToken, result.UploadToken, "invalid upload token")
}

func TestBackend_GetUpload_NotFound(t *testing.T) {
	b := newTestMetadataBackend()

	upload, err := b.GetUpload("not found")
	require.NoError(t, err, "get upload error")
	require.Nil(t, upload, "upload not nil")
}

func TestBackend_GetUploads_MissingPagingQuery(t *testing.T) {
	b := newTestMetadataBackend()

	_, _, err := b.GetUploads("", "", false, nil)
	require.Error(t, err, "get upload error expected")
}

func TestBackend_DeleteUpload(t *testing.T) {
	b := newTestMetadataBackend()

	upload := &common.Upload{}
	_ = upload.NewFile()

	createUpload(t, b, upload)

	err := b.DeleteUpload(upload.ID)
	require.NoError(t, err, "get upload error")

	upload, err = b.GetUpload(upload.ID)
	require.NoError(t, err, "get upload error")
	require.Nil(t, upload, "upload not nil")
}

func TestBackend_GetUploads(t *testing.T) {
	b := newTestMetadataBackend()

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		upload.NewFile()
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploads("", "", false, common.NewPagingQuery().WithLimit(limit))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := 0; i < limit; i++ {
		require.Equal(t, fmt.Sprintf("%d", 100-i), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test forward cursor
	uploads, cursor, err = b.GetUploads("", "", false, common.NewPagingQuery().WithLimit(limit).WithAfterCursor(*cursor.After))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.NotNil(t, cursor.Before, "invalid nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := 0; i < limit; i++ {
		require.Equal(t, fmt.Sprintf("%d", 100-limit-i), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test backward cursor
	uploads, cursor, err = b.GetUploads("", "", false, common.NewPagingQuery().WithLimit(limit).WithBeforeCursor(*cursor.Before))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := 0; i < limit; i++ {
		require.Equal(t, fmt.Sprintf("%d", 100-i), uploads[i].Comments, "invalid upload sequence")
	}
}

func TestBackend_GetUploadsWithFiles(t *testing.T) {
	b := newTestMetadataBackend()

	upload := &common.Upload{}
	upload.NewFile()
	createUpload(t, b, upload)

	uploads, cursor, err := b.GetUploads("", "", false, common.NewPagingQuery())
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, 1, "invalid upload count")
	require.Len(t, uploads[0].Files, 0, "invalid file count")
	require.Nil(t, cursor.After, "invalid non nil after cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")

	uploads, _, err = b.GetUploads("", "", true, common.NewPagingQuery())
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, 1, "invalid upload count")
	require.Len(t, uploads[0].Files, 1, "invalid file count")

}

func TestBackend_GetUploads_User(t *testing.T) {
	b := newTestMetadataBackend()

	user := &common.User{ID: "user"}

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		if i%10 == 0 {
			upload.User = user.ID
		}
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploads(user.ID, "", false, common.NewPagingQuery().WithLimit(limit))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")

	for i := 0; i < limit; i++ {
		expected := 100 - i*10
		require.Equal(t, fmt.Sprintf("%d", expected), uploads[i].Comments, "invalid upload sequence")
	}
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.Nil(t, cursor.After, "invalid non nil after cursor")
}

func TestBackend_GetUploads_Token(t *testing.T) {
	b := newTestMetadataBackend()

	token := &common.Token{Token: "token"}

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		if i%10 == 0 {
			upload.Token = token.Token
		}
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploads("", token.Token, false, &common.PagingQuery{Limit: &limit})
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")

	for i := 0; i < limit; i++ {
		expected := 100 - i*10
		require.Equal(t, fmt.Sprintf("%d", expected), uploads[i].Comments, "invalid upload sequence")
	}
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.Nil(t, cursor.After, "invalid non nil after cursor")
}

func TestBackend_DeleteExpiredUploads(t *testing.T) {
	b := newTestMetadataBackend()

	upload1 := &common.Upload{}
	createUpload(t, b, upload1)

	upload2 := &common.Upload{}
	createUpload(t, b, upload2)

	deadline2 := time.Now().Add(time.Hour)
	upload2.ExpireAt = &deadline2
	err := b.db.Save(upload2).Error
	require.NoError(t, err, "update upload error")

	upload3 := &common.Upload{}
	createUpload(t, b, upload3)

	deadline3 := time.Now().Add(-time.Hour)
	upload3.ExpireAt = &deadline3
	err = b.db.Save(upload3).Error
	require.NoError(t, err, "update upload error")

	removed, err := b.DeleteExpiredUploads()
	require.Nil(t, err, "delete expired upload error")
	require.Equal(t, 1, removed, "removed expired upload count mismatch")
}

func TestBackend_PurgeDeletedUploads(t *testing.T) {
	b := newTestMetadataBackend()

	upload := &common.Upload{}
	upload.NewFile().Status = common.FileUploaded
	createUpload(t, b, upload)

	purged, err := b.PurgeDeletedUploads()
	require.NoError(t, err, "purge deleted upload error")
	require.Equal(t, 0, purged, "invalid purged count")

	err = b.DeleteUpload(upload.ID)
	require.NoError(t, err, "delete upload error")

	purged, err = b.PurgeDeletedUploads()
	require.NoError(t, err, "purge deleted upload error")

	f := func(file *common.File) error {
		return b.UpdateFileStatus(file, file.Status, common.FileDeleted)
	}
	err = b.ForEachRemovedFile(f)
	require.NoError(t, err, "delete files error")

	purged, err = b.PurgeDeletedUploads()
	require.NoError(t, err, "purge deleted upload error")
	require.Equal(t, 1, purged, "invalid purged count")
}

func TestBackend_ForEachUpload(t *testing.T) {
	b := newTestMetadataBackend()

	upload := &common.Upload{}
	upload.Comments = "foo bar"
	upload.NewFile()
	createUpload(t, b, upload)

	count := 0
	f := func(upload *common.Upload) error {
		count++
		require.Equal(t, "foo bar", upload.Comments, "invalid upload comments")
		return nil
	}
	err := b.ForEachUpload(f)
	require.NoError(t, err, "for each upload error : %s", err)
	require.Equal(t, 1, count, "invalid upload count")

	f = func(upload *common.Upload) error {
		return fmt.Errorf("expected")
	}
	err = b.ForEachUpload(f)
	require.Errorf(t, err, "expected")
}
