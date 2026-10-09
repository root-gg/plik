package metadata

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/root-gg/plik/server/common"
)

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
		if i == 1 {
			upload.ProtectedByPassword = true
			upload.Removable = true
			upload.OneShot = true
			upload.Stream = true
			upload.ExtendTTL = true
			upload.E2EE = "age"
		}
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
	require.Equal(t, 20, stats.Usage.Lifetime.Uploads, "invalid ever upload count")
	require.Equal(t, 200, stats.Usage.Lifetime.Files, "invalid ever file count")
	require.Equal(t, int64(400), stats.Usage.Lifetime.TotalSize, "invalid ever file size")
	require.Equal(t, 10, stats.AnonymousUploads, "invalid anonymous upload count")
	require.Equal(t, int64(200), stats.AnonymousSize, "invalid anonymous file size")
	require.Equal(t, 1, stats.Usage.Current.Features.PasswordUploads, "invalid password upload count")
	require.Equal(t, 1, stats.Usage.Current.Features.RemovableUploads, "invalid removable upload count")
	require.Equal(t, 1, stats.Usage.Current.Features.OneShotUploads, "invalid one-shot upload count")
	require.Equal(t, 1, stats.Usage.Current.Features.StreamUploads, "invalid stream upload count")
	require.Equal(t, 1, stats.Usage.Current.Features.ExtendTTLUploads, "invalid extend ttl upload count")
	require.Equal(t, 1, stats.Usage.Current.Features.E2EEUploads, "invalid e2ee upload count")
	require.Equal(t, 20, stats.Usage.Current.Features.CommentUploads, "invalid comment upload count")
}

func TestBackend_GetServerStatisticsDownloadWindows(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())
	require.NoError(t, b.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        today,
		EntityType: common.DownloadStatsEntityUpload,
		EntityID:   "upload-today",
		Downloads:  1,
	}))
	require.NoError(t, b.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        today,
		EntityType: common.DownloadStatsEntityFile,
		EntityID:   "file-today",
		Downloads:  100,
	}))
	require.NoError(t, b.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        today.AddDate(0, 0, -3),
		EntityType: common.DownloadStatsEntityUpload,
		EntityID:   "upload-3d",
		Downloads:  2,
	}))
	require.NoError(t, b.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        today.AddDate(0, 0, -8),
		EntityType: common.DownloadStatsEntityUpload,
		EntityID:   "upload-8d",
		Downloads:  4,
	}))
	require.NoError(t, b.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        today.AddDate(0, 0, -31),
		EntityType: common.DownloadStatsEntityUpload,
		EntityID:   "upload-31d",
		Downloads:  8,
	}))

	upload := &common.Upload{Comments: "deleted download still counts"}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	createUpload(t, b, upload)
	require.NoError(t, b.RecordFileDownload(upload, file, 1024, true))
	require.NoError(t, b.RemoveUpload(upload.ID))

	stats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, int64(2), *stats.Usage.Downloads.Today)
	require.Equal(t, int64(4), *stats.Usage.Downloads.Last7Days)
	require.Equal(t, int64(8), *stats.Usage.Downloads.Last30Days)
	require.Equal(t, int64(1), stats.Usage.Downloads.Total)
}

func TestBackend_UserUsageStatsCurrentAndEver(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "stats-user")
	err := b.CreateUser(user)
	require.NoError(t, err)

	upload := &common.Upload{User: user.ID}
	file1 := upload.NewFile()
	file1.Size = 10
	file1.Status = common.FileUploaded
	file2 := upload.NewFile()
	file2.Size = 20
	file2.Status = common.FileUploaded
	createUpload(t, b, upload)

	stats, err := b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Uploads)
	require.Equal(t, 2, stats.Files)
	require.Equal(t, int64(30), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 2, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(30), stats.Usage.Lifetime.TotalSize)
	require.NotNil(t, stats.Usage.StartedAt)
	require.False(t, stats.Usage.StartedAt.IsZero())

	err = b.RemoveFile(file1)
	require.NoError(t, err)

	stats, err = b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Uploads)
	require.Equal(t, 1, stats.Files)
	require.Equal(t, int64(20), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 2, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(30), stats.Usage.Lifetime.TotalSize)

	err = b.RemoveUpload(upload.ID)
	require.NoError(t, err)

	stats, err = b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 0, stats.Uploads)
	require.Equal(t, 0, stats.Files)
	require.Equal(t, int64(0), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 2, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(30), stats.Usage.Lifetime.TotalSize)
}

func TestBackend_TTLDistributionCurrentAndEver(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	ttls := []int{-1, 0, 3599, 3600, 86400, 604800, 2592001}
	var oneHourUpload *common.Upload
	for _, ttl := range ttls {
		upload := &common.Upload{TTL: ttl}
		file := upload.NewFile()
		file.Size = 1
		file.Status = common.FileUploaded
		createUpload(t, b, upload)
		if ttl == 3600 {
			oneHourUpload = upload
		}
	}

	stats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 2, stats.Usage.Current.TTL.NoneUploads)
	require.Equal(t, 1, stats.Usage.Current.TTL.LessThan1HourUploads)
	require.Equal(t, 1, stats.Usage.Current.TTL.OneHourToOneDayUploads)
	require.Equal(t, 1, stats.Usage.Current.TTL.OneDayToSevenDaysUploads)
	require.Equal(t, 1, stats.Usage.Current.TTL.SevenDaysTo30DaysUploads)
	require.Equal(t, 1, stats.Usage.Current.TTL.GreaterThan30DaysUploads)
	require.Equal(t, 2, stats.Usage.Lifetime.TTL.NoneUploads)
	require.Equal(t, 1, stats.Usage.Lifetime.TTL.LessThan1HourUploads)
	require.Equal(t, 1, stats.Usage.Lifetime.TTL.OneHourToOneDayUploads)
	require.Equal(t, 1, stats.Usage.Lifetime.TTL.OneDayToSevenDaysUploads)
	require.Equal(t, 1, stats.Usage.Lifetime.TTL.SevenDaysTo30DaysUploads)
	require.Equal(t, 1, stats.Usage.Lifetime.TTL.GreaterThan30DaysUploads)

	require.NotNil(t, oneHourUpload)
	err = b.RemoveUpload(oneHourUpload.ID)
	require.NoError(t, err)

	stats, err = b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 0, stats.Usage.Current.TTL.OneHourToOneDayUploads)
	require.Equal(t, 1, stats.Usage.Lifetime.TTL.OneHourToOneDayUploads)
}

func TestBackend_FileSizeDistributionBoundaries(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for _, size := range []int64{
		0,
		fileSize1MB - 1,
		fileSize1MB,
		fileSize10MB - 1,
		fileSize10MB,
		fileSize100MB - 1,
		fileSize100MB,
		fileSize1GB - 1,
		fileSize1GB,
		fileSize10GB - 1,
		fileSize10GB,
		fileSize100GB - 1,
		fileSize100GB,
	} {
		upload := &common.Upload{}
		file := upload.NewFile()
		file.Size = size
		file.Status = common.FileUploaded
		createUpload(t, b, upload)
	}

	stats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 2, stats.Usage.Current.FileSizes.LessThan1MBFiles)
	require.Equal(t, 2, stats.Usage.Current.FileSizes.OneMBTo10MBFiles)
	require.Equal(t, 2, stats.Usage.Current.FileSizes.TenMBTo100MBFiles)
	require.Equal(t, 2, stats.Usage.Current.FileSizes.HundredMBTo1GBFiles)
	require.Equal(t, 2, stats.Usage.Current.FileSizes.OneGBTo10GBFiles)
	require.Equal(t, 2, stats.Usage.Current.FileSizes.TenGBTo100GBFiles)
	require.Equal(t, 1, stats.Usage.Current.FileSizes.GreaterThan100GBFiles)
	require.Equal(t, 2, stats.Usage.Lifetime.FileSizes.LessThan1MBFiles)
	require.Equal(t, 2, stats.Usage.Lifetime.FileSizes.OneMBTo10MBFiles)
	require.Equal(t, 2, stats.Usage.Lifetime.FileSizes.TenMBTo100MBFiles)
	require.Equal(t, 2, stats.Usage.Lifetime.FileSizes.HundredMBTo1GBFiles)
	require.Equal(t, 2, stats.Usage.Lifetime.FileSizes.OneGBTo10GBFiles)
	require.Equal(t, 2, stats.Usage.Lifetime.FileSizes.TenGBTo100GBFiles)
	require.Equal(t, 1, stats.Usage.Lifetime.FileSizes.GreaterThan100GBFiles)
}

func TestBackend_FileSizeDistributionCurrentAndEverAfterRemoval(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	removedFileUpload := &common.Upload{}
	removedFile := removedFileUpload.NewFile()
	removedFile.Size = fileSize1MB
	removedFile.Status = common.FileUploaded
	createUpload(t, b, removedFileUpload)

	removedUpload := &common.Upload{}
	removedUploadFile := removedUpload.NewFile()
	removedUploadFile.Size = fileSize10MB
	removedUploadFile.Status = common.FileUploaded
	createUpload(t, b, removedUpload)

	user := common.NewUser(common.ProviderLocal, "file-size-bulk-remove")
	err := b.CreateUser(user)
	require.NoError(t, err)
	bulkRemovedUpload := &common.Upload{User: user.ID}
	bulkRemovedFile := bulkRemovedUpload.NewFile()
	bulkRemovedFile.Size = fileSize100MB
	bulkRemovedFile.Status = common.FileUploaded
	createUpload(t, b, bulkRemovedUpload)

	err = b.RemoveFile(removedFile)
	require.NoError(t, err)
	err = b.RemoveUpload(removedUpload.ID)
	require.NoError(t, err)
	removed, err := b.RemoveUserUploads(user.ID, "")
	require.NoError(t, err)
	require.Equal(t, 1, removed)

	stats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 0, stats.Usage.Current.FileSizes.OneMBTo10MBFiles)
	require.Equal(t, 0, stats.Usage.Current.FileSizes.TenMBTo100MBFiles)
	require.Equal(t, 0, stats.Usage.Current.FileSizes.HundredMBTo1GBFiles)
	require.Equal(t, 1, stats.Usage.Lifetime.FileSizes.OneMBTo10MBFiles)
	require.Equal(t, 1, stats.Usage.Lifetime.FileSizes.TenMBTo100MBFiles)
	require.Equal(t, 1, stats.Usage.Lifetime.FileSizes.HundredMBTo1GBFiles)
}

func TestBackend_SchemaInitStatsColumns(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for _, column := range []string{
		"user_id",
		"token",
		"current_ttl_none_uploads",
		"lifetime_ttl_none_uploads",
		"current_ttl_lt1h_uploads",
		"lifetime_ttl_lt1h_uploads",
		"current_ttl_1h1d_uploads",
		"lifetime_ttl_1h1d_uploads",
		"current_ttl_1d7d_uploads",
		"lifetime_ttl_1d7d_uploads",
		"current_ttl_7d30d_uploads",
		"lifetime_ttl_7d30d_uploads",
		"current_ttl_gt30d_uploads",
		"lifetime_ttl_gt30d_uploads",
		"current_file_size_lt1m_files",
		"lifetime_file_size_lt1m_files",
		"current_file_size_1m10m_files",
		"lifetime_file_size_1m10m_files",
		"current_file_size_10m100m_files",
		"lifetime_file_size_10m100m_files",
		"current_file_size_100m1g_files",
		"lifetime_file_size_100m1g_files",
		"current_file_size_1g10g_files",
		"lifetime_file_size_1g10g_files",
		"current_file_size_10g100g_files",
		"lifetime_file_size_10g100g_files",
		"current_file_size_gt100g_files",
		"lifetime_file_size_gt100g_files",
	} {
		require.True(t, b.db.Migrator().HasColumn(&common.UsageStats{}, column), "missing usage stats column %s", column)
	}
	require.True(t, b.db.Migrator().HasColumn(&common.UsageStats{}, "last_upload_at"), "missing usage stats last upload column")
	require.True(t, b.db.Migrator().HasIndex(&common.File{}, "idx_file_upload_id"), "missing file upload index")
}

func TestBackend_FailedRegularUploadDoesNotIncrementEverFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "failed-upload-user")
	err := b.CreateUser(user)
	require.NoError(t, err)

	upload := &common.Upload{User: user.ID}
	file := upload.NewFile()
	file.Size = 42
	file.Status = common.FileMissing
	createUpload(t, b, upload)

	err = b.UpdateFileStatus(file, common.FileMissing, common.FileUploading)
	require.NoError(t, err)
	file.Size = 42
	err = b.UpdateFileStatus(file, common.FileUploading, common.FileDeleted)
	require.NoError(t, err)

	stats, err := b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 0, stats.Files)
	require.Equal(t, int64(0), stats.TotalSize)
	require.Equal(t, 0, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(0), stats.Usage.Lifetime.TotalSize)
}

func TestBackend_CompletedStreamUploadIncrementsEverFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "stream-upload-user")
	err := b.CreateUser(user)
	require.NoError(t, err)

	upload := &common.Upload{User: user.ID, Stream: true}
	file := upload.NewFile()
	file.Status = common.FileMissing
	createUpload(t, b, upload)

	err = b.UpdateFileStatus(file, common.FileMissing, common.FileUploading)
	require.NoError(t, err)
	file.Status = common.FileDeleted
	file.Size = 42
	err = b.UpdateFile(file, common.FileUploading)
	require.NoError(t, err)

	stats, err := b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 0, stats.Files)
	require.Equal(t, int64(0), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(42), stats.Usage.Lifetime.TotalSize)
}

// TestBackend_ZeroSizeFileCompletionDoesNotDriftCounters reproduces the
// counter-drift/quota-bypass exploit: a client declares a huge file size at
// upload creation, then actually transfers zero bytes. UpdateFile's completion
// call sets file.Size = 0, which used to be silently skipped by GORM's
// struct-based Updates() — leaving the declared (huge) size in the DB row
// while usage counters were credited with the real (zero) size. Removing the
// file/upload later decremented counters using the stale DB size, driving
// current_size arbitrarily negative and defeating quota enforcement
// (checkUserTotalUploadedSize reads GetUserStatistics().TotalSize).
func TestBackend_ZeroSizeFileCompletionDoesNotDriftCounters(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "zero-size-user")
	err := b.CreateUser(user)
	require.NoError(t, err)

	const declaredSize = int64(10 * 1e9) // client-declared size at upload creation

	upload := &common.Upload{User: user.ID}
	fileA := upload.NewFile() // completed then removed individually via RemoveFile
	fileA.Size = declaredSize
	fileA.Status = common.FileMissing
	fileB := upload.NewFile() // completed then removed via RemoveUpload
	fileB.Size = declaredSize
	fileB.Status = common.FileMissing
	createUpload(t, b, upload)

	completeWithEmptyBody := func(file *common.File) {
		err := b.UpdateFileStatus(file, common.FileMissing, common.FileUploading)
		require.NoError(t, err)

		// AddFile completion: the client actually sent zero bytes, so the
		// handler's computed size is 0 even though the declared size was huge.
		file.Size = 0
		file.Status = common.FileUploaded
		err = b.UpdateFile(file, common.FileUploading)
		require.NoError(t, err)
	}

	completeWithEmptyBody(fileA)
	completeWithEmptyBody(fileB)

	dbFileA, err := b.GetFile(fileA.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), dbFileA.Size, "persisted size must match the zero-byte transfer, not the declared size")

	dbFileB, err := b.GetFile(fileB.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), dbFileB.Size, "persisted size must match the zero-byte transfer, not the declared size")

	stats, err := b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), stats.TotalSize, "current size")
	require.Equal(t, int64(0), stats.Usage.Lifetime.TotalSize, "lifetime size")

	serverStats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, int64(0), serverStats.TotalSize)
	require.Equal(t, int64(0), serverStats.Usage.Lifetime.TotalSize)

	// One-shot / explicit file removal: decrements using the metadata row's
	// persisted size (stats_usage.go decrementUsageForUploadedFile).
	err = b.RemoveFile(fileA)
	require.NoError(t, err)

	stats, err = b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), stats.TotalSize, "current size must never go negative")

	// Upload removal: decrements using sum(size) aggregated from the DB
	// (upload.go RemoveUpload / uploadedFileStatsForUploads).
	err = b.RemoveUpload(upload.ID)
	require.NoError(t, err)

	stats, err = b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), stats.TotalSize, "current size must never go negative")
	require.Equal(t, int64(0), stats.Usage.Lifetime.TotalSize, "lifetime size stays exact after removal")

	serverStats, err = b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, int64(0), serverStats.TotalSize, "server current size must never go negative")
	require.Equal(t, int64(0), serverStats.Usage.Lifetime.TotalSize)
}

func TestBackend_UserUsageStatsAnonymous(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Size = 12
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	stats, err := b.GetUserStatistics(common.AnonymousUserUsageStatsID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Uploads)
	require.Equal(t, 1, stats.Files)
	require.Equal(t, int64(12), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(12), stats.Usage.Lifetime.TotalSize)
	require.NotNil(t, stats.Usage.StartedAt)
	require.False(t, stats.Usage.StartedAt.IsZero())

	err = b.RemoveUpload(upload.ID)
	require.NoError(t, err)

	stats, err = b.GetUserStatistics(common.AnonymousUserUsageStatsID, nil)
	require.NoError(t, err)
	require.Equal(t, 0, stats.Uploads)
	require.Equal(t, 0, stats.Files)
	require.Equal(t, int64(0), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(12), stats.Usage.Lifetime.TotalSize)
}

func TestBackend_TokenUsageStatsCurrentAndEver(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "token-stats-user")
	token := user.NewToken()
	err := b.CreateUser(user)
	require.NoError(t, err)

	tokens, _, err := b.GetTokens(user.ID, "", common.NewPagingQuery().WithLimit(1))
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.NotNil(t, tokens[0].Stats)
	require.NotNil(t, tokens[0].Stats.Usage)
	require.NotNil(t, tokens[0].Stats.Usage.StartedAt)
	require.WithinDuration(t, tokens[0].CreatedAt, *tokens[0].Stats.Usage.StartedAt, time.Second)

	upload := &common.Upload{User: user.ID, Token: token.Token}
	file := upload.NewFile()
	file.Size = 12
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	stats, err := b.GetUserStatistics(user.ID, &token.Token)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Uploads)
	require.Equal(t, 1, stats.Files)
	require.Equal(t, int64(12), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(12), stats.Usage.Lifetime.TotalSize)
	require.NotNil(t, stats.Usage.StartedAt)
	require.False(t, stats.Usage.StartedAt.IsZero())

	err = b.RemoveUpload(upload.ID)
	require.NoError(t, err)

	stats, err = b.GetUserStatistics(user.ID, &token.Token)
	require.NoError(t, err)
	require.Equal(t, 0, stats.Uploads)
	require.Equal(t, 0, stats.Files)
	require.Equal(t, int64(0), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(12), stats.Usage.Lifetime.TotalSize)

	deleted, err := b.DeleteToken(token.Token)
	require.NoError(t, err)
	require.True(t, deleted)

	stats, err = b.GetUserStatistics(user.ID, &token.Token)
	require.NoError(t, err)
	require.Equal(t, 0, stats.Uploads)
	require.Equal(t, 0, stats.Usage.Lifetime.Uploads)
}

func TestBackend_RevokedTokenUsageStatsAreNotRecreated(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "revoked-token-stats-user")
	token := user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Token: token.Token}
	file := upload.NewFile()
	file.Size = 12
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	deleted, err := b.DeleteToken(token.Token)
	require.NoError(t, err)
	require.True(t, deleted)

	require.NoError(t, b.RecordFileDownload(upload, file, 1024, true))
	requireTokenUsageStatsRows(t, b, user.ID, token.Token, 0)

	require.NoError(t, b.RemoveUpload(upload.ID))
	requireTokenUsageStatsRows(t, b, user.ID, token.Token, 0)

	userStats, err := b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 0, userStats.Uploads)
	require.Equal(t, 1, userStats.Usage.Lifetime.Uploads)
	require.Equal(t, int64(1), userStats.Usage.Downloads.Total)

	tokenStats, err := b.GetUserStatistics(user.ID, &token.Token)
	require.NoError(t, err)
	require.Equal(t, 0, tokenStats.Uploads)
	require.Equal(t, 0, tokenStats.Usage.Lifetime.Uploads)
	require.Equal(t, int64(0), tokenStats.Usage.Downloads.Total)
}

// TestBackend_DeletedUserUsageStatsAreNotRecreatedByLateDownload is the user-scope
// counterpart of TestBackend_RevokedTokenUsageStatsAreNotRecreated: a best-effort
// download recorded from a pre-stream upload object AFTER the upload was deleted
// (here via DeleteUser, which soft-deletes the upload and folds the user's
// (user,"") usage row into the tombstone before dropping it) must not resurrect
// that usage row or write an orphan daily rollup. The scoped upload UPDATE in the
// recorders matches no row, so recording bails before the rollup and
// usage-increment steps.
func TestBackend_DeletedUserUsageStatsAreNotRecreatedByLateDownload(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "deleted-user-late-download")
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID}
	file := upload.NewFile()
	file.Size = 12
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	deleted, err := b.DeleteUser(user.ID)
	require.NoError(t, err)
	require.True(t, deleted)
	requireTokenUsageStatsRows(t, b, user.ID, "", 0)

	// Late best-effort downloads of the now-deleted upload, replayed from the
	// pre-stream objects. Both must be no-ops.
	require.NoError(t, b.RecordFileDownload(upload, file, 1024, true))
	require.NoError(t, b.RecordArchiveDownload(upload, []*common.File{file}, 2048, true))

	// The (user,"") usage row must NOT be recreated by the late downloads.
	requireTokenUsageStatsRows(t, b, user.ID, "", 0)

	// And no orphan download_stats_daily rows for the deleted upload or its file.
	var rollupRows int64
	require.NoError(t, b.db.Model(&common.DownloadStatsDaily{}).
		Where("entity_id IN ?", []string{upload.ID, file.ID}).
		Count(&rollupRows).Error)
	require.Equal(t, int64(0), rollupRows, "a download of an already-deleted upload must not write daily rollups")
}

func requireTokenUsageStatsRows(t *testing.T, b *Backend, userID string, token string, expected int64) {
	t.Helper()

	var rows int64
	err := b.db.Model(&common.UsageStats{}).Where("user_id = ? AND token = ?", userID, token).Count(&rows).Error
	require.NoError(t, err)
	require.Equal(t, expected, rows)
}

func requireUsageStats(t *testing.T, b *Backend, userID string, token string, wantDownloads int64, wantBytes int64) {
	t.Helper()
	var s common.UsageStats
	err := b.db.Where("user_id = ? AND token = ?", userID, token).First(&s).Error
	require.NoError(t, err, "usage_stats row (user_id=%q, token=%q)", userID, token)
	require.Equal(t, wantDownloads, s.Downloads, "downloads for (user_id=%q, token=%q)", userID, token)
	require.Equal(t, wantBytes, s.DownloadedBytes, "downloaded_bytes for (user_id=%q, token=%q)", userID, token)
}

// requireServerUsageStats asserts the server-scope download event and byte totals
// via sum-on-read (there is no server usage_stats row anymore): the server
// downloads/downloaded_bytes are Σ over the token=” rows.
func requireServerUsageStats(t *testing.T, b *Backend, wantDownloads int64, wantBytes int64) {
	t.Helper()
	stats, err := b.GetServerStatistics()
	require.NoError(t, err, "get server statistics")
	require.Equal(t, wantDownloads, stats.Usage.Downloads.Total, "server downloads (sum-on-read)")
	require.Equal(t, wantBytes, stats.Usage.Downloads.Bytes, "server downloaded_bytes (sum-on-read)")
}

func requireDailyRollup(t *testing.T, b *Backend, entityType string, entityID string, wantDownloads int64, wantBytes int64) {
	t.Helper()
	var s common.DownloadStatsDaily
	err := b.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).First(&s).Error
	require.NoError(t, err, "download_stats_daily row (%s:%s)", entityType, entityID)
	require.Equal(t, wantDownloads, s.Downloads, "rollup downloads for %s:%s", entityType, entityID)
	require.Equal(t, wantBytes, s.Bytes, "rollup bytes for %s:%s", entityType, entityID)
}

func requireDailyRollupAttribution(t *testing.T, b *Backend, entityType string, entityID string, wantUserID string, wantToken string) {
	t.Helper()
	var s common.DownloadStatsDaily
	err := b.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).First(&s).Error
	require.NoError(t, err, "download_stats_daily row (%s:%s)", entityType, entityID)
	require.Equal(t, wantUserID, s.UserID, "rollup user_id for %s:%s", entityType, entityID)
	require.Equal(t, wantToken, s.Token, "rollup token for %s:%s", entityType, entityID)
}

// TestBackend_RecordFileDownloadBytes pins the byte-egress semantics of
// RecordFileDownload: a counted event records the bytes served on every scope
// (user/server/token) and both rollups; a subsequent bytes-only recording
// (mid-range GET) adds bytes everywhere but no event anywhere, and never touches
// the hot upload/file download_count rows.
func TestBackend_RecordFileDownloadBytes(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "bytes-user")
	token := user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Token: token.Token}
	file := upload.NewFile()
	file.Name = "bytes.txt"
	file.Size = 500
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	// A counted event: bytes recorded on every scope and both rollups, event
	// recorded on the hot rows and both rollups.
	require.NoError(t, b.RecordFileDownload(upload, file, 500, true))

	requireUsageStats(t, b, user.ID, "", 1, 500) // user scope
	requireServerUsageStats(t, b, 1, 500)
	requireUsageStats(t, b, user.ID, token.Token, 1, 500) // token scope
	requireDailyRollup(t, b, common.DownloadStatsEntityUpload, upload.ID, 1, 500)
	requireDailyRollup(t, b, common.DownloadStatsEntityFile, file.ID, 1, 500)

	gotUpload, err := b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotUpload.DownloadCount)
	require.Equal(t, int64(500), gotUpload.DownloadedBytes, "upload.downloaded_bytes must accumulate the counted event's bytes")
	gotFile, err := b.GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotFile.DownloadCount)

	// A bytes-only recording (mid-range GET): bytes increase everywhere, no
	// download event anywhere.
	require.NoError(t, b.RecordFileDownload(upload, file, 300, false))

	requireUsageStats(t, b, user.ID, "", 1, 800)
	requireServerUsageStats(t, b, 1, 800)
	requireUsageStats(t, b, user.ID, token.Token, 1, 800)
	requireDailyRollup(t, b, common.DownloadStatsEntityUpload, upload.ID, 1, 800)
	requireDailyRollup(t, b, common.DownloadStatsEntityFile, file.ID, 1, 800)

	// The hot upload/file event counters are untouched by a bytes-only recording,
	// but unlike DownloadCount, the upload row's DownloadedBytes DOES accumulate
	// bytes-only recordings — it mirrors usage_stats.downloaded_bytes exactly,
	// event or not.
	gotUpload, err = b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotUpload.DownloadCount, "bytes-only recording must not bump upload download_count")
	require.Equal(t, int64(800), gotUpload.DownloadedBytes, "bytes-only recording must still bump upload downloaded_bytes")
	gotFile, err = b.GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotFile.DownloadCount, "bytes-only recording must not bump file download_count")
}

// TestBackend_FixtureSeedDownloadedBytes pins the fixture/backfill-only bytes
// seam used by `plikd fakedb` (server/cmd/fakedb.go): it must add bytes to
// every usage_stats scope (user, server, token) without touching the download
// event counter anywhere, and without creating any download_stats_daily
// rollup row (fakedb seeds those separately, via CreateDownloadStatsDaily).
func TestBackend_FixtureSeedDownloadedBytes(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "seed-bytes-user")
	token := user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Token: token.Token}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	require.NoError(t, b.FixtureSeedDownloadedBytes(upload, 4096))

	requireUsageStats(t, b, user.ID, "", 0, 4096) // user scope
	requireServerUsageStats(t, b, 0, 4096)
	requireUsageStats(t, b, user.ID, token.Token, 0, 4096) // token scope

	gotUpload, err := b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.Zero(t, gotUpload.DownloadCount, "FixtureSeedDownloadedBytes must not touch the hot upload row")
	require.Zero(t, gotUpload.DownloadedBytes, "FixtureSeedDownloadedBytes must not touch the upload row's downloaded_bytes either — fakedb sets that column directly before CreateUpload, and folding it in here too would double-count")

	rows := 0
	require.NoError(t, b.ForEachDownloadStatsDaily(func(*common.DownloadStatsDaily) error {
		rows++
		return nil
	}))
	require.Zero(t, rows, "FixtureSeedDownloadedBytes must not create any rollup row")

	// Non-positive/nil inputs are no-ops, not errors.
	require.NoError(t, b.FixtureSeedDownloadedBytes(upload, 0))
	require.NoError(t, b.FixtureSeedDownloadedBytes(upload, -1))
	require.NoError(t, b.FixtureSeedDownloadedBytes(nil, 100))
	requireUsageStats(t, b, user.ID, "", 0, 4096)
}

// TestBackend_RecordFileDownloadBytesAnonymous pins that anonymous uploads
// record downloaded_bytes on the anonymous usage row.
func TestBackend_RecordFileDownloadBytesAnonymous(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "anon.txt"
	file.Size = 250
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	require.NoError(t, b.RecordFileDownload(upload, file, 250, true))

	requireUsageStats(t, b, common.AnonymousUserUsageStatsID, "", 1, 250) // anonymous scope
	requireServerUsageStats(t, b, 1, 250)

	// A bytes-only mid-range recording on the same anonymous upload.
	require.NoError(t, b.RecordFileDownload(upload, file, 40, false))
	requireUsageStats(t, b, common.AnonymousUserUsageStatsID, "", 1, 290)
	requireServerUsageStats(t, b, 1, 290)

	gotUpload, err := b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(290), gotUpload.DownloadedBytes, "anonymous upload's downloaded_bytes must accumulate event and bytes-only recordings alike")
}

// TestBackend_RecordArchiveDownloadPartialFailureRecordsBytes pins the archive
// partial-failure contract: get_archive.go calls RecordArchiveDownload with
// countEvent=false when the zip stream fails mid-transfer, so the bytes
// already served to the client are still recorded — as a bytes-only event,
// exactly like RecordFileDownload's mid-range-GET path — while the event
// counters (download_count, last_downloaded_at, per-file rollups) stay
// untouched. A prior successful archive download seeds a non-zero baseline so
// the assertions prove the event counters are left alone, not merely zero.
func TestBackend_RecordArchiveDownloadPartialFailureRecordsBytes(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "archive-partial-failure-user")
	err := b.CreateUser(user)
	require.NoError(t, err)

	upload := &common.Upload{User: user.ID}
	file1 := upload.NewFile()
	file1.Name = "partial-a.txt"
	file1.Size = 10
	file1.Status = common.FileUploaded
	file2 := upload.NewFile()
	file2.Name = "partial-b.txt"
	file2.Size = 20
	file2.Status = common.FileUploaded
	createUpload(t, b, upload)

	require.NoError(t, b.RecordArchiveDownload(upload, []*common.File{file1, file2}, 300, true))
	requireUsageStats(t, b, user.ID, "", 1, 300)

	const partialBytes = int64(777)
	err = b.RecordArchiveDownload(upload, []*common.File{file1, file2}, partialBytes, false)
	require.NoError(t, err)

	gotUpload, err := b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotUpload.DownloadCount, "a bytes-only archive failure must not increment download_count")
	require.Equal(t, int64(300)+partialBytes, gotUpload.DownloadedBytes, "upload.downloaded_bytes must accumulate the partial archive egress")

	gotFile1, err := b.GetFile(file1.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotFile1.DownloadCount, "a bytes-only archive failure must not touch file download_count")

	requireUsageStats(t, b, user.ID, "", 1, 300+partialBytes)
	requireServerUsageStats(t, b, 1, 300+partialBytes)

	requireDailyRollup(t, b, common.DownloadStatsEntityUpload, upload.ID, 1, 300+partialBytes)
	requireDailyRollup(t, b, common.DownloadStatsEntityFile, file1.ID, 1, 0)
}

func TestBackend_RecordDownloadStatsAndTrending(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "trending-user")
	err := b.CreateUser(user)
	require.NoError(t, err)

	upload := &common.Upload{User: user.ID, Comments: "trending upload"}
	file1 := upload.NewFile()
	file1.Name = "trending-a.txt"
	file1.Size = 10
	file1.Status = common.FileUploaded
	file2 := upload.NewFile()
	file2.Name = "trending-b.txt"
	file2.Size = 20
	file2.Status = common.FileUploaded
	createUpload(t, b, upload)

	err = b.RecordFileDownload(upload, file1, 1024, true)
	require.NoError(t, err)
	err = b.RecordArchiveDownload(upload, []*common.File{file1, file2}, 2048, true)
	require.NoError(t, err)

	gotUpload, err := b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), gotUpload.DownloadCount)
	require.Equal(t, int64(1024+2048), gotUpload.DownloadedBytes, "upload.downloaded_bytes must accumulate both the direct download and the whole-zip archive egress")
	require.NotNil(t, gotUpload.LastDownloadedAt)

	gotFile1, err := b.GetFile(file1.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), gotFile1.DownloadCount)
	require.NotNil(t, gotFile1.LastDownloadedAt)

	gotFile2, err := b.GetFile(file2.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotFile2.DownloadCount)

	uploads, err := b.GetTrendingUploads("1d", "", 10)
	require.NoError(t, err)
	require.NotEmpty(t, uploads)
	require.Equal(t, upload.ID, uploads[0].ID)
	require.Equal(t, int64(2), uploads[0].DownloadCount)
	require.Equal(t, 2, uploads[0].Files)
	require.Equal(t, int64(30), uploads[0].Size)

	files, err := b.GetTrendingFiles("all", 10)
	require.NoError(t, err)
	require.NotEmpty(t, files)
	require.Equal(t, file1.ID, files[0].ID)
	require.Equal(t, int64(2), files[0].DownloadCount)
}

func TestBackend_TrendingWindowsAndEligibility(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	activeUpload := &common.Upload{Comments: "active"}
	activeFile := activeUpload.NewFile()
	activeFile.Name = "active.txt"
	activeFile.Size = 10
	activeFile.Status = common.FileUploaded
	createUpload(t, b, activeUpload)
	require.NoError(t, b.RecordFileDownload(activeUpload, activeFile, 1024, true))

	zeroUpload := &common.Upload{Comments: "zero"}
	zeroFile := zeroUpload.NewFile()
	zeroFile.Name = "zero.txt"
	zeroFile.Status = common.FileUploaded
	createUpload(t, b, zeroUpload)

	deletedUpload := &common.Upload{Comments: "deleted"}
	deletedFile := deletedUpload.NewFile()
	deletedFile.Name = "deleted.txt"
	deletedFile.Status = common.FileUploaded
	createUpload(t, b, deletedUpload)
	require.NoError(t, b.RecordFileDownload(deletedUpload, deletedFile, 1024, true))
	require.NoError(t, b.RemoveUpload(deletedUpload.ID))

	removedFileUpload := &common.Upload{Comments: "removed file"}
	removedFile := removedFileUpload.NewFile()
	removedFile.Name = "removed.txt"
	removedFile.Status = common.FileUploaded
	createUpload(t, b, removedFileUpload)
	require.NoError(t, b.RecordFileDownload(removedFileUpload, removedFile, 1024, true))
	require.NoError(t, b.RemoveFile(removedFile))

	uploads, err := b.GetTrendingUploads("all", "", 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, uploads, activeUpload.ID)
	requireContainsTrendingID(t, uploads, removedFileUpload.ID)
	requireNotContainsTrendingID(t, uploads, zeroUpload.ID)
	requireNotContainsTrendingID(t, uploads, deletedUpload.ID)

	files, err := b.GetTrendingFiles("1d", 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, files, activeFile.ID)
	requireNotContainsTrendingID(t, files, zeroFile.ID)
	requireNotContainsTrendingID(t, files, deletedFile.ID)
	requireNotContainsTrendingID(t, files, removedFile.ID)
}

// TestBackend_TrendingUploadsTieBreakerOrder pins the documented deterministic
// order for the "all" window (getTrendingUploadsAll):
// download_count desc, last_downloaded_at desc, id asc. All three uploads
// below tie on download_count so the ordering can only come from the two
// documented tie-breakers, never from the counts themselves.
func TestBackend_TrendingUploadsTieBreakerOrder(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	older := time.Now().Add(-time.Hour)
	newer := time.Now()

	// "newest" breaks the count tie outright via a later last_downloaded_at.
	// "tie-a-older" and "tie-z-older" share both download_count AND
	// last_downloaded_at, so only the id-asc tie-breaker can separate them.
	newest := &common.Upload{ID: "tie-newest", Comments: "newest"}
	newest.NewFile().Status = common.FileUploaded
	newest.DownloadCount = 5
	newest.LastDownloadedAt = &newer
	createUpload(t, b, newest)

	tieZ := &common.Upload{ID: "tie-z-older", Comments: "z"}
	tieZ.NewFile().Status = common.FileUploaded
	tieZ.DownloadCount = 5
	tieZ.LastDownloadedAt = &older
	createUpload(t, b, tieZ)

	tieA := &common.Upload{ID: "tie-a-older", Comments: "a"}
	tieA.NewFile().Status = common.FileUploaded
	tieA.DownloadCount = 5
	tieA.LastDownloadedAt = &older
	createUpload(t, b, tieA)

	uploads, err := b.GetTrendingUploads("all", "", 10)
	require.NoError(t, err)
	require.Len(t, uploads, 3)
	require.Equal(t, newest.ID, uploads[0].ID, "a later last_downloaded_at must win the download_count tie")
	require.Equal(t, tieA.ID, uploads[1].ID, "equal count and last_downloaded_at must fall back to id asc")
	require.Equal(t, tieZ.ID, uploads[2].ID)
}

// TestBackend_TrendingFilesTieBreakerOrder is the file-scoped equivalent of
// TestBackend_TrendingUploadsTieBreakerOrder: getTrendingFilesAll orders by
// files.download_count desc, files.last_downloaded_at desc, files.id asc.
func TestBackend_TrendingFilesTieBreakerOrder(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	older := time.Now().Add(-time.Hour)
	newer := time.Now()

	newUpload := func(fileID string) *common.Upload {
		upload := &common.Upload{}
		file := upload.NewFile()
		file.ID = fileID
		file.Status = common.FileUploaded
		return upload
	}

	newest := newUpload("tie-file-newest")
	newest.Files[0].DownloadCount = 5
	newest.Files[0].LastDownloadedAt = &newer
	createUpload(t, b, newest)

	tieZ := newUpload("tie-file-z-older")
	tieZ.Files[0].DownloadCount = 5
	tieZ.Files[0].LastDownloadedAt = &older
	createUpload(t, b, tieZ)

	tieA := newUpload("tie-file-a-older")
	tieA.Files[0].DownloadCount = 5
	tieA.Files[0].LastDownloadedAt = &older
	createUpload(t, b, tieA)

	files, err := b.GetTrendingFiles("all", 10)
	require.NoError(t, err)
	require.Len(t, files, 3)
	require.Equal(t, newest.Files[0].ID, files[0].ID, "a later last_downloaded_at must win the download_count tie")
	require.Equal(t, tieA.Files[0].ID, files[1].ID, "equal count and last_downloaded_at must fall back to id asc")
	require.Equal(t, tieZ.Files[0].ID, files[2].ID)
}

func TestBackend_TrendingDailyWindows(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{Comments: "old daily"}
	file := upload.NewFile()
	file.Name = "old-daily.txt"
	file.Size = 42
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	today := statsDay(time.Now())
	require.NoError(t, b.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        today.AddDate(0, 0, -8),
		EntityType: common.DownloadStatsEntityUpload,
		EntityID:   upload.ID,
		Downloads:  3,
	}))
	require.NoError(t, b.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        today.AddDate(0, 0, -8),
		EntityType: common.DownloadStatsEntityFile,
		EntityID:   file.ID,
		Downloads:  3,
	}))

	uploads, err := b.GetTrendingUploads("7d", "", 10)
	require.NoError(t, err)
	requireNotContainsTrendingID(t, uploads, upload.ID)

	files, err := b.GetTrendingFiles("7d", 10)
	require.NoError(t, err)
	requireNotContainsTrendingID(t, files, file.ID)

	uploads, err = b.GetTrendingUploads("30d", "", 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, uploads, upload.ID)
	require.Equal(t, int64(3), trendingItemByID(t, uploads, upload.ID).DownloadCount)

	files, err = b.GetTrendingFiles("30d", 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, files, file.ID)
	require.Equal(t, int64(3), trendingItemByID(t, files, file.ID).DownloadCount)
}

// TestBackend_TrendingUploadsSortByDownloadedBytes pins that the new sort
// dimension ranks by bytes served rather than download count, and that
// DownloadedBytes/DownloadCount are BOTH populated regardless of which sort is
// requested.
func TestBackend_TrendingUploadsSortByDownloadedBytes(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// "big" has fewer downloads but far more bytes served; "many" has more
	// downloads but fewer bytes. The two sorts must disagree on ranking.
	big := &common.Upload{Comments: "big"}
	bigFile := big.NewFile()
	bigFile.Status = common.FileUploaded
	bigFile.Size = 10_000_000
	createUpload(t, b, big)
	require.NoError(t, b.RecordFileDownload(big, bigFile, 9_000_000, true))
	require.NoError(t, b.RecordFileDownload(big, bigFile, 9_000_000, true))

	many := &common.Upload{Comments: "many"}
	manyFile := many.NewFile()
	manyFile.Status = common.FileUploaded
	manyFile.Size = 100
	createUpload(t, b, many)
	for range 5 {
		require.NoError(t, b.RecordFileDownload(many, manyFile, 100, true))
	}

	byDownloads, err := b.GetTrendingUploads("all", "", 10)
	require.NoError(t, err)
	require.Equal(t, many.ID, byDownloads[0].ID, "default sort ranks by download count")
	require.Equal(t, int64(5), byDownloads[0].DownloadCount)
	require.Equal(t, int64(500), byDownloads[0].DownloadedBytes, "downloadedBytes must be populated even when sorting by downloads")

	byBytes, err := b.GetTrendingUploads("all", TrendingSortDownloadedBytes, 10)
	require.NoError(t, err)
	require.Equal(t, big.ID, byBytes[0].ID, "downloadedBytes sort must rank by bytes served, not download count")
	require.Equal(t, int64(18_000_000), byBytes[0].DownloadedBytes)
	require.Equal(t, int64(2), byBytes[0].DownloadCount, "downloadCount must stay populated even when sorting by bytes")

	// Windowed mode must agree with "all" mode on both metrics/orderings for
	// same-day seed data.
	windowedByDownloads, err := b.GetTrendingUploads("1d", "", 10)
	require.NoError(t, err)
	require.Equal(t, many.ID, windowedByDownloads[0].ID)
	require.Equal(t, int64(500), windowedByDownloads[0].DownloadedBytes)

	windowedByBytes, err := b.GetTrendingUploads("1d", TrendingSortDownloadedBytes, 10)
	require.NoError(t, err)
	require.Equal(t, big.ID, windowedByBytes[0].ID, "windowed downloadedBytes sort must also rank by bytes served")
	require.Equal(t, int64(18_000_000), windowedByBytes[0].DownloadedBytes)
	require.Equal(t, int64(2), windowedByBytes[0].DownloadCount)
}

// TestBackend_TrendingUploadsSortOmitsZeroForChosenMetric pins that the
// zero-value omission rule follows the SELECTED metric, not always downloads:
// an upload with downloads but zero bytes served (e.g. a 0-byte range request)
// must be omitted only when sorting by downloadedBytes.
func TestBackend_TrendingUploadsSortOmitsZeroForChosenMetric(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	zeroBytes := &common.Upload{Comments: "zero bytes"}
	zeroBytesFile := zeroBytes.NewFile()
	zeroBytesFile.Status = common.FileUploaded
	createUpload(t, b, zeroBytes)
	require.NoError(t, b.RecordFileDownload(zeroBytes, zeroBytesFile, 0, true))

	byDownloadsAll, err := b.GetTrendingUploads("all", "", 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, byDownloadsAll, zeroBytes.ID)

	byBytesAll, err := b.GetTrendingUploads("all", TrendingSortDownloadedBytes, 10)
	require.NoError(t, err)
	requireNotContainsTrendingID(t, byBytesAll, zeroBytes.ID)

	byDownloadsWindow, err := b.GetTrendingUploads("1d", "", 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, byDownloadsWindow, zeroBytes.ID)

	byBytesWindow, err := b.GetTrendingUploads("1d", TrendingSortDownloadedBytes, 10)
	require.NoError(t, err)
	requireNotContainsTrendingID(t, byBytesWindow, zeroBytes.ID)
}

// TestBackend_TrendingWindowsAndEligibilityByDownloadedBytes extends
// TestBackend_TrendingWindowsAndEligibility's deleted/expired-upload exclusion
// invariant to the downloadedBytes sort.
func TestBackend_TrendingWindowsAndEligibilityByDownloadedBytes(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	activeUpload := &common.Upload{Comments: "active"}
	activeFile := activeUpload.NewFile()
	activeFile.Name = "active.txt"
	activeFile.Size = 10
	activeFile.Status = common.FileUploaded
	createUpload(t, b, activeUpload)
	require.NoError(t, b.RecordFileDownload(activeUpload, activeFile, 1024, true))

	deletedUpload := &common.Upload{Comments: "deleted"}
	deletedFile := deletedUpload.NewFile()
	deletedFile.Name = "deleted.txt"
	deletedFile.Status = common.FileUploaded
	createUpload(t, b, deletedUpload)
	require.NoError(t, b.RecordFileDownload(deletedUpload, deletedFile, 1024, true))
	require.NoError(t, b.RemoveUpload(deletedUpload.ID))

	uploads, err := b.GetTrendingUploads("all", TrendingSortDownloadedBytes, 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, uploads, activeUpload.ID)
	requireNotContainsTrendingID(t, uploads, deletedUpload.ID)
}

// TestBackend_GetUserTrendingUploadsScoping pins the user-scoped variant:
// results are restricted to uploads.user = userID (excluding other owners'
// uploads and anonymous uploads), in both "all" and windowed modes, while the
// unscoped admin query keeps returning everyone's uploads unaffected.
func TestBackend_GetUserTrendingUploadsScoping(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	owner := common.NewUser(common.ProviderLocal, "trending-owner")
	require.NoError(t, b.CreateUser(owner))
	other := common.NewUser(common.ProviderLocal, "trending-other")
	require.NoError(t, b.CreateUser(other))

	mine := &common.Upload{User: owner.ID, Comments: "mine"}
	mineFile := mine.NewFile()
	mineFile.Status = common.FileUploaded
	createUpload(t, b, mine)
	require.NoError(t, b.RecordFileDownload(mine, mineFile, 1024, true))

	theirs := &common.Upload{User: other.ID, Comments: "theirs"}
	theirsFile := theirs.NewFile()
	theirsFile.Status = common.FileUploaded
	createUpload(t, b, theirs)
	require.NoError(t, b.RecordFileDownload(theirs, theirsFile, 2048, true))

	anon := &common.Upload{Comments: "anon"}
	anonFile := anon.NewFile()
	anonFile.Status = common.FileUploaded
	createUpload(t, b, anon)
	require.NoError(t, b.RecordFileDownload(anon, anonFile, 4096, true))

	itemsAll, err := b.GetUserTrendingUploads(owner.ID, "all", "", 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, itemsAll, mine.ID)
	requireNotContainsTrendingID(t, itemsAll, theirs.ID)
	requireNotContainsTrendingID(t, itemsAll, anon.ID)

	itemsWindow, err := b.GetUserTrendingUploads(owner.ID, "1d", "", 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, itemsWindow, mine.ID)
	requireNotContainsTrendingID(t, itemsWindow, theirs.ID)
	requireNotContainsTrendingID(t, itemsWindow, anon.ID)

	itemsBytesSort, err := b.GetUserTrendingUploads(owner.ID, "all", TrendingSortDownloadedBytes, 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, itemsBytesSort, mine.ID)
	requireNotContainsTrendingID(t, itemsBytesSort, theirs.ID)
	requireNotContainsTrendingID(t, itemsBytesSort, anon.ID)

	// Scoping is additive, not a replacement of the unscoped admin query.
	all, err := b.GetTrendingUploads("all", "", 10)
	require.NoError(t, err)
	requireContainsTrendingID(t, all, mine.ID)
	requireContainsTrendingID(t, all, theirs.ID)
	requireContainsTrendingID(t, all, anon.ID)
}

func TestBackend_ForEachDownloadStatsDaily(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	day := statsDay(time.Now())
	require.NoError(t, b.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        day,
		EntityType: common.DownloadStatsEntityUpload,
		EntityID:   "upload-1",
		Downloads:  1,
	}))
	require.NoError(t, b.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        day,
		EntityType: common.DownloadStatsEntityFile,
		EntityID:   "file-1",
		Downloads:  2,
	}))

	seen := map[string]int64{}
	err := b.ForEachDownloadStatsDaily(func(stats *common.DownloadStatsDaily) error {
		seen[stats.EntityType+":"+stats.EntityID] = stats.Downloads
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), seen[common.DownloadStatsEntityUpload+":upload-1"])
	require.Equal(t, int64(2), seen[common.DownloadStatsEntityFile+":file-1"])

	stop := errors.New("stop")
	err = b.ForEachDownloadStatsDaily(func(stats *common.DownloadStatsDaily) error {
		return stop
	})
	require.ErrorIs(t, err, stop)
}

func TestBackend_DeleteExpiredDownloadStatsDaily(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())
	rows := []*common.DownloadStatsDaily{
		{Day: today, EntityType: common.DownloadStatsEntityUpload, EntityID: "today", Downloads: 1},
		{Day: today.AddDate(0, 0, -29), EntityType: common.DownloadStatsEntityUpload, EntityID: "window-edge", Downloads: 2},
		{Day: today.AddDate(0, 0, -30), EntityType: common.DownloadStatsEntityUpload, EntityID: "retention-edge", Downloads: 4},
		{Day: today.AddDate(0, 0, -31), EntityType: common.DownloadStatsEntityUpload, EntityID: "old-upload", Downloads: 8},
		{Day: today.AddDate(0, 0, -31), EntityType: common.DownloadStatsEntityFile, EntityID: "old-file", Downloads: 16},
	}
	for _, row := range rows {
		require.NoError(t, b.CreateDownloadStatsDaily(row))
	}

	deleted, err := b.DeleteExpiredDownloadStatsDaily()
	require.NoError(t, err)
	require.Equal(t, 2, deleted)

	seen := map[string]bool{}
	err = b.ForEachDownloadStatsDaily(func(stats *common.DownloadStatsDaily) error {
		seen[stats.EntityID] = true
		return nil
	})
	require.NoError(t, err)
	require.True(t, seen["today"])
	require.True(t, seen["window-edge"])
	require.True(t, seen["retention-edge"])
	require.False(t, seen["old-upload"])
	require.False(t, seen["old-file"])

	stats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, int64(1), *stats.Usage.Downloads.Today)
	require.Equal(t, int64(1), *stats.Usage.Downloads.Last7Days)
	require.Equal(t, int64(3), *stats.Usage.Downloads.Last30Days)
}

func TestBackend_DeleteExpiredDownloadStatsDailyEmpty(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	deleted, err := b.DeleteExpiredDownloadStatsDaily()
	require.NoError(t, err)
	require.Equal(t, 0, deleted)
}

// TestBackend_RecordDailyDownloadsAttribution pins that both RecordFileDownload
// and RecordArchiveDownload write user_id/token verbatim from the upload onto
// both the upload-entity and file-entity rollup rows, and that an anonymous
// upload's "" User is stored as-is rather than translated to the usage_stats
// sentinel (common.AnonymousUserUsageStatsID applies only to usage_stats,
// never to download_stats_daily).
func TestBackend_RecordDailyDownloadsAttribution(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "attribution-user")
	token := user.NewToken()
	createUser(t, b, user)

	owned := &common.Upload{User: user.ID, Token: token.Token}
	ownedFile1 := owned.NewFile()
	ownedFile1.Status = common.FileUploaded
	ownedFile2 := owned.NewFile()
	ownedFile2.Status = common.FileUploaded
	createUpload(t, b, owned)

	require.NoError(t, b.RecordFileDownload(owned, ownedFile1, 10, true))
	require.NoError(t, b.RecordArchiveDownload(owned, []*common.File{ownedFile1, ownedFile2}, 20, true))

	requireDailyRollupAttribution(t, b, common.DownloadStatsEntityUpload, owned.ID, user.ID, token.Token)
	requireDailyRollupAttribution(t, b, common.DownloadStatsEntityFile, ownedFile1.ID, user.ID, token.Token)
	requireDailyRollupAttribution(t, b, common.DownloadStatsEntityFile, ownedFile2.ID, user.ID, token.Token)

	anonymous := &common.Upload{}
	anonymousFile := anonymous.NewFile()
	anonymousFile.Status = common.FileUploaded
	createUpload(t, b, anonymous)

	require.NoError(t, b.RecordFileDownload(anonymous, anonymousFile, 5, true))
	require.NoError(t, b.RecordArchiveDownload(anonymous, []*common.File{anonymousFile}, 5, true))

	requireDailyRollupAttribution(t, b, common.DownloadStatsEntityUpload, anonymous.ID, "", "")
	requireDailyRollupAttribution(t, b, common.DownloadStatsEntityFile, anonymousFile.ID, "", "")
}

// TestBackend_RecordDailyDownloadsAttributionImmutableOnConflict pins that a
// second recordDailyDownloads call for the same (day, entity_type, entity_id)
// bucket never overwrites the attribution set by the first insert. In
// production an entity's owning upload cannot change between calls, but the
// on-conflict update map intentionally omits user_id/token, so attribution is
// first-writer-wins even in principle.
func TestBackend_RecordDailyDownloadsAttributionImmutableOnConflict(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	day := statsDay(time.Now())
	err := b.db.Transaction(func(tx *gorm.DB) error {
		return b.recordDailyDownloads(tx, day, common.DownloadStatsEntityUpload, "conflict-upload", "user-a", "token-a", 1, 100)
	})
	require.NoError(t, err)

	err = b.db.Transaction(func(tx *gorm.DB) error {
		return b.recordDailyDownloads(tx, day, common.DownloadStatsEntityUpload, "conflict-upload", "user-b", "token-b", 1, 50)
	})
	require.NoError(t, err)

	var s common.DownloadStatsDaily
	err = b.db.Where("entity_type = ? AND entity_id = ?", common.DownloadStatsEntityUpload, "conflict-upload").First(&s).Error
	require.NoError(t, err)
	require.Equal(t, "user-a", s.UserID, "attribution must be immutable on conflict")
	require.Equal(t, "token-a", s.Token, "attribution must be immutable on conflict")
	require.Equal(t, int64(2), s.Downloads, "downloads must still increment on conflict")
	require.Equal(t, int64(150), s.Bytes, "bytes must still increment on conflict")
}

// TestBackend_GetUserActivityStatsDailyDownloads pins the exact per-day shape of
// the activity series' download measures: dense (one point per requested day,
// zero-filled gaps), upload-entity rows only (file-entity rows for the same
// download must not be double counted), and scoped to the requested user_id only
// (another user's and anonymous rows excluded).
func TestBackend_GetUserActivityStatsDailyDownloads(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	userA := common.NewUser(common.ProviderLocal, "series-user-a")
	userB := common.NewUser(common.ProviderLocal, "series-user-b")
	createUser(t, b, userA)
	createUser(t, b, userB)

	today := statsDay(time.Now())
	rows := []*common.DownloadStatsDaily{
		// userA, today: upload-entity row that must be summed.
		{Day: today, EntityType: common.DownloadStatsEntityUpload, EntityID: "a-today-upload", UserID: userA.ID, Downloads: 3, Bytes: 300},
		// userA, today: file-entity row for the same logical download — must
		// NOT be double counted into the series.
		{Day: today, EntityType: common.DownloadStatsEntityFile, EntityID: "a-today-file", UserID: userA.ID, Downloads: 3, Bytes: 300},
		// userA, 2 days ago: no row on the day in between (yesterday), so the
		// series must zero-fill that gap.
		{Day: today.AddDate(0, 0, -2), EntityType: common.DownloadStatsEntityUpload, EntityID: "a-2d-upload", UserID: userA.ID, Downloads: 5, Bytes: 500},
		// userB, today: must be excluded from userA's series.
		{Day: today, EntityType: common.DownloadStatsEntityUpload, EntityID: "b-today-upload", UserID: userB.ID, Downloads: 7, Bytes: 700},
		// anonymous, today: must be excluded from userA's series.
		{Day: today, EntityType: common.DownloadStatsEntityUpload, EntityID: "anon-today-upload", UserID: "", Downloads: 9, Bytes: 900},
	}
	for _, row := range rows {
		require.NoError(t, b.CreateDownloadStatsDaily(row))
	}

	series, err := b.GetUserActivityStatsDaily(userA.ID, 3)
	require.NoError(t, err)
	require.Len(t, series, 3)

	require.Equal(t, today.AddDate(0, 0, -2), series[0].Day)
	require.Equal(t, int64(5), series[0].Downloads)
	require.Equal(t, int64(500), series[0].DownloadedBytes)

	require.Equal(t, today.AddDate(0, 0, -1), series[1].Day)
	require.Equal(t, int64(0), series[1].Downloads, "gap day must be zero-filled, not omitted")
	require.Equal(t, int64(0), series[1].DownloadedBytes)

	require.Equal(t, today, series[2].Day)
	require.Equal(t, int64(3), series[2].Downloads, "file-entity row for the same download must not be double counted")
	require.Equal(t, int64(300), series[2].DownloadedBytes)
}

// TestBackend_GetUserActivityStatsDailyAfterUploadRemoved is the point of
// storing attribution on the rollup row rather than joining through uploads:
// deleting the upload must not remove its downloads from the owner's series.
func TestBackend_GetUserActivityStatsDailyAfterUploadRemoved(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "series-deleted-user")
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Comments: "deleted upload still in user series"}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	require.NoError(t, b.RecordFileDownload(upload, file, 1024, true))

	series, err := b.GetUserActivityStatsDaily(user.ID, 1)
	require.NoError(t, err)
	require.Len(t, series, 1)
	require.Equal(t, int64(1), series[0].Downloads)
	require.Equal(t, int64(1024), series[0].DownloadedBytes)

	require.NoError(t, b.RemoveUpload(upload.ID))

	series, err = b.GetUserActivityStatsDaily(user.ID, 1)
	require.NoError(t, err)
	require.Len(t, series, 1)
	require.Equal(t, int64(1), series[0].Downloads, "deleted upload's download must still be in the user series")
	require.Equal(t, int64(1024), series[0].DownloadedBytes)
}

// TestBackend_GetUserActivityStatsDailyWindowEdge reuses the exact-boundary
// style of the existing window tests: a 7-day series must include exactly the
// last 7 UTC days (today plus the previous 6) and exclude the 8th day back.
func TestBackend_GetUserActivityStatsDailyWindowEdge(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "series-window-edge-user")
	createUser(t, b, user)

	today := statsDay(time.Now())
	rows := []*common.DownloadStatsDaily{
		{Day: today, EntityType: common.DownloadStatsEntityUpload, EntityID: "window-today", UserID: user.ID, Downloads: 1},
		{Day: today.AddDate(0, 0, -6), EntityType: common.DownloadStatsEntityUpload, EntityID: "window-edge", UserID: user.ID, Downloads: 2},
		{Day: today.AddDate(0, 0, -7), EntityType: common.DownloadStatsEntityUpload, EntityID: "window-excluded", UserID: user.ID, Downloads: 4},
	}
	for _, row := range rows {
		require.NoError(t, b.CreateDownloadStatsDaily(row))
	}

	series, err := b.GetUserActivityStatsDaily(user.ID, 7)
	require.NoError(t, err)
	require.Len(t, series, 7)
	require.Equal(t, today.AddDate(0, 0, -6), series[0].Day)
	require.Equal(t, today, series[6].Day)

	var total int64
	for _, point := range series {
		total += point.Downloads
	}
	require.Equal(t, int64(3), total, "the 8th day back (window-excluded) must not be included in a 7-day series")
	require.Equal(t, int64(2), series[0].Downloads)
	require.Equal(t, int64(1), series[6].Downloads)
}

// TestStatsDayUsesUTCCalendarDay pins statsDay's UTC-day
// normalization against fixed, hand-computed expectations. Every other test in
// this file that needs a "today" fixture calls statsDay(time.Now()) for
// both the input AND the expected value, so a local-time regression in
// statsDay itself would shift the fixture and the expectation together
// and those tests would keep passing. This test never calls statsDay to
// build its own expectation, so it can actually catch that class of bug.
func TestStatsDayUsesUTCCalendarDay(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{
			name: "already UTC midnight",
			in:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "end of the UTC day truncates down to the same UTC day",
			in:   time.Date(2026, 3, 15, 23, 59, 59, 999999999, time.UTC),
			want: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			// 23:30 at a fixed UTC-5 offset is 04:30 UTC the *next* calendar
			// day: a naive local-day extraction (ignoring the offset) would
			// wrongly keep this on the 15th.
			name: "local time behind UTC rolls into the next UTC day",
			in:   time.Date(2026, 3, 15, 23, 30, 0, 0, time.FixedZone("UTC-5", -5*60*60)),
			want: time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			// 01:00 at a fixed UTC+10 offset is 15:00 UTC the *previous*
			// calendar day: a naive local-day extraction would wrongly keep
			// this on the 15th instead of rolling it back to the 14th.
			name: "local time ahead of UTC rolls into the previous UTC day",
			in:   time.Date(2026, 3, 15, 1, 0, 0, 0, time.FixedZone("UTC+10", 10*60*60)),
			want: time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			// DST-boundary local date: US Eastern springs forward on
			// 2026-03-08 (02:00 EST -> 03:00 EDT). A fixed post-transition
			// EDT (UTC-4) offset is used explicitly, both to keep the fixture
			// deterministic/CI-safe (no dependency on the tzdata database
			// being installed) and to give an unambiguous wall-clock time,
			// while still exercising a real DST-transition calendar date: a
			// bug that derived the UTC day from the local date component
			// alone, without applying the offset, would wrongly keep this on
			// the 8th instead of rolling it forward to the 9th.
			name: "DST-boundary local date (US Eastern spring-forward day)",
			in:   time.Date(2026, 3, 8, 21, 0, 0, 0, time.FixedZone("EDT", -4*60*60)),
			want: time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statsDay(tt.in)
			require.True(t, got.Equal(tt.want), "statsDay(%v) = %v, want %v", tt.in, got, tt.want)
			require.Equal(t, time.UTC, got.Location(), "statsDay must return a time.Time located in UTC")
		})
	}
}

// TestBackend_GetServerActivityStatsDailyDownloads pins the exact per-day
// server-wide download measures of the activity series and cross-checks them
// against the downloads 1d/7d/30d window sums (populateServerActivityWindows)
// for the same seeded data.
func TestBackend_GetServerActivityStatsDailyDownloads(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())
	rows := []*common.DownloadStatsDaily{
		{Day: today, EntityType: common.DownloadStatsEntityUpload, EntityID: "server-today-upload", Downloads: 1, Bytes: 111},
		{Day: today, EntityType: common.DownloadStatsEntityFile, EntityID: "server-today-file", Downloads: 100, Bytes: 999},
		{Day: today.AddDate(0, 0, -3), EntityType: common.DownloadStatsEntityUpload, EntityID: "server-3d-upload", Downloads: 2, Bytes: 222},
		{Day: today.AddDate(0, 0, -8), EntityType: common.DownloadStatsEntityUpload, EntityID: "server-8d-upload", Downloads: 4, Bytes: 444},
		{Day: today.AddDate(0, 0, -31), EntityType: common.DownloadStatsEntityUpload, EntityID: "server-31d-upload", Downloads: 8, Bytes: 888},
	}
	for _, row := range rows {
		require.NoError(t, b.CreateDownloadStatsDaily(row))
	}

	series, err := b.GetServerActivityStatsDaily(30)
	require.NoError(t, err)
	require.Len(t, series, 30)
	require.Equal(t, today.AddDate(0, 0, -29), series[0].Day)
	require.Equal(t, today, series[29].Day)

	require.Equal(t, int64(1), series[29].Downloads, "file-entity row on the same day must not be double counted")
	require.Equal(t, int64(111), series[29].DownloadedBytes)
	require.Equal(t, int64(2), series[26].Downloads, "today-3d")
	require.Equal(t, int64(4), series[21].Downloads, "today-8d")

	var total int64
	for _, point := range series {
		total += point.Downloads
	}
	require.Equal(t, int64(7), total, "server-31d-upload is outside the 30-day series and must be excluded")

	stats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, int64(1), *stats.Usage.Downloads.Today)
	require.Equal(t, int64(3), *stats.Usage.Downloads.Last7Days)
	require.Equal(t, int64(7), *stats.Usage.Downloads.Last30Days)
}

// TestBackend_EverUsersIsAppendOnly asserts that the server-wide lifetime user
// counter counts creations, not current headcount: deleting a user must never
// decrease it, unlike Users which always reflects the live user count.
func TestBackend_EverUsersIsAppendOnly(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	userA := common.NewUser(common.ProviderLocal, "ever-users-a")
	userB := common.NewUser(common.ProviderLocal, "ever-users-b")
	createUser(t, b, userA)
	createUser(t, b, userB)

	deleted, err := b.DeleteUser(userA.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	stats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 1, stats.Users, "current user count must reflect the live headcount")
	require.Equal(t, 2, stats.LifetimeUsers, "lifetime user count must not decrease on delete")
}

// TestBackend_DeleteUserFoldsLifetimeIntoTombstone is the fold invariant: under
// sum-on-read, deleting a user's token=” row would shrink server lifetime totals,
// so DeleteUser folds that row into the ("__deleted__","") tombstone first. Server
// lifetime uploads/files/size/downloads/bytes and lifetime_users must be UNCHANGED
// across the deletion, while server current totals drop by the removed uploads.
// A repeat delete must be idempotent (no double fold).
func TestBackend_DeleteUserFoldsLifetimeIntoTombstone(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// A surviving user so the server sum has a live contributor alongside the
	// tombstone (proving the two are summed together).
	keep := common.NewUser(common.ProviderLocal, "fold-keep")
	createUser(t, b, keep)
	keepUpload := &common.Upload{User: keep.ID}
	kf := keepUpload.NewFile()
	kf.Size = 111
	kf.Status = common.FileUploaded
	createUpload(t, b, keepUpload)

	// The user to delete, with upload + download + bytes history.
	victim := common.NewUser(common.ProviderLocal, "fold-victim")
	createUser(t, b, victim)
	upload := &common.Upload{User: victim.ID}
	file := upload.NewFile()
	file.Size = 5000
	file.Status = common.FileUploaded
	createUpload(t, b, upload)
	require.NoError(t, b.RecordFileDownload(upload, file, 2048, true))

	before, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 2, before.Uploads, "current uploads before delete: keep + victim")
	require.Equal(t, 2, before.Usage.Lifetime.Uploads)
	require.Equal(t, int64(5111), before.Usage.Lifetime.TotalSize)
	require.Equal(t, int64(1), before.Usage.Downloads.Total)
	require.Equal(t, int64(2048), before.Usage.Uploads.Bytes+before.Usage.Downloads.Bytes, "bytes recorded")
	require.Equal(t, 2, before.LifetimeUsers)

	deleted, err := b.DeleteUser(victim.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	after, err := b.GetServerStatistics()
	require.NoError(t, err)

	// Current totals drop by the removed victim upload; only the surviving user's
	// upload remains current.
	require.Equal(t, 1, after.Uploads, "server current uploads drop by the removed user's uploads")
	require.Equal(t, 1, after.Files)
	require.Equal(t, int64(111), after.TotalSize)

	// Lifetime totals are UNCHANGED — the tombstone holds the victim's history.
	require.Equal(t, before.Usage.Lifetime.Uploads, after.Usage.Lifetime.Uploads, "server lifetime uploads unchanged")
	require.Equal(t, before.Usage.Lifetime.Files, after.Usage.Lifetime.Files, "server lifetime files unchanged")
	require.Equal(t, before.Usage.Lifetime.TotalSize, after.Usage.Lifetime.TotalSize, "server lifetime size unchanged")
	require.Equal(t, before.Usage.Downloads.Total, after.Usage.Downloads.Total, "server lifetime downloads unchanged")
	require.Equal(t, before.Usage.Downloads.Bytes, after.Usage.Downloads.Bytes, "server downloaded bytes unchanged")
	require.Equal(t, before.LifetimeUsers, after.LifetimeUsers, "server lifetime_users unchanged")
	require.Equal(t, 1, after.Users, "live headcount drops to the surviving user")

	// Repeat delete is idempotent: no second fold, totals stay put.
	deleted, err = b.DeleteUser(victim.ID)
	require.NoError(t, err)
	require.False(t, deleted, "second delete finds no user")

	again, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, after.Usage.Lifetime.Uploads, again.Usage.Lifetime.Uploads, "repeat delete must not double-fold")
	require.Equal(t, after.Usage.Lifetime.TotalSize, again.Usage.Lifetime.TotalSize, "repeat delete must not double-fold size")
	require.Equal(t, after.Usage.Downloads.Total, again.Usage.Downloads.Total, "repeat delete must not double-fold downloads")
	require.Equal(t, after.LifetimeUsers, again.LifetimeUsers, "repeat delete must not double-fold users")
}

// TestBackend_DeleteUserFoldRecreatesAbsentTombstone covers the unmigrated edge:
// if the tombstone row is missing when a user is deleted, the fold upsert-creates
// it so server lifetime totals are still preserved.
func TestBackend_DeleteUserFoldRecreatesAbsentTombstone(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// Remove the tombstone seeded at init to simulate an unmigrated instance.
	err := b.db.Where("user_id = ? AND token = ?", common.DeletedUserUsageStatsID, "").Delete(&common.UsageStats{}).Error
	require.NoError(t, err)

	user := common.NewUser(common.ProviderLocal, "absent-tombstone-user")
	createUser(t, b, user)
	upload := &common.Upload{User: user.ID}
	file := upload.NewFile()
	file.Size = 321
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	before, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 1, before.Usage.Lifetime.Uploads)
	require.Equal(t, int64(321), before.Usage.Lifetime.TotalSize)

	deleted, err := b.DeleteUser(user.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	// The tombstone was recreated by the fold and preserves the lifetime history.
	tombstone, err := b.GetDeletedUsageTombstone()
	require.NoError(t, err)
	require.NotNil(t, tombstone, "fold must recreate the absent tombstone")
	require.Equal(t, 1, tombstone.LifetimeUploads)
	require.Equal(t, int64(321), tombstone.LifetimeSize)
	require.Equal(t, 1, tombstone.LifetimeUsers)

	after, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, before.Usage.Lifetime.Uploads, after.Usage.Lifetime.Uploads, "lifetime uploads preserved via recreated tombstone")
	require.Equal(t, before.Usage.Lifetime.TotalSize, after.Usage.Lifetime.TotalSize)
	require.Equal(t, before.LifetimeUsers, after.LifetimeUsers)
}

// TestBackend_DeleteUserFoldPreservesEarliestStartedAt pins the "Stats since"
// anchor across deletion of the oldest scope. The server StartedAt is MIN(started_at)
// over the token=” rows; folding the deleted user's row into the tombstone must
// carry that row's started_at back to the tombstone, so dropping the row does not
// let the anchor jump FORWARD when the deleted user was the oldest.
func TestBackend_DeleteUserFoldPreservesEarliestStartedAt(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// A user created far in the past — older than the init tombstone/anonymous rows.
	historical := time.Now().Add(-1000 * time.Hour).Truncate(time.Second)
	old := common.NewUser(common.ProviderLocal, "oldest-user")
	old.CreatedAt = historical
	createUser(t, b, old)

	// A newer surviving user so the sum is not trivially empty after deletion.
	keep := common.NewUser(common.ProviderLocal, "newer-user")
	createUser(t, b, keep)

	// The oldest user's started_at is the server anchor before deletion.
	before, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.NotNil(t, before.Usage.StartedAt)
	require.WithinDuration(t, historical, *before.Usage.StartedAt, time.Second, "anchor before delete = oldest user's started_at")

	deleted, err := b.DeleteUser(old.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	// The fold carried the historical started_at into the tombstone, so the anchor
	// does not move forward.
	after, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.NotNil(t, after.Usage.StartedAt)
	require.WithinDuration(t, historical, *after.Usage.StartedAt, time.Second, "anchor must stay at the deleted oldest user's started_at, not jump forward")

	tombstone, err := b.GetDeletedUsageTombstone()
	require.NoError(t, err)
	require.NotNil(t, tombstone)
	require.WithinDuration(t, historical, tombstone.StartedAt, time.Second, "tombstone started_at pulled back to the folded row's")
}

// TestBackend_DeleteUserFoldVsConcurrentMutationsIsExact races a DeleteUser fold
// against uploads and downloads by OTHER users. Whatever the interleaving, server
// lifetime totals must equal the exact sum of every user's history (the deleted
// user's, folded into the tombstone, plus the concurrent users' live rows), with
// -race green. It runs with -race via the Makefile.
func TestBackend_DeleteUserFoldVsConcurrentMutationsIsExact(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	const others = 8
	const fileSize = int64(1000)

	// Concurrent users each create one upload and download it once.
	otherUsers := make([]*common.User, others)
	otherUploads := make([]*common.Upload, others)
	otherFiles := make([]*common.File, others)
	for i := range others {
		u := common.NewUser(common.ProviderLocal, fmt.Sprintf("fold-race-other-%d", i))
		createUser(t, b, u)
		up := &common.Upload{User: u.ID}
		f := up.NewFile()
		f.Size = fileSize
		f.Status = common.FileUploaded
		createUpload(t, b, up)
		otherUsers[i] = u
		otherUploads[i] = up
		otherFiles[i] = f
	}

	// The victim: created with lifetime history, deleted concurrently.
	victim := common.NewUser(common.ProviderLocal, "fold-race-victim")
	createUser(t, b, victim)
	vUpload := &common.Upload{User: victim.ID}
	vFile := vUpload.NewFile()
	vFile.Size = fileSize
	vFile.Status = common.FileUploaded
	createUpload(t, b, vUpload)

	var wg sync.WaitGroup
	errs := make(chan error, others+1)
	for i := range others {
		wg.Go(func() {
			errs <- b.RecordFileDownload(otherUploads[i], otherFiles[i], 512, true)
		})
	}
	wg.Go(func() {
		_, err := b.DeleteUser(victim.ID)
		errs <- err
	})
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	stats, err := b.GetServerStatistics()
	require.NoError(t, err)
	// others + victim uploads all lifetime-counted; victim folded into tombstone.
	require.Equal(t, others+1, stats.Usage.Lifetime.Uploads, "server lifetime uploads exact across the race")
	require.Equal(t, others+1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(others+1)*fileSize, stats.Usage.Lifetime.TotalSize)
	require.Equal(t, int64(others), stats.Usage.Downloads.Total, "every concurrent download counted once")
	require.Equal(t, others+1, stats.LifetimeUsers, "append-only across the deletion")
	require.Equal(t, others, stats.Uploads, "victim's upload removed; others remain current")
}

// TestBackend_ServerStatisticsSumsUserRowsExcludingTokens proves the sum-on-read
// contract: server totals are Σ over the token=” rows (a user row, anonymous,
// and the tombstone), and the token rows are excluded so a token'd upload — which
// fans out to both the owner's user row and its token row — is counted once, not
// twice. It also proves the tombstone's folded lifetime counters are included and
// that startedAt is the MIN across the summed rows (the migration/init tombstone
// anchor is never later than a lazily-created user row).
func TestBackend_ServerStatisticsSumsUserRowsExcludingTokens(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "sum-user")
	token := user.NewToken()
	createUser(t, b, user)

	// A token'd upload increments the user row (token='') AND the token row.
	tokenUpload := &common.Upload{User: user.ID, Token: token.Token}
	tf := tokenUpload.NewFile()
	tf.Size = 100
	tf.Status = common.FileUploaded
	createUpload(t, b, tokenUpload)

	// A tokenless (web-session) upload for the same user.
	webUpload := &common.Upload{User: user.ID}
	wf := webUpload.NewFile()
	wf.Size = 200
	wf.Status = common.FileUploaded
	createUpload(t, b, webUpload)

	// An anonymous upload.
	anonUpload := &common.Upload{}
	af := anonUpload.NewFile()
	af.Size = 400
	af.Status = common.FileUploaded
	createUpload(t, b, anonUpload)

	// Fold a deleted user with lifetime history into the tombstone.
	deleted := common.NewUser(common.ProviderLocal, "sum-deleted")
	createUser(t, b, deleted)
	delUpload := &common.Upload{User: deleted.ID}
	df := delUpload.NewFile()
	df.Size = 800
	df.Status = common.FileUploaded
	createUpload(t, b, delUpload)
	gone, err := b.DeleteUser(deleted.ID)
	require.NoError(t, err)
	require.True(t, gone)

	stats, err := b.GetServerStatistics()
	require.NoError(t, err)

	// Current = the two retained live uploads (token'd + web) + anonymous. The
	// deleted user's upload was removed with the account, so it is not current.
	require.Equal(t, 3, stats.Uploads, "server current uploads = user(2) + anon(1); token row excluded")
	require.Equal(t, 3, stats.Files)
	require.Equal(t, int64(700), stats.TotalSize, "100 + 200 + 400; the deleted user's 800 was removed")

	// Lifetime includes the deleted user's folded history via the tombstone.
	require.Equal(t, 4, stats.Usage.Lifetime.Uploads, "user(2) + anon(1) + tombstone(1)")
	require.Equal(t, 4, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(1500), stats.Usage.Lifetime.TotalSize, "100 + 200 + 400 + 800 (tombstone)")

	// Two users were created; the deleted one's lifetime_users folded into the
	// tombstone, so the append-only server counter still reads 2.
	require.Equal(t, 1, stats.Users, "live headcount")
	require.Equal(t, 2, stats.LifetimeUsers, "append-only: live user(1) + tombstone(1)")

	// Anonymous fields read the anonymous row directly.
	require.Equal(t, 1, stats.AnonymousUploads)
	require.Equal(t, int64(400), stats.AnonymousSize)

	require.NotNil(t, stats.Usage.StartedAt)
	require.False(t, stats.Usage.StartedAt.IsZero())
}

func TestBackend_RecordFileDownloadConcurrent(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	const count = 20
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for range count {
		wg.Go(func() {
			errs <- b.RecordFileDownload(upload, file, 1024, true)
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	gotFile, err := b.GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(count), gotFile.DownloadCount)

	gotUpload, err := b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(count), gotUpload.DownloadCount)
	require.Equal(t, int64(count)*1024, gotUpload.DownloadedBytes, "concurrent downloads must not lose or double-count egress bytes")
}

func trendingItemByID(t *testing.T, items []*common.TrendingItem, id string) *common.TrendingItem {
	t.Helper()
	for _, item := range items {
		if item.ID == id {
			return item
		}
	}
	require.Failf(t, "missing trending item", "id %s not found", id)
	return nil
}

func requireContainsTrendingID(t *testing.T, items []*common.TrendingItem, id string) {
	t.Helper()
	require.NotNil(t, trendingItemByID(t, items, id))
}

func requireNotContainsTrendingID(t *testing.T, items []*common.TrendingItem, id string) {
	t.Helper()
	for _, item := range items {
		require.NotEqual(t, id, item.ID, "unexpected trending item")
	}
}

func TestBackend_StatsMigrationBackfill(t *testing.T) {
	oldConfig := *metadataBackendConfig
	oldConfig.EraseFirst = true
	oldConfig.disableSchemaInit = true
	oldConfig.migrationFilter = func(migrations []*gormigrate.Migration) []*gormigrate.Migration {
		var filtered []*gormigrate.Migration
		for _, migration := range migrations {
			if migration.ID == "0011-stats" {
				continue
			}
			filtered = append(filtered, migration)
		}
		return filtered
	}
	if oldConfig.Driver == "sqlite3" {
		path := fmt.Sprintf("/tmp/plik.stats-migration-%d.db", time.Now().UnixNano())
		defer func() { _ = os.Remove(path) }()
		oldConfig.ConnectionString = path
	}

	oldBackend, err := NewBackend(&oldConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)

	user := common.NewUser(common.ProviderLocal, "migration-user")
	err = oldBackend.db.Table("users").Create(map[string]any{
		"id":         user.ID,
		"provider":   user.Provider,
		"login":      user.Login,
		"created_at": time.Now(),
	}).Error
	require.NoError(t, err)

	token := user.NewToken()
	err = oldBackend.db.Table("tokens").Create(map[string]any{
		"token":      token.Token,
		"comment":    token.Comment,
		"user_id":    token.UserID,
		"created_at": time.Now(),
	}).Error
	require.NoError(t, err)

	uploadCreatedAt := time.Now()
	uploadID := common.GenerateRandomID(16)
	fileID := common.GenerateRandomID(16)
	err = oldBackend.db.Table("uploads").Create(map[string]any{
		"id":           uploadID,
		"ttl":          3600,
		"upload_token": common.GenerateRandomID(32),
		"user":         user.ID,
		"token":        token.Token,
		"created_at":   uploadCreatedAt,
	}).Error
	require.NoError(t, err)
	err = oldBackend.db.Table("files").Create(map[string]any{
		"id":         fileID,
		"upload_id":  uploadID,
		"name":       "migration-file",
		"status":     common.FileUploaded,
		"size":       42,
		"created_at": time.Now(),
	}).Error
	require.NoError(t, err)

	err = oldBackend.Shutdown()
	require.NoError(t, err)

	newConfig := *metadataBackendConfig
	newConfig.ConnectionString = oldConfig.ConnectionString
	newConfig.EraseFirst = false
	newConfig.disableSchemaInit = true
	newBackend, err := NewBackend(&newConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)
	defer shutdownTestMetadataBackend(newBackend)

	stats, err := newBackend.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Uploads)
	require.Equal(t, 1, stats.Files)
	require.Equal(t, int64(42), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(42), stats.Usage.Lifetime.TotalSize)
	require.NotNil(t, stats.Usage.StartedAt)
	require.False(t, stats.Usage.StartedAt.IsZero())

	tokenStats, err := newBackend.GetUserStatistics(user.ID, &token.Token)
	require.NoError(t, err)
	require.Equal(t, 1, tokenStats.Uploads)
	require.Equal(t, 1, tokenStats.Files)
	require.Equal(t, int64(42), tokenStats.TotalSize)

	tokens, _, err := newBackend.GetTokens(user.ID, "", common.NewPagingQuery().WithLimit(1))
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.NotNil(t, tokens[0].Stats.Usage.LastUploadAt)
	require.WithinDuration(t, uploadCreatedAt, *tokens[0].Stats.Usage.LastUploadAt, time.Second)

	serverStats, err := newBackend.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 1, serverStats.Usage.Current.TTL.OneHourToOneDayUploads)
	require.Equal(t, 1, serverStats.Usage.Lifetime.TTL.OneHourToOneDayUploads)
	require.Equal(t, 1, serverStats.Usage.Current.FileSizes.LessThan1MBFiles)
	require.Equal(t, 1, serverStats.Usage.Lifetime.FileSizes.LessThan1MBFiles)
	require.Equal(t, 1, serverStats.Users, "invalid current user count")
	require.Equal(t, 1, serverStats.LifetimeUsers, "backfilled lifetime user count must match seeded users")
	require.True(t, newBackend.db.Migrator().HasIndex(&common.File{}, "idx_file_upload_id"), "missing file upload index")
}

// TestBackend_StatsMigrationBackfillSemantics builds a real pre-0011 database
// by skipping only the stats migration, inserts legacy rows directly, then
// reopens the backend normally so 0011-stats performs the backfill. The fixture
// intentionally mixes current and soft-deleted uploads/files because current
// counters must ignore soft-deleted rows while lifetime counters must include
// retained historical metadata.
func TestBackend_StatsMigrationBackfillSemantics(t *testing.T) {
	oldConfig := *metadataBackendConfig
	oldConfig.EraseFirst = true
	oldConfig.disableSchemaInit = true
	oldConfig.migrationFilter = func(migrations []*gormigrate.Migration) []*gormigrate.Migration {
		var filtered []*gormigrate.Migration
		for _, migration := range migrations {
			if migration.ID == "0011-stats" {
				continue
			}
			filtered = append(filtered, migration)
		}
		return filtered
	}
	if oldConfig.Driver == "sqlite3" {
		path := fmt.Sprintf("/tmp/plik.stats-migration-semantics-%d.db", time.Now().UnixNano())
		defer func() { _ = os.Remove(path) }()
		oldConfig.ConnectionString = path
	}

	oldBackend, err := NewBackend(&oldConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)

	user := common.NewUser(common.ProviderLocal, "migration-rich-user")
	err = oldBackend.db.Table("users").Create(map[string]any{
		"id":         user.ID,
		"provider":   user.Provider,
		"login":      user.Login,
		"created_at": time.Now(),
	}).Error
	require.NoError(t, err)

	token := user.NewToken()
	err = oldBackend.db.Table("tokens").Create(map[string]any{
		"token":      token.Token,
		"comment":    token.Comment,
		"user_id":    token.UserID,
		"created_at": time.Now(),
	}).Error
	require.NoError(t, err)

	base := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	deletedAt := base.Add(3 * time.Hour)

	userCurrentUploadID := common.GenerateRandomID(16)
	userDeletedUploadID := common.GenerateRandomID(16)
	anonymousCurrentUploadID := common.GenerateRandomID(16)
	anonymousDeletedUploadID := common.GenerateRandomID(16)
	tokenLastUploadAt := base.Add(2 * time.Hour)

	legacyUploads := []map[string]any{
		{
			"id":                    userCurrentUploadID,
			"ttl":                   0,
			"upload_token":          common.GenerateRandomID(32),
			"user":                  user.ID,
			"token":                 token.Token,
			"comments":              "current token upload",
			"protected_by_password": true,
			"removable":             true,
			"created_at":            base,
		},
		{
			"id":           userDeletedUploadID,
			"ttl":          3600,
			"upload_token": common.GenerateRandomID(32),
			"user":         user.ID,
			"token":        token.Token,
			"comments":     "deleted token upload",
			"one_shot":     true,
			"stream":       true,
			"extend_ttl":   true,
			"e2ee":         "age",
			"created_at":   tokenLastUploadAt,
			"deleted_at":   deletedAt,
		},
		{
			"id":           anonymousCurrentUploadID,
			"ttl":          5184000,
			"upload_token": common.GenerateRandomID(32),
			"user":         "",
			"token":        "",
			"created_at":   base.Add(time.Hour),
		},
		{
			"id":           anonymousDeletedUploadID,
			"ttl":          0,
			"upload_token": common.GenerateRandomID(32),
			"user":         "",
			"token":        "",
			"created_at":   base.Add(30 * time.Minute),
			"deleted_at":   deletedAt,
		},
	}
	for _, upload := range legacyUploads {
		err = oldBackend.db.Table("uploads").Create(upload).Error
		require.NoError(t, err)
	}

	legacyFiles := []map[string]any{
		{
			"id":         common.GenerateRandomID(16),
			"upload_id":  userCurrentUploadID,
			"name":       "user-current.txt",
			"status":     common.FileUploaded,
			"size":       10,
			"created_at": base,
		},
		{
			"id":         common.GenerateRandomID(16),
			"upload_id":  userDeletedUploadID,
			"name":       "user-removed.txt",
			"status":     common.FileRemoved,
			"size":       20,
			"created_at": base,
		},
		{
			"id":         common.GenerateRandomID(16),
			"upload_id":  userDeletedUploadID,
			"name":       "user-deleted.txt",
			"status":     common.FileDeleted,
			"size":       30,
			"created_at": base,
		},
		{
			"id":         common.GenerateRandomID(16),
			"upload_id":  anonymousCurrentUploadID,
			"name":       "anonymous-current.txt",
			"status":     common.FileUploaded,
			"size":       40,
			"created_at": base,
		},
		{
			"id":         common.GenerateRandomID(16),
			"upload_id":  anonymousDeletedUploadID,
			"name":       "anonymous-deleted.txt",
			"status":     common.FileDeleted,
			"size":       50,
			"created_at": base,
		},
	}
	for _, file := range legacyFiles {
		err = oldBackend.db.Table("files").Create(file).Error
		require.NoError(t, err)
	}

	err = oldBackend.Shutdown()
	require.NoError(t, err)

	newConfig := *metadataBackendConfig
	newConfig.ConnectionString = oldConfig.ConnectionString
	newConfig.EraseFirst = false
	newConfig.disableSchemaInit = true
	newBackend, err := NewBackend(&newConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)
	defer shutdownTestMetadataBackend(newBackend)

	userStats, err := newBackend.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, userStats.Uploads)
	require.Equal(t, 1, userStats.Files)
	require.Equal(t, int64(10), userStats.TotalSize)
	require.Equal(t, 2, userStats.Usage.Lifetime.Uploads)
	require.Equal(t, 3, userStats.Usage.Lifetime.Files)
	require.Equal(t, int64(60), userStats.Usage.Lifetime.TotalSize)
	require.NotNil(t, userStats.Usage.StartedAt)
	require.False(t, userStats.Usage.StartedAt.IsZero())

	tokenStats, err := newBackend.GetUserStatistics(user.ID, &token.Token)
	require.NoError(t, err)
	require.Equal(t, 1, tokenStats.Uploads)
	require.Equal(t, 1, tokenStats.Files)
	require.Equal(t, int64(10), tokenStats.TotalSize)
	require.Equal(t, 2, tokenStats.Usage.Lifetime.Uploads)
	require.Equal(t, 3, tokenStats.Usage.Lifetime.Files)
	require.Equal(t, int64(60), tokenStats.Usage.Lifetime.TotalSize)

	tokens, _, err := newBackend.GetTokens(user.ID, "", common.NewPagingQuery().WithLimit(1))
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.NotNil(t, tokens[0].Stats.Usage.LastUploadAt)
	require.WithinDuration(t, tokenLastUploadAt, *tokens[0].Stats.Usage.LastUploadAt, time.Second)

	anonymousStats, err := newBackend.GetUserStatistics(common.AnonymousUserUsageStatsID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, anonymousStats.Uploads)
	require.Equal(t, 1, anonymousStats.Files)
	require.Equal(t, int64(40), anonymousStats.TotalSize)
	require.Equal(t, 2, anonymousStats.Usage.Lifetime.Uploads)
	require.Equal(t, 2, anonymousStats.Usage.Lifetime.Files)
	require.Equal(t, int64(90), anonymousStats.Usage.Lifetime.TotalSize)
	require.NotNil(t, anonymousStats.Usage.StartedAt)
	require.False(t, anonymousStats.Usage.StartedAt.IsZero())

	serverStats, err := newBackend.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 1, serverStats.Users)
	require.Equal(t, 2, serverStats.Uploads)
	require.Equal(t, 2, serverStats.Files)
	require.Equal(t, int64(50), serverStats.TotalSize)
	require.Equal(t, 1, serverStats.AnonymousUploads)
	require.Equal(t, int64(40), serverStats.AnonymousSize)
	require.Equal(t, 1, serverStats.LifetimeUsers)
	require.Equal(t, 4, serverStats.Usage.Lifetime.Uploads)
	require.Equal(t, 5, serverStats.Usage.Lifetime.Files)
	require.Equal(t, int64(150), serverStats.Usage.Lifetime.TotalSize)
	require.Equal(t, 2, serverStats.AnonymousUsage.Lifetime.Uploads)
	require.Equal(t, int64(90), serverStats.AnonymousUsage.Lifetime.TotalSize)
	require.Equal(t, int64(0), serverStats.Usage.Downloads.Total)
	require.Equal(t, int64(0), *serverStats.Usage.Downloads.Today)
	require.Equal(t, int64(0), *serverStats.Usage.Downloads.Last7Days)
	require.Equal(t, int64(0), *serverStats.Usage.Downloads.Last30Days)

	require.Equal(t, 1, serverStats.Usage.Current.Features.PasswordUploads)
	require.Equal(t, 1, serverStats.Usage.Current.Features.RemovableUploads)
	require.Equal(t, 0, serverStats.Usage.Current.Features.OneShotUploads)
	require.Equal(t, 0, serverStats.Usage.Current.Features.StreamUploads)
	require.Equal(t, 0, serverStats.Usage.Current.Features.ExtendTTLUploads)
	require.Equal(t, 0, serverStats.Usage.Current.Features.E2EEUploads)
	require.Equal(t, 1, serverStats.Usage.Current.Features.CommentUploads)
	require.Equal(t, 1, serverStats.Usage.Lifetime.Features.PasswordUploads)
	require.Equal(t, 1, serverStats.Usage.Lifetime.Features.RemovableUploads)
	require.Equal(t, 1, serverStats.Usage.Lifetime.Features.OneShotUploads)
	require.Equal(t, 1, serverStats.Usage.Lifetime.Features.StreamUploads)
	require.Equal(t, 1, serverStats.Usage.Lifetime.Features.ExtendTTLUploads)
	require.Equal(t, 1, serverStats.Usage.Lifetime.Features.E2EEUploads)
	require.Equal(t, 2, serverStats.Usage.Lifetime.Features.CommentUploads)

	require.Equal(t, 1, serverStats.Usage.Current.TTL.NoneUploads)
	require.Equal(t, 0, serverStats.Usage.Current.TTL.OneHourToOneDayUploads)
	require.Equal(t, 1, serverStats.Usage.Current.TTL.GreaterThan30DaysUploads)
	require.Equal(t, 2, serverStats.Usage.Lifetime.TTL.NoneUploads)
	require.Equal(t, 1, serverStats.Usage.Lifetime.TTL.OneHourToOneDayUploads)
	require.Equal(t, 1, serverStats.Usage.Lifetime.TTL.GreaterThan30DaysUploads)

	require.Equal(t, 2, serverStats.Usage.Current.FileSizes.LessThan1MBFiles)
	require.Equal(t, 5, serverStats.Usage.Lifetime.FileSizes.LessThan1MBFiles)
	require.Equal(t, 0, serverStats.Usage.Current.FileSizes.OneMBTo10MBFiles)
	require.Equal(t, 0, serverStats.Usage.Lifetime.FileSizes.OneMBTo10MBFiles)

	require.True(t, newBackend.db.Migrator().HasIndex(&common.File{}, "idx_file_upload_id"), "missing file upload index")
}

// TestBackend_StatsMigrationBackfillRerunAfterPartialFailure simulates a 0011
// backfill that committed some usage_stats rows and then died before recording
// the migration id. gormigrate runs with UseTransaction=false, so partial rows
// autocommit and the migration re-runs on the next start. The re-run must wipe
// the stale partial state and rebuild correct counters instead of aborting on a
// duplicate (user_id, token) primary key.
func TestBackend_StatsMigrationBackfillRerunAfterPartialFailure(t *testing.T) {
	oldConfig := *metadataBackendConfig
	oldConfig.EraseFirst = true
	oldConfig.disableSchemaInit = true
	oldConfig.migrationFilter = func(migrations []*gormigrate.Migration) []*gormigrate.Migration {
		var filtered []*gormigrate.Migration
		for _, migration := range migrations {
			if migration.ID == "0011-stats" {
				continue
			}
			filtered = append(filtered, migration)
		}
		return filtered
	}
	if oldConfig.Driver == "sqlite3" {
		path := fmt.Sprintf("/tmp/plik.stats-migration-rerun-%d.db", time.Now().UnixNano())
		defer func() { _ = os.Remove(path) }()
		oldConfig.ConnectionString = path
	}

	oldBackend, err := NewBackend(&oldConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)

	user := common.NewUser(common.ProviderLocal, "migration-rerun-user")
	err = oldBackend.db.Table("users").Create(map[string]any{
		"id":         user.ID,
		"provider":   user.Provider,
		"login":      user.Login,
		"created_at": time.Now(),
	}).Error
	require.NoError(t, err)

	token := user.NewToken()
	err = oldBackend.db.Table("tokens").Create(map[string]any{
		"token":      token.Token,
		"comment":    token.Comment,
		"user_id":    token.UserID,
		"created_at": time.Now(),
	}).Error
	require.NoError(t, err)

	uploadID := common.GenerateRandomID(16)
	fileID := common.GenerateRandomID(16)
	err = oldBackend.db.Table("uploads").Create(map[string]any{
		"id":           uploadID,
		"ttl":          3600,
		"upload_token": common.GenerateRandomID(32),
		"user":         user.ID,
		"token":        token.Token,
		"created_at":   time.Now(),
	}).Error
	require.NoError(t, err)
	err = oldBackend.db.Table("files").Create(map[string]any{
		"id":         fileID,
		"upload_id":  uploadID,
		"name":       "rerun-file",
		"status":     common.FileUploaded,
		"size":       42,
		"created_at": time.Now(),
	}).Error
	require.NoError(t, err)

	// Reproduce the partial-failure state: 0011's AutoMigrate created usage_stats
	// and the backfill committed one row before dying. The stale row carries wrong
	// counters and its (user_id, token) key collides with the row the re-run
	// rebuilds for this user.
	require.NoError(t, oldBackend.db.AutoMigrate(&common.UsageStats{}))
	err = oldBackend.db.Table("usage_stats").Create(map[string]any{
		"user_id":          user.ID,
		"token":            "",
		"current_uploads":  999,
		"lifetime_uploads": 999,
		"started_at":       time.Now(),
	}).Error
	require.NoError(t, err)

	err = oldBackend.Shutdown()
	require.NoError(t, err)

	newConfig := *metadataBackendConfig
	newConfig.ConnectionString = oldConfig.ConnectionString
	newConfig.EraseFirst = false
	newConfig.disableSchemaInit = true
	newBackend, err := NewBackend(&newConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)
	defer shutdownTestMetadataBackend(newBackend)

	stats, err := newBackend.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Uploads)
	require.Equal(t, 1, stats.Files)
	require.Equal(t, int64(42), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(42), stats.Usage.Lifetime.TotalSize)

	tokenStats, err := newBackend.GetUserStatistics(user.ID, &token.Token)
	require.NoError(t, err)
	require.Equal(t, 1, tokenStats.Uploads)
	require.Equal(t, 1, tokenStats.Files)
	require.Equal(t, int64(42), tokenStats.TotalSize)
}

// TestBackend_StatsMigrationBackfillParameterLimit seeds one user with more
// uploads than SQLite's 32,766 bind-parameter cap. The old backfill plucked
// every upload id and expanded it into a WHERE upload_id IN (?) list, which
// overflowed that limit and aborted the migration. The subquery-based backfill
// keeps the ids in the database, so 0011 succeeds and reports the exact count.
func TestBackend_StatsMigrationBackfillParameterLimit(t *testing.T) {
	if metadataBackendConfig.Driver != "sqlite3" {
		t.Skip("parameter-limit backfill test targets SQLite's variable cap")
	}

	oldConfig := *metadataBackendConfig
	oldConfig.EraseFirst = true
	oldConfig.disableSchemaInit = true
	oldConfig.migrationFilter = func(migrations []*gormigrate.Migration) []*gormigrate.Migration {
		var filtered []*gormigrate.Migration
		for _, migration := range migrations {
			if migration.ID == "0011-stats" {
				continue
			}
			filtered = append(filtered, migration)
		}
		return filtered
	}
	path := fmt.Sprintf("/tmp/plik.stats-migration-paramlimit-%d.db", time.Now().UnixNano())
	defer func() { _ = os.Remove(path) }()
	oldConfig.ConnectionString = path

	oldBackend, err := NewBackend(&oldConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)

	user := common.NewUser(common.ProviderLocal, "migration-scale-user")
	err = oldBackend.db.Table("users").Create(map[string]any{
		"id":         user.ID,
		"provider":   user.Provider,
		"login":      user.Login,
		"created_at": time.Now(),
	}).Error
	require.NoError(t, err)

	// Seed more uploads than the SQLite parameter cap. Files are not needed; the
	// upload count alone overflows an IN-list backfill. Batch raw inserts keep
	// the fixture fast while staying well under the cap per statement.
	const uploadCount = 33500
	createdAt := time.Now()
	batch := make([]map[string]any, 0, 500)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := oldBackend.db.Table("uploads").Create(batch).Error
		batch = batch[:0]
		return err
	}
	for i := range uploadCount {
		batch = append(batch, map[string]any{
			"id":           fmt.Sprintf("scale-upload-%06d", i),
			"ttl":          3600,
			"upload_token": fmt.Sprintf("scale-token-%06d", i),
			"user":         user.ID,
			"created_at":   createdAt,
		})
		if len(batch) == cap(batch) {
			require.NoError(t, flush())
		}
	}
	require.NoError(t, flush())

	var seeded int64
	require.NoError(t, oldBackend.db.Table("uploads").Count(&seeded).Error)
	require.Equal(t, int64(uploadCount), seeded)

	err = oldBackend.Shutdown()
	require.NoError(t, err)

	newConfig := *metadataBackendConfig
	newConfig.ConnectionString = oldConfig.ConnectionString
	newConfig.EraseFirst = false
	newConfig.disableSchemaInit = true
	newBackend, err := NewBackend(&newConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)
	defer shutdownTestMetadataBackend(newBackend)

	stats, err := newBackend.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, uploadCount, stats.Uploads)
	require.Equal(t, uploadCount, stats.Usage.Lifetime.Uploads)
}

// sumDailyDownloads returns the total downloads recorded across every daily
// rollup bucket for one entity.
func sumDailyDownloads(t *testing.T, b *Backend, entityType string, entityID string) int64 {
	t.Helper()

	var total int64
	err := b.db.Model(&common.DownloadStatsDaily{}).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Select("coalesce(sum(downloads), 0)").
		Scan(&total).Error
	require.NoError(t, err)
	return total
}

// sumDailyDownloadBytes mirrors sumDailyDownloads for the daily rollup's bytes
// column, so concurrency exactness tests can pin egress byte totals alongside
// download event counts.
func sumDailyDownloadBytes(t *testing.T, b *Backend, entityType string, entityID string) int64 {
	t.Helper()

	var total int64
	err := b.db.Model(&common.DownloadStatsDaily{}).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Select("coalesce(sum(bytes), 0)").
		Scan(&total).Error
	require.NoError(t, err)
	return total
}

// TestBackend_ConcurrentMixedDownloadsAreExact hammers a single upload with
// interleaved direct and archive downloads from many goroutines. It is the
// regression test for the AB-BA lock-order inversion between RecordFileDownload
// (was file-then-upload) and RecordArchiveDownload (upload-then-file): under the
// canonical order both take the upload row first, so concurrent downloads
// serialize on it instead of deadlocking on Postgres/MySQL. Download recording
// is single-shot best-effort (no retry), so the exact totals below also prove
// the lock order alone prevents conflicts. It runs with -race via the Makefile
// and must be green on SQLite, Postgres, and MySQL.
func TestBackend_ConcurrentMixedDownloadsAreExact(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "concurrent-download-user")
	token := user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Token: token.Token, Comments: "concurrent downloads"}
	const numFiles = 3
	files := make([]*common.File, numFiles)
	for i := range files {
		file := upload.NewFile()
		file.Name = fmt.Sprintf("file-%d.txt", i)
		file.Size = int64(i+1) * 100
		file.Status = common.FileUploaded
		files[i] = file
	}
	createUpload(t, b, upload)

	const directDownloads = 10
	const archiveDownloads = 10

	// Deterministic assignment of direct downloads to files so per-file totals
	// are exact rather than probabilistic.
	expectedDirectPerFile := make([]int64, numFiles)
	for i := range directDownloads {
		expectedDirectPerFile[i%numFiles]++
	}

	var wg sync.WaitGroup
	errs := make(chan error, directDownloads+archiveDownloads)

	for i := range directDownloads {
		wg.Go(func() {
			errs <- b.RecordFileDownload(upload, files[i%numFiles], 1024, true)
		})
	}
	for range archiveDownloads {
		wg.Go(func() {
			errs <- b.RecordArchiveDownload(upload, files, 2048, true)
		})
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err, "concurrent download recording must not fail")
	}

	// Every archive download counts one download for each file; direct downloads
	// add their deterministic per-file share on top. Archive downloads never add
	// bytes to the per-file rollup (a single zip stream cannot be split across
	// its files), so the per-file rollup bytes come only from direct downloads.
	var expectedFileTotal int64
	for i, file := range files {
		want := expectedDirectPerFile[i] + archiveDownloads
		expectedFileTotal += want

		gotFile, err := b.GetFile(file.ID)
		require.NoError(t, err)
		require.Equal(t, want, gotFile.DownloadCount, "file %d download_count", i)
		require.Equal(t, want, sumDailyDownloads(t, b, common.DownloadStatsEntityFile, file.ID), "file %d daily rollup", i)
		require.Equal(t, expectedDirectPerFile[i]*1024, sumDailyDownloadBytes(t, b, common.DownloadStatsEntityFile, file.ID), "file %d daily rollup bytes", i)
	}

	// The upload row and its rollup count one download per direct call plus one
	// per archive call, and accumulate every direct (1024) and archive (2048)
	// call's egress bytes — the upload row/rollup get bytes from both kinds.
	const expectedUploadTotal = directDownloads + archiveDownloads
	const expectedUploadBytesTotal = directDownloads*1024 + archiveDownloads*2048
	gotUpload, err := b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(expectedUploadTotal), gotUpload.DownloadCount, "upload download_count")
	require.Equal(t, int64(expectedUploadTotal), sumDailyDownloads(t, b, common.DownloadStatsEntityUpload, upload.ID), "upload daily rollup")
	require.Equal(t, int64(expectedUploadBytesTotal), gotUpload.DownloadedBytes, "upload downloaded_bytes")
	require.Equal(t, int64(expectedUploadBytesTotal), sumDailyDownloadBytes(t, b, common.DownloadStatsEntityUpload, upload.ID), "upload daily rollup bytes")

	// The sum of file rollups must equal the sum of the per-file download counts.
	require.Equal(t, expectedFileTotal, sumDailyDownloads(t, b, common.DownloadStatsEntityFile, files[0].ID)+
		sumDailyDownloads(t, b, common.DownloadStatsEntityFile, files[1].ID)+
		sumDailyDownloads(t, b, common.DownloadStatsEntityFile, files[2].ID))

	// Every download event (direct or archive) increments the downloads counter,
	// and every direct/archive call's bytes increment the downloaded bytes
	// counter, once for the user, server, and token usage scopes.
	userStats, err := b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, int64(expectedUploadTotal), userStats.Usage.Downloads.Total, "user usage downloads")
	require.Equal(t, int64(expectedUploadBytesTotal), userStats.Usage.Downloads.Bytes, "user usage downloaded bytes")

	tokenStats, err := b.GetUserStatistics(user.ID, &token.Token)
	require.NoError(t, err)
	require.Equal(t, int64(expectedUploadTotal), tokenStats.Usage.Downloads.Total, "token usage downloads")
	require.Equal(t, int64(expectedUploadBytesTotal), tokenStats.Usage.Downloads.Bytes, "token usage downloaded bytes")

	serverStats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, int64(expectedUploadTotal), serverStats.Usage.Downloads.Total, "server usage downloads")
	require.Equal(t, int64(expectedUploadBytesTotal), serverStats.Usage.Downloads.Bytes, "server usage downloaded bytes")
}

// isBenignStatusMiss reports whether a fused file-status transition failed only
// because a concurrent removal already moved the file out of its expected
// status. UpdateFileStatus returns "<oldStatus> file not found" in that case;
// it is an expected race outcome in these tests, not a real error.
func isBenignStatusMiss(err error) bool {
	return err != nil && strings.Contains(err.Error(), "file not found")
}

// requireCurrentUsageZero asserts that every current usage scope touched by the
// racing tests (user, token, server) has settled back to zero. Because each
// test iteration fully removes its single upload, correct counters must return
// to exactly zero and must never drift negative from a double decrement.
func requireCurrentUsageZero(t *testing.T, b *Backend, userID string, token string, msg string) {
	t.Helper()

	userStats, err := b.GetUserStatistics(userID, nil)
	require.NoError(t, err)
	require.Equal(t, 0, userStats.Uploads, "%s: user current uploads", msg)
	require.Equal(t, 0, userStats.Files, "%s: user current files", msg)
	require.Equal(t, int64(0), userStats.TotalSize, "%s: user current size", msg)

	tokenStats, err := b.GetUserStatistics(userID, &token)
	require.NoError(t, err)
	require.Equal(t, 0, tokenStats.Uploads, "%s: token current uploads", msg)
	require.Equal(t, 0, tokenStats.Files, "%s: token current files", msg)
	require.Equal(t, int64(0), tokenStats.TotalSize, "%s: token current size", msg)

	serverStats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 0, serverStats.Uploads, "%s: server current uploads", msg)
	require.Equal(t, 0, serverStats.Files, "%s: server current files", msg)
	require.Equal(t, int64(0), serverStats.TotalSize, "%s: server current size", msg)
}

// TestBackend_RemoveUploadVsOneShotDecrementIsExact stresses upload removal
// racing with one-shot-style file decrements (UpdateFileStatus uploaded ->
// removed, each fused with its own current-counter decrement). Whatever the
// interleaving, every file must be decremented exactly once: current counters
// settle to zero and never drift negative from a double decrement, while
// lifetime counters are untouched by removal.
func TestBackend_RemoveUploadVsOneShotDecrementIsExact(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "removal-oneshot-user")
	token := user.NewToken()
	createUser(t, b, user)

	const iterations = 40
	const numFiles = 4
	const fileSize = int64(2000)

	for iter := range iterations {
		upload := &common.Upload{User: user.ID, Token: token.Token}
		files := make([]*common.File, numFiles)
		for i := range files {
			f := upload.NewFile()
			f.Size = fileSize
			f.Status = common.FileUploaded
			files[i] = f
		}
		createUpload(t, b, upload)

		var wg sync.WaitGroup
		errs := make(chan error, numFiles+1)
		for i := range files {
			f := files[i]
			wg.Go(func() {
				err := b.UpdateFileStatus(f, common.FileUploaded, common.FileRemoved)
				if err != nil && !isBenignStatusMiss(err) {
					errs <- err
				}
			})
		}
		wg.Go(func() {
			if err := b.RemoveUpload(upload.ID); err != nil {
				errs <- err
			}
		})
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err, "iteration %d", iter)
		}

		requireCurrentUsageZero(t, b, user.ID, token.Token, fmt.Sprintf("iteration %d", iter))

		userStats, err := b.GetUserStatistics(user.ID, nil)
		require.NoError(t, err)
		require.Equal(t, iter+1, userStats.Usage.Lifetime.Uploads, "iteration %d ever uploads", iter)
		require.Equal(t, (iter+1)*numFiles, userStats.Usage.Lifetime.Files, "iteration %d ever files", iter)
		require.Equal(t, int64(iter+1)*numFiles*fileSize, userStats.Usage.Lifetime.TotalSize, "iteration %d ever size", iter)
	}
}

// TestBackend_RemoveUploadVsCompletionIsExact stresses upload removal racing
// with concurrent file completions (UpdateFileStatus uploading -> uploaded,
// each fused with its current+lifetime increment). Every file that reached the
// uploaded state must have been either decremented by the removal or never left
// counted as current: current counters settle to zero (never leaking a stray
// increment), and lifetime files equal exactly the number of completions that
// succeeded before their upload was removed.
func TestBackend_RemoveUploadVsCompletionIsExact(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "removal-completion-user")
	token := user.NewToken()
	createUser(t, b, user)

	const iterations = 40
	const numFiles = 4
	const fileSize = int64(3000)

	var totalCompletions int64

	for iter := range iterations {
		upload := &common.Upload{User: user.ID, Token: token.Token}
		files := make([]*common.File, numFiles)
		for i := range files {
			f := upload.NewFile()
			f.Size = fileSize
			f.Status = common.FileUploading
			files[i] = f
		}
		createUpload(t, b, upload)

		var completions int64
		var wg sync.WaitGroup
		errs := make(chan error, numFiles+1)
		for i := range files {
			f := files[i]
			wg.Go(func() {
				err := b.UpdateFileStatus(f, common.FileUploading, common.FileUploaded)
				if err == nil {
					atomic.AddInt64(&completions, 1)
				} else if !isBenignStatusMiss(err) {
					errs <- err
				}
			})
		}
		wg.Go(func() {
			if err := b.RemoveUpload(upload.ID); err != nil {
				errs <- err
			}
		})
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err, "iteration %d", iter)
		}
		totalCompletions += completions

		requireCurrentUsageZero(t, b, user.ID, token.Token, fmt.Sprintf("iteration %d", iter))
	}

	// Lifetime files/size equal exactly the completions that reached the uploaded
	// state before their upload was removed; lifetime never moves backwards.
	userStats, err := b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, iterations, userStats.Usage.Lifetime.Uploads, "ever uploads")
	require.Equal(t, int(totalCompletions), userStats.Usage.Lifetime.Files, "ever files must equal successful completions")
	require.Equal(t, totalCompletions*fileSize, userStats.Usage.Lifetime.TotalSize, "ever size must equal successful completions")
}

// TestBackend_RemoveUserUploadsCountersExact verifies bulk user-upload removal
// keeps every usage scope exact across two tokens plus a tokenless upload:
// current counters drop to zero for user, server, and each token, while
// lifetime counters stay intact. The removal is structured as one race-free
// per-upload pass (no per-token aggregate N+1), which this asserts by value.
func TestBackend_RemoveUserUploadsCountersExact(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "bulk-remove-user")
	token1 := user.NewToken()
	token2 := user.NewToken()
	createUser(t, b, user)

	mk := func(tok string, size int64) {
		u := &common.Upload{User: user.ID, Token: tok}
		f := u.NewFile()
		f.Size = size
		f.Status = common.FileUploaded
		createUpload(t, b, u)
	}
	mk(token1.Token, 1000)
	mk(token1.Token, 2000)
	mk(token2.Token, 3000)
	mk("", 4000) // tokenless (web session) upload

	userStats, err := b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 4, userStats.Uploads)
	require.Equal(t, 4, userStats.Files)
	require.Equal(t, int64(10000), userStats.TotalSize)

	removed, err := b.RemoveUserUploads(user.ID, "")
	require.NoError(t, err)
	require.Equal(t, 4, removed)

	userStats, err = b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 0, userStats.Uploads, "user current uploads")
	require.Equal(t, 0, userStats.Files, "user current files")
	require.Equal(t, int64(0), userStats.TotalSize, "user current size")
	require.Equal(t, 4, userStats.Usage.Lifetime.Uploads, "user lifetime uploads")
	require.Equal(t, 4, userStats.Usage.Lifetime.Files, "user lifetime files")
	require.Equal(t, int64(10000), userStats.Usage.Lifetime.TotalSize, "user lifetime size")

	t1, err := b.GetUserStatistics(user.ID, &token1.Token)
	require.NoError(t, err)
	require.Equal(t, 0, t1.Uploads, "token1 current uploads")
	require.Equal(t, 0, t1.Files, "token1 current files")
	require.Equal(t, int64(0), t1.TotalSize, "token1 current size")
	require.Equal(t, 2, t1.Usage.Lifetime.Uploads, "token1 lifetime uploads")
	require.Equal(t, 2, t1.Usage.Lifetime.Files, "token1 lifetime files")
	require.Equal(t, int64(3000), t1.Usage.Lifetime.TotalSize, "token1 lifetime size")

	t2, err := b.GetUserStatistics(user.ID, &token2.Token)
	require.NoError(t, err)
	require.Equal(t, 0, t2.Uploads, "token2 current uploads")
	require.Equal(t, 0, t2.Files, "token2 current files")
	require.Equal(t, int64(0), t2.TotalSize, "token2 current size")
	require.Equal(t, 1, t2.Usage.Lifetime.Uploads, "token2 lifetime uploads")
	require.Equal(t, 1, t2.Usage.Lifetime.Files, "token2 lifetime files")
	require.Equal(t, int64(3000), t2.Usage.Lifetime.TotalSize, "token2 lifetime size")

	serverStats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 0, serverStats.Uploads, "server current uploads")
	require.Equal(t, 0, serverStats.Files, "server current files")
	require.Equal(t, int64(0), serverStats.TotalSize, "server current size")
	require.Equal(t, 4, serverStats.Usage.Lifetime.Uploads, "server lifetime uploads")
	require.Equal(t, 4, serverStats.Usage.Lifetime.Files, "server lifetime files")
	require.Equal(t, int64(10000), serverStats.Usage.Lifetime.TotalSize, "server lifetime size")
}

// TestBackend_ImportBackfillLifetimeCounterAgreement pins the contract
// documented in ARCHITECTURE.md ("Import, Export, and Repair"): for the same
// final upload/file rows, the import-rebuilt usage_stats counters and the
// migration-0011-backfilled counters must be identical, because both share
// the exact same removed/deleted-file approximation (a bare status cannot
// tell a successfully-uploaded-then-removed file from a failed-cleanup one).
// If a future change makes one path smarter than the other without updating
// both, this test catches the resulting drift.
//
// The dataset intentionally carries no downloads: download_count/
// last_downloaded_at columns don't exist before migration 0011 runs, so a
// pre-migration legacy row can never carry a nonzero historical download
// count. That asymmetry is pre-existing, documented ("Migration Backfill")
// behavior, unrelated to the file/size approximation under test here.
func TestBackend_ImportBackfillLifetimeCounterAgreement(t *testing.T) {
	aliceID := common.GetUserID(common.ProviderLocal, "agreement-alice")
	bobID := common.GetUserID(common.ProviderLocal, "agreement-bob")
	tokenStr := common.GenerateRandomID(32)
	now := time.Now().Truncate(time.Second)
	deletedAt := now.Add(time.Hour)

	uploadA := common.GenerateRandomID(16) // completed, retained
	uploadB := common.GenerateRandomID(16) // one-shot consumed, tokenless
	uploadC := common.GenerateRandomID(16) // failed cleanup, removed via upload removal
	uploadD := common.GenerateRandomID(16) // stream consumed
	uploadE := common.GenerateRandomID(16) // anonymous, completed

	// --- Path 1: import-rebuilt ---------------------------------------
	// A source backend holds the exact same final rows, inserted directly so
	// none of Backend's counter-mutating methods run; only its raw metadata
	// is exported and re-imported into a fresh backend.
	source := newTestMetadataBackend()

	require.NoError(t, source.db.Create(&common.User{ID: aliceID, Provider: common.ProviderLocal, Login: "agreement-alice", CreatedAt: now}).Error)
	require.NoError(t, source.db.Create(&common.User{ID: bobID, Provider: common.ProviderLocal, Login: "agreement-bob", CreatedAt: now}).Error)
	require.NoError(t, source.db.Create(&common.Token{Token: tokenStr, UserID: aliceID, CreatedAt: now}).Error)

	sourceUploads := []*common.Upload{
		{ID: uploadA, User: aliceID, Token: tokenStr, CreatedAt: now},
		{ID: uploadB, User: aliceID, OneShot: true, CreatedAt: now},
		{ID: uploadC, User: aliceID, Token: tokenStr, CreatedAt: now, DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: true}},
		{ID: uploadD, User: aliceID, Token: tokenStr, Stream: true, CreatedAt: now},
		{ID: uploadE, CreatedAt: now},
	}
	for _, u := range sourceUploads {
		require.NoError(t, source.db.Create(u).Error)
	}

	sourceFiles := []*common.File{
		{ID: common.GenerateRandomID(16), UploadID: uploadA, Status: common.FileUploaded, Size: 1000, CreatedAt: now},
		{ID: common.GenerateRandomID(16), UploadID: uploadB, Status: common.FileRemoved, Size: 2000, CreatedAt: now},
		{ID: common.GenerateRandomID(16), UploadID: uploadC, Status: common.FileRemoved, Size: 3000, CreatedAt: now},
		{ID: common.GenerateRandomID(16), UploadID: uploadD, Status: common.FileDeleted, Size: 4000, CreatedAt: now},
		{ID: common.GenerateRandomID(16), UploadID: uploadE, Status: common.FileUploaded, Size: 500, CreatedAt: now},
	}
	for _, f := range sourceFiles {
		require.NoError(t, source.db.Create(f).Error)
	}

	path := filepath.Join(t.TempDir(), "plik.metadata.agreement.snappy.gob")
	require.NoError(t, source.Export(path))
	shutdownTestMetadataBackend(source)

	imported := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(imported)
	require.NoError(t, imported.Import(path, &ImportOptions{}))

	// --- Path 2: migration-backfilled -----------------------------------
	// Same final rows, inserted as raw legacy rows into a pre-0011 database,
	// then migration 0011 backfills usage_stats on reopen.
	oldConfig := *metadataBackendConfig
	oldConfig.EraseFirst = true
	oldConfig.disableSchemaInit = true
	oldConfig.migrationFilter = func(migrations []*gormigrate.Migration) []*gormigrate.Migration {
		var filtered []*gormigrate.Migration
		for _, migration := range migrations {
			if migration.ID == "0011-stats" {
				continue
			}
			filtered = append(filtered, migration)
		}
		return filtered
	}
	if oldConfig.Driver == "sqlite3" {
		p := fmt.Sprintf("/tmp/plik.stats-agreement-%d.db", time.Now().UnixNano())
		defer func() { _ = os.Remove(p) }()
		oldConfig.ConnectionString = p
	}

	oldBackend, err := NewBackend(&oldConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)

	require.NoError(t, oldBackend.db.Table("users").Create(map[string]any{
		"id": aliceID, "provider": common.ProviderLocal, "login": "agreement-alice", "created_at": now,
	}).Error)
	require.NoError(t, oldBackend.db.Table("users").Create(map[string]any{
		"id": bobID, "provider": common.ProviderLocal, "login": "agreement-bob", "created_at": now,
	}).Error)
	require.NoError(t, oldBackend.db.Table("tokens").Create(map[string]any{
		"token": tokenStr, "user_id": aliceID, "created_at": now,
	}).Error)

	legacyUploads := []map[string]any{
		{"id": uploadA, "upload_token": common.GenerateRandomID(32), "user": aliceID, "token": tokenStr, "created_at": now},
		{"id": uploadB, "upload_token": common.GenerateRandomID(32), "user": aliceID, "token": "", "one_shot": true, "created_at": now},
		{"id": uploadC, "upload_token": common.GenerateRandomID(32), "user": aliceID, "token": tokenStr, "created_at": now, "deleted_at": deletedAt},
		{"id": uploadD, "upload_token": common.GenerateRandomID(32), "user": aliceID, "token": tokenStr, "stream": true, "created_at": now},
		{"id": uploadE, "upload_token": common.GenerateRandomID(32), "user": "", "token": "", "created_at": now},
	}
	for _, u := range legacyUploads {
		require.NoError(t, oldBackend.db.Table("uploads").Create(u).Error)
	}

	legacyFiles := []map[string]any{
		{"id": common.GenerateRandomID(16), "upload_id": uploadA, "status": common.FileUploaded, "size": 1000, "created_at": now},
		{"id": common.GenerateRandomID(16), "upload_id": uploadB, "status": common.FileRemoved, "size": 2000, "created_at": now},
		{"id": common.GenerateRandomID(16), "upload_id": uploadC, "status": common.FileRemoved, "size": 3000, "created_at": now},
		{"id": common.GenerateRandomID(16), "upload_id": uploadD, "status": common.FileDeleted, "size": 4000, "created_at": now},
		{"id": common.GenerateRandomID(16), "upload_id": uploadE, "status": common.FileUploaded, "size": 500, "created_at": now},
	}
	for _, f := range legacyFiles {
		require.NoError(t, oldBackend.db.Table("files").Create(f).Error)
	}

	require.NoError(t, oldBackend.Shutdown())

	newConfig := *metadataBackendConfig
	newConfig.ConnectionString = oldConfig.ConnectionString
	newConfig.EraseFirst = false
	newConfig.disableSchemaInit = true
	backfilled, err := NewBackend(&newConfig, common.NewConfiguration().NewLogger())
	require.NoError(t, err)
	defer shutdownTestMetadataBackend(backfilled)

	// --- Compare ---------------------------------------------------------
	requireEqualUserStats := func(scope string, a, b *common.UserStats) {
		require.Equal(t, a.Uploads, b.Uploads, "%s current uploads", scope)
		require.Equal(t, a.Files, b.Files, "%s current files", scope)
		require.Equal(t, a.TotalSize, b.TotalSize, "%s current size", scope)
		require.Equal(t, a.Usage.Lifetime.Uploads, b.Usage.Lifetime.Uploads, "%s ever uploads", scope)
		require.Equal(t, a.Usage.Lifetime.Files, b.Usage.Lifetime.Files, "%s ever files", scope)
		require.Equal(t, a.Usage.Lifetime.TotalSize, b.Usage.Lifetime.TotalSize, "%s ever size", scope)
	}

	importedAlice, err := imported.GetUserStatistics(aliceID, nil)
	require.NoError(t, err)
	backfilledAlice, err := backfilled.GetUserStatistics(aliceID, nil)
	require.NoError(t, err)
	requireEqualUserStats("alice user scope", importedAlice, backfilledAlice)
	require.NotZero(t, importedAlice.Usage.Lifetime.Files, "sanity: not both trivially zero")

	importedToken, err := imported.GetUserStatistics(aliceID, &tokenStr)
	require.NoError(t, err)
	backfilledToken, err := backfilled.GetUserStatistics(aliceID, &tokenStr)
	require.NoError(t, err)
	requireEqualUserStats("token scope", importedToken, backfilledToken)

	importedAnon, err := imported.GetUserStatistics(common.AnonymousUserUsageStatsID, nil)
	require.NoError(t, err)
	backfilledAnon, err := backfilled.GetUserStatistics(common.AnonymousUserUsageStatsID, nil)
	require.NoError(t, err)
	requireEqualUserStats("anonymous scope", importedAnon, backfilledAnon)

	importedServer, err := imported.GetServerStatistics()
	require.NoError(t, err)
	backfilledServer, err := backfilled.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, importedServer.Uploads, backfilledServer.Uploads, "server current uploads")
	require.Equal(t, importedServer.Files, backfilledServer.Files, "server current files")
	require.Equal(t, importedServer.TotalSize, backfilledServer.TotalSize, "server current size")
	require.Equal(t, importedServer.Usage.Lifetime.Uploads, backfilledServer.Usage.Lifetime.Uploads, "server ever uploads")
	require.Equal(t, importedServer.Usage.Lifetime.Files, backfilledServer.Usage.Lifetime.Files, "server ever files")
	require.Equal(t, importedServer.Usage.Lifetime.TotalSize, backfilledServer.Usage.Lifetime.TotalSize, "server ever size")
	require.Equal(t, importedServer.AnonymousUploads, backfilledServer.AnonymousUploads)
	require.Equal(t, importedServer.AnonymousSize, backfilledServer.AnonymousSize)
	require.Equal(t, importedServer.AnonymousUsage.Lifetime.Uploads, backfilledServer.AnonymousUsage.Lifetime.Uploads)
	require.Equal(t, importedServer.AnonymousUsage.Lifetime.TotalSize, backfilledServer.AnonymousUsage.Lifetime.TotalSize)
	require.Equal(t, importedServer.Usage.Current.Features.OneShotUploads, backfilledServer.Usage.Current.Features.OneShotUploads)
	require.Equal(t, importedServer.Usage.Lifetime.Features.OneShotUploads, backfilledServer.Usage.Lifetime.Features.OneShotUploads)
	require.Equal(t, importedServer.Usage.Current.Features.StreamUploads, backfilledServer.Usage.Current.Features.StreamUploads)
	require.Equal(t, importedServer.Usage.Lifetime.Features.StreamUploads, backfilledServer.Usage.Lifetime.Features.StreamUploads)
	require.Equal(t, importedServer.LifetimeUsers, backfilledServer.LifetimeUsers, "ever users")
	require.Equal(t, 2, importedServer.LifetimeUsers)
	// Server-scope Downloads is trivially 0==0 for this download-free dataset
	// (see the func comment), but including it in the compared fields makes
	// the intent of this agreement test self-evident: import-rebuilt and
	// migration-backfilled download counters must agree too, not just the
	// upload/file/size counters checked above.
	require.Equal(t, importedServer.Usage.Downloads.Total, backfilledServer.Usage.Downloads.Total, "server downloads")

	// Pin the actual approximation value once, in absolute terms, so a change
	// that makes both paths agree on a *different* (wrong) number doesn't
	// silently pass this test.
	require.Equal(t, 5, importedServer.Usage.Lifetime.Files)
	require.Equal(t, int64(10500), importedServer.Usage.Lifetime.TotalSize)
}
