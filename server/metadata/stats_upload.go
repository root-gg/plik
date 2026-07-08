package metadata

import (
	"time"

	"github.com/root-gg/plik/server/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// This file contains upload rollup persistence and the merged activity daily
// series. upload_stats_daily holds one bounded row per UTC day per (user_id,
// token) attribution pair: Uploads counts upload creations and Bytes counts the
// wire bytes received (ingress). It is the upload-side twin of
// download_stats_daily (stats_download.go).
//
// Canonical stats-write lock order (see stats_download.go for the full contract):
// upload row -> file rows (asc) -> rollups -> usage rows. The two upload writers
// obey it: recordDailyUploads is called inside CreateUpload (after the upload row
// is created, before the usage rows) and inside incrementUsageForCompletedFile
// (after the upload row is locked and the file row updated, before the usage
// rows). The best-effort partial-failure recorder (RecordUploadedBytes) runs
// single-shot: it touches only the rollup then the usage rows, a safe suffix of
// the canonical order, exactly like a bytes-only download recording.
//
// Uploaded bytes (ingress) counting policy
// -----------------------------------------
// The AddFile handler wraps the client file stream in a counting io.Reader
// (server/handlers/add_file.go). On SUCCESSFUL completion the fused UpdateFile
// transaction records the completed file's Size — which, for a fully consumed
// stream, is exactly the counted wire bytes — via incrementUsageForCompletedFile.
// On a stream/abort/backend FAILURE before completion the handler calls
// RecordUploadedBytes with the partial counted bytes, single-shot best-effort:
// a recording failure is logged and the request is unaffected. Multipart
// framing overhead is excluded (the part reader strips it). Both destinations —
// upload_stats_daily.bytes and usage_stats.uploaded_bytes — are updated together.

// recordDailyUploads atomically increments one daily upload rollup bucket's
// uploads and/or byte counters. The day must already be normalized with
// statsDay. Only the non-zero counters are incremented, so a bytes-only
// recording bumps bytes without touching uploads and vice versa. userID and token
// are the attribution pair; because they are part of the primary key the row's
// attribution is naturally immutable across increments.
func (b *Backend) recordDailyUploads(tx *gorm.DB, day time.Time, userID string, token string, uploads int64, bytes int64) error {
	if uploads <= 0 && bytes <= 0 {
		return nil
	}

	now := time.Now()
	stat := &common.UploadStatsDaily{
		Day:       day,
		UserID:    userID,
		Token:     token,
		Uploads:   uploads,
		Bytes:     bytes,
		UpdatedAt: now,
	}

	updates := map[string]any{"updated_at": now}
	if uploads > 0 {
		updates["uploads"] = incrementColumn("upload_stats_daily", "uploads", uploads)
	}
	if bytes > 0 {
		updates["bytes"] = incrementColumn("upload_stats_daily", "bytes", bytes)
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "day"},
			{Name: "user_id"},
			{Name: "token"},
		},
		DoUpdates: clause.Assignments(updates),
	}).Create(stat).Error
}

// CreateUploadStatsDaily persists one daily upload rollup during import or tests.
// Unlike CreateDownloadStatsDaily's plain insert, this upserts absolute values on
// conflict: import replays CreateUpload first (which regenerates each day's
// Uploads count with Bytes=0 because the wire-byte path is not replayed), then
// this restores the authoritative {uploads, bytes} row exported from the source,
// so the imported table matches the export exactly instead of colliding on the
// (day, user_id, token) primary key.
func (b *Backend) CreateUploadStatsDaily(stats *common.UploadStatsDaily) error {
	return b.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "day"},
			{Name: "user_id"},
			{Name: "token"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"uploads", "bytes", "updated_at"}),
	}).Create(stats).Error
}

// ForEachUploadStatsDaily executes f for every daily upload stats row. Export
// uses this to stream rollups without loading the whole table.
func (b *Backend) ForEachUploadStatsDaily(f func(stats *common.UploadStatsDaily) error) error {
	rows, err := b.db.Model(&common.UploadStatsDaily{}).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		stats := &common.UploadStatsDaily{}
		err = b.db.ScanRows(rows, stats)
		if err != nil {
			return err
		}
		err = f(stats)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteExpiredUploadStatsDaily prunes old daily upload rollups with the same
// today+30 policy as download rollups: it keeps today's UTC bucket plus the
// previous 30 UTC buckets and deletes anything older.
func (b *Backend) DeleteExpiredUploadStatsDaily() (int, error) {
	cutoff := statsDay(time.Now()).AddDate(0, 0, -30)
	result := b.db.
		Where("day < ?", cutoff).
		Delete(&common.UploadStatsDaily{})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

// incrementUploadedBytesUsage applies wire-byte ingress to the user/anonymous and
// token usage rows in the canonical lock order. Mirror of incrementDownloadUsage's
// byte handling. There is no server row; server bytes are summed on read.
func (b *Backend) incrementUploadedBytesUsage(tx *gorm.DB, upload *common.Upload, bytes int64) error {
	delta := usageDelta{uploadedBytes: bytes}
	return b.applyUsageDelta(tx, upload.User, upload.Token, delta)
}

// recordUploadedBytesTx records wire bytes for one upload into the daily rollup
// (for the given normalized day) and the usage_stats.uploaded_bytes counter, in
// canonical lock order (rollup then usage rows). It is the shared body of the
// live partial-failure path (RecordUploadedBytes, day=today) and the fakedb byte
// seeder (FixtureSeedUploadedBytes, day=upload creation day).
func (b *Backend) recordUploadedBytesTx(tx *gorm.DB, upload *common.Upload, day time.Time, bytes int64) error {
	err := b.recordDailyUploads(tx, day, upload.User, upload.Token, 0, bytes)
	if err != nil {
		return err
	}
	return b.incrementUploadedBytesUsage(tx, upload, bytes)
}

// RecordUploadedBytes records wire bytes received for an upload whose transfer
// did NOT reach the fused completion transaction (a stream error, aborted
// transfer, or backend write failure in AddFile). It is single-shot best-effort:
// the caller logs any error and lets the failing request return regardless.
// It attributes the partial bytes to today's UTC bucket and the usage rows, never
// touching the hot upload/file rows (there is no completed file to credit).
func (b *Backend) RecordUploadedBytes(upload *common.Upload, bytes int64) error {
	if upload == nil || bytes <= 0 {
		return nil
	}
	day := statsDay(time.Now())
	return b.db.Transaction(func(tx *gorm.DB) error {
		return b.recordUploadedBytesTx(tx, upload, day, bytes)
	})
}

// FixtureSeedUploadedBytes adds wire-byte ingress for an upload to the day of
// its creation, incrementing both the daily rollup and the
// usage_stats.uploaded_bytes counter. It exists for fixture code (`plikd
// fakedb`, server/cmd/fakedb.go), which has already created the upload (so
// CreateUpload wrote the day's Uploads +1 rollup) and now needs the matching
// wire-byte totals. It must not be used on the live request path.
func (b *Backend) FixtureSeedUploadedBytes(upload *common.Upload, bytes int64) error {
	if upload == nil || bytes <= 0 {
		return nil
	}
	day := statsDay(upload.CreatedAt)
	return b.db.Transaction(func(tx *gorm.DB) error {
		return b.recordUploadedBytesTx(tx, upload, day, bytes)
	})
}

// GetUserActivityStatsDaily returns a dense per-day activity series for userID,
// covering the last days UTC days (oldest first, today last). Attribution is read
// from the rollup rows themselves (written once at record time, never updated),
// so the series still includes days whose upload has since been deleted.
func (b *Backend) GetUserActivityStatsDaily(userID string, days int) ([]*common.ActivityDailyPoint, error) {
	return b.getActivityStatsDailySeries(days, &userID)
}

// GetServerActivityStatsDaily returns the server-wide activity series summed
// across every user, anonymous upload, and token, for the admin dashboard chart.
func (b *Backend) GetServerActivityStatsDaily(days int) ([]*common.ActivityDailyPoint, error) {
	return b.getActivityStatsDailySeries(days, nil)
}

// getActivityStatsDailySeries backs both GetUserActivityStatsDaily and
// GetServerActivityStatsDaily and is the single query set that feeds all four
// activity measures (so the bounded windows derived from it can never drift from
// the chart). It sums upload-entity download rows and upload rows by day for the
// last days UTC days, optionally scoped to one user_id, and returns a dense,
// oldest-first slice of exactly `days` points with zero-filled gaps. Downloads
// use only the upload-entity rows (file-entity rows would double-count the same
// logical download, matching getDownloadStatsDailySeries' original behavior).
func (b *Backend) getActivityStatsDailySeries(days int, userID *string) ([]*common.ActivityDailyPoint, error) {
	if days <= 0 {
		return nil, nil
	}

	today := statsDay(time.Now())
	since := today.AddDate(0, 0, -(days - 1))

	points := make([]*common.ActivityDailyPoint, days)
	index := make(map[int64]*common.ActivityDailyPoint, days)
	for i := range days {
		day := since.AddDate(0, 0, i)
		points[i] = &common.ActivityDailyPoint{Day: day}
		index[day.Unix()] = points[i]
	}

	// Download side: upload-entity rows only.
	type downloadRow struct {
		Day       time.Time
		Downloads int64
		Bytes     int64
	}
	dq := b.db.Model(&common.DownloadStatsDaily{}).
		Select("day, sum(downloads) as downloads, sum(bytes) as bytes").
		Where("entity_type = ?", common.DownloadStatsEntityUpload).
		Where("day >= ?", since).
		Group("day")
	if userID != nil {
		dq = dq.Where("user_id = ?", *userID)
	}
	var dRows []downloadRow
	if err := dq.Scan(&dRows).Error; err != nil {
		return nil, err
	}
	for _, r := range dRows {
		if p, ok := index[statsDay(r.Day).Unix()]; ok {
			p.Downloads = r.Downloads
			p.DownloadedBytes = r.Bytes
		}
	}

	// Upload side.
	type uploadRow struct {
		Day     time.Time
		Uploads int64
		Bytes   int64
	}
	uq := b.db.Model(&common.UploadStatsDaily{}).
		Select("day, sum(uploads) as uploads, sum(bytes) as bytes").
		Where("day >= ?", since).
		Group("day")
	if userID != nil {
		uq = uq.Where("user_id = ?", *userID)
	}
	var uRows []uploadRow
	if err := uq.Scan(&uRows).Error; err != nil {
		return nil, err
	}
	for _, r := range uRows {
		if p, ok := index[statsDay(r.Day).Unix()]; ok {
			p.Uploads = r.Uploads
			p.UploadedBytes = r.Bytes
		}
	}

	return points, nil
}

// sumActivityPoints sums the field selected by sel over the last n points of a
// dense oldest-first activity series (i.e. the most recent n UTC days). Used to
// derive the bounded upload/download windows from the same per-day series the
// chart draws, so the two never drift apart.
func sumActivityPoints(points []*common.ActivityDailyPoint, n int, sel func(*common.ActivityDailyPoint) int64) int64 {
	if n > len(points) {
		n = len(points)
	}
	var sum int64
	for _, point := range points[len(points)-n:] {
		sum += sel(point)
	}
	return sum
}

// applyActivityWindows sets the bounded download AND upload count windows (today
// / last 7 / last 30 days) on a usage response from a dense oldest-first activity
// series. It is called only for scopes backed by a series (server usage, user
// usage); other scopes leave the windows nil so they are omitted from the JSON.
func applyActivityWindows(resp *common.UsageStatsResponse, points []*common.ActivityDailyPoint) {
	if resp == nil {
		return
	}
	dToday := sumActivityPoints(points, 1, func(p *common.ActivityDailyPoint) int64 { return p.Downloads })
	d7 := sumActivityPoints(points, 7, func(p *common.ActivityDailyPoint) int64 { return p.Downloads })
	d30 := sumActivityPoints(points, activityWindowDays, func(p *common.ActivityDailyPoint) int64 { return p.Downloads })
	resp.Downloads.Today = &dToday
	resp.Downloads.Last7Days = &d7
	resp.Downloads.Last30Days = &d30

	uToday := sumActivityPoints(points, 1, func(p *common.ActivityDailyPoint) int64 { return p.Uploads })
	u7 := sumActivityPoints(points, 7, func(p *common.ActivityDailyPoint) int64 { return p.Uploads })
	u30 := sumActivityPoints(points, activityWindowDays, func(p *common.ActivityDailyPoint) int64 { return p.Uploads })
	resp.Uploads.Today = &uToday
	resp.Uploads.Last7Days = &u7
	resp.Uploads.Last30Days = &u30
}
