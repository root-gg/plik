package metadata

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

// requireUploadRollup asserts the exact uploads/bytes of one upload_stats_daily
// bucket, keyed by its (day, user_id, token) primary key.
func requireUploadRollup(t *testing.T, b *Backend, day time.Time, userID string, token string, wantUploads int64, wantBytes int64) {
	t.Helper()
	var s common.UploadStatsDaily
	err := b.db.Where("day = ? AND user_id = ? AND token = ?", day, userID, token).First(&s).Error
	require.NoError(t, err, "upload_stats_daily row (day=%v user_id=%q token=%q)", day, userID, token)
	require.Equal(t, wantUploads, s.Uploads, "rollup uploads for (%v,%q,%q)", day, userID, token)
	require.Equal(t, wantBytes, s.Bytes, "rollup bytes for (%v,%q,%q)", day, userID, token)
}

// requireUploadedBytes asserts the usage_stats.uploaded_bytes counter for a scope.
func requireUploadedBytes(t *testing.T, b *Backend, userID string, token string, want int64) {
	t.Helper()
	var s common.UsageStats
	err := b.db.Where("user_id = ? AND token = ?", userID, token).First(&s).Error
	require.NoError(t, err, "usage_stats row (user_id=%q token=%q)", userID, token)
	require.Equal(t, want, s.UploadedBytes, "uploaded_bytes for (user_id=%q token=%q)", userID, token)
}

// requireServerUploadedBytes asserts the server-scope uploaded_bytes via
// sum-on-read (there is no server usage_stats row): server uploaded_bytes is the
// UploadedBytes total exposed on the server usage payload (Usage.Uploads.Bytes).
func requireServerUploadedBytes(t *testing.T, b *Backend, want int64) {
	t.Helper()
	stats, err := b.GetServerStatistics()
	require.NoError(t, err, "get server statistics")
	require.Equal(t, want, stats.Usage.Uploads.Bytes, "server uploaded_bytes (sum-on-read)")
}

func countUploadRollupRows(t *testing.T, b *Backend) int64 {
	t.Helper()
	var n int64
	require.NoError(t, b.db.Model(&common.UploadStatsDaily{}).Count(&n).Error)
	return n
}

// TestBackend_CreateUploadRecordsDailyRollup pins the +1 uploads rollup written
// inside the fused CreateUpload transaction, with attribution stored verbatim on
// the (day, user_id, token) key — including the anonymous "" user.
func TestBackend_CreateUploadRecordsDailyRollup(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())

	user := common.NewUser(common.ProviderLocal, "upload-rollup-user")
	token := user.NewToken()
	createUser(t, b, user)

	// Two uploads by the same (user, token) on the same day accumulate to 2.
	up1 := &common.Upload{User: user.ID, Token: token.Token}
	up2 := &common.Upload{User: user.ID, Token: token.Token}
	createUpload(t, b, up1)
	createUpload(t, b, up2)
	requireUploadRollup(t, b, today, user.ID, token.Token, 2, 0)

	// An anonymous upload attributes verbatim to ("", "") — no sentinel.
	anon := &common.Upload{}
	createUpload(t, b, anon)
	requireUploadRollup(t, b, today, "", "", 1, 0)
}

// TestBackend_UploadedBytesOnCompletion pins the SUCCESS-path wire-byte recording
// in the fused UpdateFile completion transaction: the completed file's size lands
// in both the daily rollup and every usage scope's uploaded_bytes counter.
func TestBackend_UploadedBytesOnCompletion(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())

	user := common.NewUser(common.ProviderLocal, "uploaded-bytes-user")
	token := user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Token: token.Token}
	file := upload.NewFile()
	file.Name = "f.txt"
	file.Status = common.FileMissing
	createUpload(t, b, upload)

	// Drive the AddFile completion transitions: Missing -> Uploading -> Uploaded.
	require.NoError(t, b.UpdateFileStatus(file, common.FileMissing, common.FileUploading))
	file.Size = 700
	file.Status = common.FileUploaded
	require.NoError(t, b.UpdateFile(file, common.FileUploading))

	// Rollup: 1 upload (CreateUpload) + 700 wire bytes (completion) on today's bucket.
	requireUploadRollup(t, b, today, user.ID, token.Token, 1, 700)
	// uploaded_bytes on user / server / token scopes.
	requireUploadedBytes(t, b, user.ID, "", 700)
	requireServerUploadedBytes(t, b, 700)
	requireUploadedBytes(t, b, user.ID, token.Token, 700)
}

// TestBackend_UploadedBytesOnStreamCompletion pins the same recording for the
// stream completion transition (FileUploading -> FileDeleted).
func TestBackend_UploadedBytesOnStreamCompletion(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())

	user := common.NewUser(common.ProviderLocal, "uploaded-bytes-stream-user")
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Stream: true}
	file := upload.NewFile()
	file.Name = "s.bin"
	file.Status = common.FileMissing
	createUpload(t, b, upload)

	require.NoError(t, b.UpdateFileStatus(file, common.FileMissing, common.FileUploading))
	file.Size = 500
	file.Status = common.FileDeleted
	require.NoError(t, b.UpdateFile(file, common.FileUploading))

	requireUploadRollup(t, b, today, user.ID, "", 1, 500)
	requireUploadedBytes(t, b, user.ID, "", 500)
	requireServerUploadedBytes(t, b, 500)
}

// TestBackend_RecordUploadedBytes pins the best-effort partial-failure path
// (server/handlers/add_file.go): it adds wire bytes to today's rollup and every
// usage scope without touching the hot upload/file rows, and is a no-op for
// nil/zero/negative inputs.
func TestBackend_RecordUploadedBytes(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())

	user := common.NewUser(common.ProviderLocal, "partial-upload-user")
	token := user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Token: token.Token}
	createUpload(t, b, upload) // rollup uploads=1, bytes=0

	// A partial/aborted transfer records only bytes on today's bucket.
	require.NoError(t, b.RecordUploadedBytes(upload, 321))
	requireUploadRollup(t, b, today, user.ID, token.Token, 1, 321)
	requireUploadedBytes(t, b, user.ID, "", 321)
	requireServerUploadedBytes(t, b, 321)
	requireUploadedBytes(t, b, user.ID, token.Token, 321)

	// A second partial recording accumulates.
	require.NoError(t, b.RecordUploadedBytes(upload, 79))
	requireUploadRollup(t, b, today, user.ID, token.Token, 1, 400)

	// No-ops: nil upload, zero, negative.
	require.NoError(t, b.RecordUploadedBytes(nil, 100))
	require.NoError(t, b.RecordUploadedBytes(upload, 0))
	require.NoError(t, b.RecordUploadedBytes(upload, -5))
	requireUploadRollup(t, b, today, user.ID, token.Token, 1, 400)
}

// TestBackend_RecordUploadedBytesConcurrent pins RecordUploadedBytes' byte
// exactness under many concurrent best-effort partial-upload recordings — the
// concurrent uploaded-bytes test the byte-accounting surface previously
// lacked, mirroring TestBackend_RecordFileDownloadConcurrent on the download
// side (server/metadata/stats_test.go).
func TestBackend_RecordUploadedBytesConcurrent(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())

	user := common.NewUser(common.ProviderLocal, "concurrent-upload-bytes-user")
	token := user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Token: token.Token}
	createUpload(t, b, upload) // rollup uploads=1, bytes=0

	const count = 20
	const bytesPerCall = 512
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for range count {
		wg.Go(func() {
			errs <- b.RecordUploadedBytes(upload, bytesPerCall)
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	requireUploadRollup(t, b, today, user.ID, token.Token, 1, count*bytesPerCall)
	requireUploadedBytes(t, b, user.ID, "", count*bytesPerCall)
	requireServerUploadedBytes(t, b, count*bytesPerCall)
	requireUploadedBytes(t, b, user.ID, token.Token, count*bytesPerCall)
}

// TestBackend_UploadRollupAttributionImmutable proves attribution is the primary
// key: repeated recordings for the same (day, user, token) increment one row,
// while a different attribution creates a distinct row.
func TestBackend_UploadRollupAttributionImmutable(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())

	userA := common.NewUser(common.ProviderLocal, "rollup-attr-a")
	userB := common.NewUser(common.ProviderLocal, "rollup-attr-b")
	createUser(t, b, userA)
	createUser(t, b, userB)

	upA := &common.Upload{User: userA.ID}
	createUpload(t, b, upA)
	require.NoError(t, b.RecordUploadedBytes(upA, 100))
	require.NoError(t, b.RecordUploadedBytes(upA, 50))

	upB := &common.Upload{User: userB.ID}
	createUpload(t, b, upB)
	require.NoError(t, b.RecordUploadedBytes(upB, 999))

	requireUploadRollup(t, b, today, userA.ID, "", 1, 150)
	requireUploadRollup(t, b, today, userB.ID, "", 1, 999)
	require.Equal(t, int64(2), countUploadRollupRows(t, b), "two distinct attribution pairs -> two rows")
}

// TestBackend_GetUserActivityStatsDailyUploads pins the exact per-day upload
// series: dense (zero-filled gaps), scoped to the requested user, oldest-first.
func TestBackend_GetUserActivityStatsDailyUploads(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	userA := common.NewUser(common.ProviderLocal, "upload-series-a")
	userB := common.NewUser(common.ProviderLocal, "upload-series-b")
	createUser(t, b, userA)
	createUser(t, b, userB)

	today := statsDay(time.Now())
	rows := []*common.UploadStatsDaily{
		{Day: today, UserID: userA.ID, Uploads: 3, Bytes: 300},
		{Day: today.AddDate(0, 0, -2), UserID: userA.ID, Uploads: 5, Bytes: 500},
		{Day: today, UserID: userB.ID, Uploads: 7, Bytes: 700}, // excluded from A
		{Day: today, UserID: "", Uploads: 9, Bytes: 900},       // anonymous, excluded from A
	}
	for _, r := range rows {
		require.NoError(t, b.CreateUploadStatsDaily(r))
	}

	series, err := b.GetUserActivityStatsDaily(userA.ID, 3)
	require.NoError(t, err)
	require.Len(t, series, 3)

	require.Equal(t, today.AddDate(0, 0, -2), series[0].Day)
	require.Equal(t, int64(5), series[0].Uploads)
	require.Equal(t, int64(500), series[0].UploadedBytes)

	require.Equal(t, int64(0), series[1].Uploads, "gap day zero-filled")
	require.Equal(t, int64(0), series[1].UploadedBytes)

	require.Equal(t, today, series[2].Day)
	require.Equal(t, int64(3), series[2].Uploads)
	require.Equal(t, int64(300), series[2].UploadedBytes)
}

// TestBackend_GetServerActivityStatsDailyUploads pins the server-wide upload
// series (summed across every attribution) and its 30-day window edge.
func TestBackend_GetServerActivityStatsDailyUploads(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())
	rows := []*common.UploadStatsDaily{
		{Day: today, UserID: "u1", Uploads: 1, Bytes: 111},
		{Day: today, UserID: "", Uploads: 2, Bytes: 222}, // anonymous same day sums in
		{Day: today.AddDate(0, 0, -29), UserID: "u1", Uploads: 4, Bytes: 444},
		{Day: today.AddDate(0, 0, -30), UserID: "u1", Uploads: 8, Bytes: 888}, // outside 30-day window
	}
	for _, r := range rows {
		require.NoError(t, b.CreateUploadStatsDaily(r))
	}

	series, err := b.GetServerActivityStatsDaily(30)
	require.NoError(t, err)
	require.Len(t, series, 30)
	require.Equal(t, today.AddDate(0, 0, -29), series[0].Day)
	require.Equal(t, today, series[29].Day)

	require.Equal(t, int64(3), series[29].Uploads, "today sums u1 + anonymous")
	require.Equal(t, int64(333), series[29].UploadedBytes)
	require.Equal(t, int64(4), series[0].Uploads, "the -29d edge is inside the window")

	var total int64
	for _, p := range series {
		total += p.Uploads
	}
	require.Equal(t, int64(7), total, "the -30d row is outside a 30-day series")
}

// TestBackend_UploadSeriesSurvivesUploadDeletion mirrors the download-side
// guarantee: the rollup is attributed on its own row, so deleting the upload
// leaves the owner's upload series intact.
func TestBackend_UploadSeriesSurvivesUploadDeletion(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := common.NewUser(common.ProviderLocal, "upload-series-deleted")
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID}
	file := upload.NewFile()
	file.Name = "d.txt"
	file.Status = common.FileMissing
	createUpload(t, b, upload)
	require.NoError(t, b.UpdateFileStatus(file, common.FileMissing, common.FileUploading))
	file.Size = 256
	file.Status = common.FileUploaded
	require.NoError(t, b.UpdateFile(file, common.FileUploading))

	series, err := b.GetUserActivityStatsDaily(user.ID, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), series[0].Uploads)
	require.Equal(t, int64(256), series[0].UploadedBytes)

	require.NoError(t, b.RemoveUpload(upload.ID))

	series, err = b.GetUserActivityStatsDaily(user.ID, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), series[0].Uploads, "deleted upload's creation stays in the user series")
	require.Equal(t, int64(256), series[0].UploadedBytes)
}

// TestBackend_DeleteExpiredUploadStatsDaily pins the today+30 retention.
func TestBackend_DeleteExpiredUploadStatsDaily(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	today := statsDay(time.Now())
	rows := []*common.UploadStatsDaily{
		{Day: today, UserID: "u", Uploads: 1},
		{Day: today.AddDate(0, 0, -30), UserID: "u", Uploads: 1},  // kept (boundary)
		{Day: today.AddDate(0, 0, -31), UserID: "u", Uploads: 1},  // pruned
		{Day: today.AddDate(0, 0, -100), UserID: "u", Uploads: 1}, // pruned
	}
	for _, r := range rows {
		require.NoError(t, b.CreateUploadStatsDaily(r))
	}

	deleted, err := b.DeleteExpiredUploadStatsDaily()
	require.NoError(t, err)
	require.Equal(t, 2, deleted, "rows older than today-30 are pruned")
	require.Equal(t, int64(2), countUploadRollupRows(t, b), "today and today-30 are kept")
}
