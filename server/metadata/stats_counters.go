package metadata

import (
	"strings"
	"time"

	"github.com/root-gg/plik/server/common"
)

// This file is the single source of truth for the usage_stats counter columns.
//
// Every counter is declared exactly once, in the usageCounters registry below.
// The write-side plumbing that used to be a set of hand-synchronized column
// lists — the increment SET map, the lazy row-creation literal, and the
// migration backfill row constructor — is now derived from this registry.
// Adding a counter is a two-line change: one field on
// common.UsageStats (the schema) and one counterSpec entry here. A missing
// entry is caught by TestUsageCountersRegistryMatchesSchema, which reflects over
// the struct's gorm column tags and asserts set-equality with the registry.
//
// usageDelta and common.UsageStats deliberately stay plain structs with named
// fields: callers build deltas with readable field assignments and the compiler
// still type-checks every access. The registry only closes over those fields; it
// does not replace them with a map or reflection on the hot path.

// usageDelta is intentionally made of signed deltas, not absolute values.
// Metadata mutations add or subtract these counters inside the same DB
// transaction as the upload/file/user change so multiple Plik instances can
// share one database without lost updates.
type usageDelta struct {
	currentUploads               int
	currentFiles                 int
	currentSize                  int64
	lifetimeUploads              int
	lifetimeFiles                int
	lifetimeSize                 int64
	downloads                    int64
	downloadedBytes              int64
	uploadedBytes                int64
	lifetimeUsers                int
	currentPasswordUploads       int
	lifetimePasswordUploads      int
	currentRemovableUploads      int
	lifetimeRemovableUploads     int
	currentOneShotUploads        int
	lifetimeOneShotUploads       int
	currentStreamUploads         int
	lifetimeStreamUploads        int
	currentExtendTTLUploads      int
	lifetimeExtendTTLUploads     int
	currentE2EEUploads           int
	lifetimeE2EEUploads          int
	currentCommentUploads        int
	lifetimeCommentUploads       int
	currentTTLNoneUploads        int
	lifetimeTTLNoneUploads       int
	currentTTLLt1hUploads        int
	lifetimeTTLLt1hUploads       int
	currentTTL1h1dUploads        int
	lifetimeTTL1h1dUploads       int
	currentTTL1d7dUploads        int
	lifetimeTTL1d7dUploads       int
	currentTTL7d30dUploads       int
	lifetimeTTL7d30dUploads      int
	currentTTLGt30dUploads       int
	lifetimeTTLGt30dUploads      int
	currentFileSizeLt1mFiles     int
	lifetimeFileSizeLt1mFiles    int
	currentFileSize1m10mFiles    int
	lifetimeFileSize1m10mFiles   int
	currentFileSize10m100mFiles  int
	lifetimeFileSize10m100mFiles int
	currentFileSize100m1gFiles   int
	lifetimeFileSize100m1gFiles  int
	currentFileSize1g10gFiles    int
	lifetimeFileSize1g10gFiles   int
	currentFileSize10g100gFiles  int
	lifetimeFileSize10g100gFiles int
	currentFileSizeGt100gFiles   int
	lifetimeFileSizeGt100gFiles  int

	// lastUploadAt is bookkeeping, not a counter: it is a "max", not a signed
	// sum, so it is handled explicitly by the increment/create paths and stays
	// out of the registry.
	lastUploadAt *time.Time
}

// counterSpec is one row of the counter registry. It ties a usage_stats DB
// column to the usageDelta field that carries its signed delta and to the
// common.UsageStats field that holds its absolute value, through tiny closures
// so no per-counter code has to be repeated across the write path.
type counterSpec struct {
	// column is the usage_stats DB column, identical to the field's
	// gorm:"column:..." tag on common.UsageStats.
	column string
	// get reads this counter's signed delta from a usageDelta.
	get func(d *usageDelta) int64
	// set writes an absolute value onto a common.UsageStats row.
	set func(s *common.UsageStats, v int64)
	// getStats reads this counter's absolute value from a common.UsageStats row.
	// It is the inverse of set and drives the server sum-on-read select list and
	// the DeleteUser tombstone fold, so neither hand-lists the counter columns.
	getStats func(s *common.UsageStats) int64
}

// usageCounters declares every counter column exactly once. The order is the
// struct field order for readability; nothing depends on it.
var usageCounters = []counterSpec{
	{
		column:   "current_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentUploads) },
	},
	{
		column:   "current_files",
		get:      func(d *usageDelta) int64 { return int64(d.currentFiles) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentFiles = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentFiles) },
	},
	{
		column:   "current_size",
		get:      func(d *usageDelta) int64 { return d.currentSize },
		set:      func(s *common.UsageStats, v int64) { s.CurrentSize = v },
		getStats: func(s *common.UsageStats) int64 { return s.CurrentSize },
	},
	{
		column:   "lifetime_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeUploads) },
	},
	{
		column:   "lifetime_files",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeFiles) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeFiles = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeFiles) },
	},
	{
		column:   "lifetime_size",
		get:      func(d *usageDelta) int64 { return d.lifetimeSize },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeSize = v },
		getStats: func(s *common.UsageStats) int64 { return s.LifetimeSize },
	},
	{
		column:   "downloads",
		get:      func(d *usageDelta) int64 { return d.downloads },
		set:      func(s *common.UsageStats, v int64) { s.Downloads = v },
		getStats: func(s *common.UsageStats) int64 { return s.Downloads },
	},
	{
		column:   "downloaded_bytes",
		get:      func(d *usageDelta) int64 { return d.downloadedBytes },
		set:      func(s *common.UsageStats, v int64) { s.DownloadedBytes = v },
		getStats: func(s *common.UsageStats) int64 { return s.DownloadedBytes },
	},
	{
		column:   "uploaded_bytes",
		get:      func(d *usageDelta) int64 { return d.uploadedBytes },
		set:      func(s *common.UsageStats, v int64) { s.UploadedBytes = v },
		getStats: func(s *common.UsageStats) int64 { return s.UploadedBytes },
	},
	{
		column:   "lifetime_users",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeUsers) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeUsers = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeUsers) },
	},
	{
		column:   "current_password_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentPasswordUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentPasswordUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentPasswordUploads) },
	},
	{
		column:   "lifetime_password_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimePasswordUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimePasswordUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimePasswordUploads) },
	},
	{
		column:   "current_removable_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentRemovableUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentRemovableUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentRemovableUploads) },
	},
	{
		column:   "lifetime_removable_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeRemovableUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeRemovableUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeRemovableUploads) },
	},
	{
		column:   "current_one_shot_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentOneShotUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentOneShotUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentOneShotUploads) },
	},
	{
		column:   "lifetime_one_shot_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeOneShotUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeOneShotUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeOneShotUploads) },
	},
	{
		column:   "current_stream_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentStreamUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentStreamUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentStreamUploads) },
	},
	{
		column:   "lifetime_stream_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeStreamUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeStreamUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeStreamUploads) },
	},
	{
		column:   "current_extend_ttl_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentExtendTTLUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentExtendTTLUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentExtendTTLUploads) },
	},
	{
		column:   "lifetime_extend_ttl_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeExtendTTLUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeExtendTTLUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeExtendTTLUploads) },
	},
	{
		column:   "current_e2ee_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentE2EEUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentE2EEUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentE2EEUploads) },
	},
	{
		column:   "lifetime_e2ee_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeE2EEUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeE2EEUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeE2EEUploads) },
	},
	{
		column:   "current_comment_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentCommentUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentCommentUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentCommentUploads) },
	},
	{
		column:   "lifetime_comment_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeCommentUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeCommentUploads = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeCommentUploads) },
	},
	{
		column:   "current_ttl_none_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentTTLNoneUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentUploadTTLNone = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentUploadTTLNone) },
	},
	{
		column:   "lifetime_ttl_none_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeTTLNoneUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeUploadTTLNone = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeUploadTTLNone) },
	},
	{
		column:   "current_ttl_lt1h_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentTTLLt1hUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentUploadTTLLessThan1Hour = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentUploadTTLLessThan1Hour) },
	},
	{
		column:   "lifetime_ttl_lt1h_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeTTLLt1hUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeUploadTTLLessThan1Hour = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeUploadTTLLessThan1Hour) },
	},
	{
		column:   "current_ttl_1h1d_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentTTL1h1dUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentUploadTTL1HourTo1Day = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentUploadTTL1HourTo1Day) },
	},
	{
		column:   "lifetime_ttl_1h1d_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeTTL1h1dUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeUploadTTL1HourTo1Day = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeUploadTTL1HourTo1Day) },
	},
	{
		column:   "current_ttl_1d7d_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentTTL1d7dUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentUploadTTL1DayTo7Days = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentUploadTTL1DayTo7Days) },
	},
	{
		column:   "lifetime_ttl_1d7d_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeTTL1d7dUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeUploadTTL1DayTo7Days = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeUploadTTL1DayTo7Days) },
	},
	{
		column:   "current_ttl_7d30d_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentTTL7d30dUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentUploadTTL7DaysTo30Days = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentUploadTTL7DaysTo30Days) },
	},
	{
		column:   "lifetime_ttl_7d30d_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeTTL7d30dUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeUploadTTL7DaysTo30Days = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeUploadTTL7DaysTo30Days) },
	},
	{
		column:   "current_ttl_gt30d_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.currentTTLGt30dUploads) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentUploadTTLGreaterThan30Days = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentUploadTTLGreaterThan30Days) },
	},
	{
		column:   "lifetime_ttl_gt30d_uploads",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeTTLGt30dUploads) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeUploadTTLGreaterThan30Days = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeUploadTTLGreaterThan30Days) },
	},
	{
		column:   "current_file_size_lt1m_files",
		get:      func(d *usageDelta) int64 { return int64(d.currentFileSizeLt1mFiles) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentFilesLessThan1MB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentFilesLessThan1MB) },
	},
	{
		column:   "lifetime_file_size_lt1m_files",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeFileSizeLt1mFiles) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeFilesLessThan1MB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeFilesLessThan1MB) },
	},
	{
		column:   "current_file_size_1m10m_files",
		get:      func(d *usageDelta) int64 { return int64(d.currentFileSize1m10mFiles) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentFiles1MBTo10MB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentFiles1MBTo10MB) },
	},
	{
		column:   "lifetime_file_size_1m10m_files",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeFileSize1m10mFiles) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeFiles1MBTo10MB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeFiles1MBTo10MB) },
	},
	{
		column:   "current_file_size_10m100m_files",
		get:      func(d *usageDelta) int64 { return int64(d.currentFileSize10m100mFiles) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentFiles10MBTo100MB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentFiles10MBTo100MB) },
	},
	{
		column:   "lifetime_file_size_10m100m_files",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeFileSize10m100mFiles) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeFiles10MBTo100MB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeFiles10MBTo100MB) },
	},
	{
		column:   "current_file_size_100m1g_files",
		get:      func(d *usageDelta) int64 { return int64(d.currentFileSize100m1gFiles) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentFiles100MBTo1GB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentFiles100MBTo1GB) },
	},
	{
		column:   "lifetime_file_size_100m1g_files",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeFileSize100m1gFiles) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeFiles100MBTo1GB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeFiles100MBTo1GB) },
	},
	{
		column:   "current_file_size_1g10g_files",
		get:      func(d *usageDelta) int64 { return int64(d.currentFileSize1g10gFiles) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentFiles1GBTo10GB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentFiles1GBTo10GB) },
	},
	{
		column:   "lifetime_file_size_1g10g_files",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeFileSize1g10gFiles) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeFiles1GBTo10GB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeFiles1GBTo10GB) },
	},
	{
		column:   "current_file_size_10g100g_files",
		get:      func(d *usageDelta) int64 { return int64(d.currentFileSize10g100gFiles) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentFiles10GBTo100GB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentFiles10GBTo100GB) },
	},
	{
		column:   "lifetime_file_size_10g100g_files",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeFileSize10g100gFiles) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeFiles10GBTo100GB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeFiles10GBTo100GB) },
	},
	{
		column:   "current_file_size_gt100g_files",
		get:      func(d *usageDelta) int64 { return int64(d.currentFileSizeGt100gFiles) },
		set:      func(s *common.UsageStats, v int64) { s.CurrentFilesGreaterThan100GB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.CurrentFilesGreaterThan100GB) },
	},
	{
		column:   "lifetime_file_size_gt100g_files",
		get:      func(d *usageDelta) int64 { return int64(d.lifetimeFileSizeGt100gFiles) },
		set:      func(s *common.UsageStats, v int64) { s.LifetimeFilesGreaterThan100GB = int(v) },
		getStats: func(s *common.UsageStats) int64 { return int64(s.LifetimeFilesGreaterThan100GB) },
	},
}

// buildIncrementUpdates builds the "SET col = col + ?" map for one signed
// counter delta. Only non-zero counters are emitted, so each mutation writes the
// smallest possible UPDATE (shrinking SQL, WAL, and lock footprint); a zero
// delta on a column is a no-op and is skipped. updated_at is always set;
// last_upload_at is set only when the delta carries one.
func buildIncrementUpdates(delta *usageDelta, now time.Time) map[string]any {
	updates := make(map[string]any, len(usageCounters)+2)
	for _, c := range usageCounters {
		if v := c.get(delta); v != 0 {
			updates[c.column] = incrementColumn("usage_stats", c.column, v)
		}
	}
	updates["updated_at"] = now
	if delta.lastUploadAt != nil {
		updates["last_upload_at"] = delta.lastUploadAt
	}
	return updates
}

// applyDeltaToUsageStats writes every counter's value from a delta onto a
// UsageStats row. It is used to construct new rows (lazy create and migration
// backfill) from the same registry that drives the increment path, so the two
// can never disagree on the column set.
func applyDeltaToUsageStats(s *common.UsageStats, delta *usageDelta) {
	for _, c := range usageCounters {
		c.set(s, c.get(delta))
	}
}

// serverUsageSumSelect builds the aggregate SELECT list that sums every counter
// column, one `COALESCE(SUM(col),0) AS col` per registry entry. It is generated
// from the registry so the server sum-on-read (GetServerStatistics) can never
// omit or misname a counter. The alias matches the DB column so GORM scans each
// summed value back onto the right common.UsageStats field; COALESCE keeps an
// empty scope at 0 instead of NULL. The MIN(started_at) anchor is appended by the
// caller.
func serverUsageSumSelect() string {
	exprs := make([]string, 0, len(usageCounters))
	for _, c := range usageCounters {
		exprs = append(exprs, "COALESCE(SUM("+c.column+"),0) AS "+c.column)
	}
	return strings.Join(exprs, ", ")
}

// buildTombstoneFoldUpdates builds the `col = col + <src.col>` increment map that
// folds a deleted user's usage row into the tombstone. It is the read-from-a-row
// mirror of buildIncrementUpdates: only non-zero counters are emitted, and the
// column set is registry-generated so the fold can never disagree with the sum.
func buildTombstoneFoldUpdates(src *common.UsageStats, now time.Time) map[string]any {
	updates := make(map[string]any, len(usageCounters)+1)
	for _, c := range usageCounters {
		if v := c.getStats(src); v != 0 {
			updates[c.column] = incrementColumn("usage_stats", c.column, v)
		}
	}
	updates["updated_at"] = now
	return updates
}

// copyUsageCounters copies every registry counter's absolute value from src onto
// dst. It seeds a freshly created tombstone row (the unmigrated edge where the
// tombstone does not yet exist) from the folded source row, using the same
// registry that drives buildTombstoneFoldUpdates so the create and update paths
// can never diverge on the column set.
func copyUsageCounters(dst, src *common.UsageStats) {
	for _, c := range usageCounters {
		c.set(dst, c.getStats(src))
	}
}
