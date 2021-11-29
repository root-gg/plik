package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestBackend_GetUploadStatistics(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 2
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	uploads, files, totalSize, err := b.GetUploadStatistics(nil, nil)
	require.NoError(t, err, "unexpected error")
	require.Equal(t, 100, uploads, "invalid upload count")
	require.Equal(t, 1000, files, "invalid file count")
	require.Equal(t, int64(2000), totalSize, "invalid file size")
}

func TestBackend_GetUploadNoFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		createUpload(t, b, upload)
	}

	uploads, files, totalSize, err := b.GetUploadStatistics(nil, nil)
	require.NoError(t, err, "unexpected error")
	require.Equal(t, 100, uploads, "invalid upload count")
	require.Equal(t, 0, files, "invalid file count")
	require.Equal(t, int64(0), totalSize, "invalid file size")
}

func TestBackend_GetUploadStatistics_User(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 2
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	userID := "user"
	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		upload.User = userID
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 2
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	uploads, files, totalSize, err := b.GetUploadStatistics(&userID, nil)
	require.NoError(t, err, "unexpected error")
	require.Equal(t, 100, uploads, "invalid upload count")
	require.Equal(t, 1000, files, "invalid file count")
	require.Equal(t, int64(2000), totalSize, "invalid file size")
}

func TestBackend_GetUploadStatistics_Token(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 2
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	userID := "user"
	tokenStr := "token"
	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		upload.User = userID
		if i%2 == 0 {
			upload.Token = tokenStr
		}
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 2
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	uploads, files, totalSize, err := b.GetUploadStatistics(&userID, &tokenStr)
	require.NoError(t, err, "unexpected error")
	require.Equal(t, 50, uploads, "invalid upload count")
	require.Equal(t, 500, files, "invalid file count")
	require.Equal(t, int64(1000), totalSize, "invalid file size")
}

func TestBackend_GetUploadStatistics_Anonymous(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 3
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	userID := "user"
	tokenStr := "token"
	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		upload.User = userID
		if i%2 == 0 {
			upload.Token = tokenStr
		}
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 2
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	userID = ""
	tokenStr = ""
	uploads, files, totalSize, err := b.GetUploadStatistics(&userID, &tokenStr)
	require.NoError(t, err, "unexpected error")
	require.Equal(t, 100, uploads, "invalid upload count")
	require.Equal(t, 1000, files, "invalid file count")
	require.Equal(t, int64(3000), totalSize, "invalid file size")
}

func TestBackend_GetUserStatistics(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{User: "user_id", Comments: fmt.Sprintf("%d", i)}
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 2
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	stats, err := b.GetUserStatistics("user_id", nil)
	require.NoError(t, err, "unexpected error")
	require.Equal(t, 100, stats.Uploads, "invalid upload count")
	require.Equal(t, 1000, stats.Files, "invalid file count")
	require.Equal(t, int64(2000), stats.TotalSize, "invalid file size")
}

func TestBackend_GetServerStatistics(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 10; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 2
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	for i := 1; i <= 10; i++ {
		upload := &common.Upload{User: "user_id", Comments: fmt.Sprintf("%d", i)}
		for j := 1; j <= 10; j++ {
			file := upload.NewFile()
			file.Size = 2
			file.Status = common.FileUploaded
		}
		createUpload(t, b, upload)
	}

	stats, err := b.GetServerStatistics()
	require.NoError(t, err, "unexpected error")
	require.Equal(t, 20, stats.Uploads, "invalid upload count")
	require.Equal(t, 200, stats.Files, "invalid file count")
	require.Equal(t, int64(400), stats.TotalSize, "invalid file size")
}
