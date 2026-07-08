package metadata

import (
	"database/sql"
	"strings"
	"time"

	"github.com/root-gg/plik/server/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// This file contains counter mutation helpers for the unified usage_stats table.
// All counters are updated as signed SQL deltas inside the same transaction as
// the metadata change that caused them.

// The usageDelta struct and the counter registry that derives every write-side
// column list from it (the increment SET map, the lazy-row-creation literal,
// the delta merge helper, and the migration backfill row constructor) live in
// stats_counters.go, the single source of truth for the counter columns.

// incrementColumn builds a table-qualified "column = column + delta" expression.
// The table prefix avoids ambiguous column names in updates that use joins or
// backend-specific generated SQL.
func incrementColumn(table string, column string, delta any) clause.Expr {
	return gorm.Expr(table+"."+column+" + ?", delta)
}

// activityWindowDays is the daily-rollup horizon used to derive the bounded
// download AND upload windows (today / last 7 / last 30 days) surfaced on
// scoped usage from the merged activity series.
const activityWindowDays = 30

// usageStatsPeriod maps one period (current or lifetime) of a DB stats row using
// the two accessors so the current and lifetime nested shapes are built from the
// exact same field layout.
func usageStatsPeriod(uploads int, files int, size int64, features common.UsageFeatureStats, ttl common.UsageTTLStats, sizes common.UsageFileSizeStats) common.UsageStatsPeriod {
	return common.UsageStatsPeriod{
		Uploads:   uploads,
		Files:     files,
		TotalSize: size,
		Features:  features,
		TTL:       ttl,
		FileSizes: sizes,
	}
}

// usageStatsResponseFromUsage maps one DB stats row to the canonical nested API
// shape reused by every scope (server, anonymous, user, token). Download/upload
// windows are NOT set here — callers that have a daily series apply them
// separately via applyActivityWindows so token/anonymous/user-list scopes omit
// them.
func usageStatsResponseFromUsage(usage *common.UsageStats) *common.UsageStatsResponse {
	if usage == nil {
		return &common.UsageStatsResponse{}
	}
	return &common.UsageStatsResponse{
		StartedAt:    &usage.StartedAt,
		LastUploadAt: usage.LastUploadAt,
		Downloads: common.UsageDownloadStats{
			Total: usage.Downloads,
			Bytes: usage.DownloadedBytes,
		},
		Uploads: common.UsageUploadStats{
			Total: int64(usage.LifetimeUploads),
			Bytes: usage.UploadedBytes,
		},
		Current: usageStatsPeriod(
			usage.CurrentUploads, usage.CurrentFiles, usage.CurrentSize,
			common.UsageFeatureStats{
				PasswordUploads:  usage.CurrentPasswordUploads,
				RemovableUploads: usage.CurrentRemovableUploads,
				OneShotUploads:   usage.CurrentOneShotUploads,
				StreamUploads:    usage.CurrentStreamUploads,
				ExtendTTLUploads: usage.CurrentExtendTTLUploads,
				E2EEUploads:      usage.CurrentE2EEUploads,
				CommentUploads:   usage.CurrentCommentUploads,
			},
			common.UsageTTLStats{
				NoneUploads:              usage.CurrentUploadTTLNone,
				LessThan1HourUploads:     usage.CurrentUploadTTLLessThan1Hour,
				OneHourToOneDayUploads:   usage.CurrentUploadTTL1HourTo1Day,
				OneDayToSevenDaysUploads: usage.CurrentUploadTTL1DayTo7Days,
				SevenDaysTo30DaysUploads: usage.CurrentUploadTTL7DaysTo30Days,
				GreaterThan30DaysUploads: usage.CurrentUploadTTLGreaterThan30Days,
			},
			common.UsageFileSizeStats{
				LessThan1MBFiles:      usage.CurrentFilesLessThan1MB,
				OneMBTo10MBFiles:      usage.CurrentFiles1MBTo10MB,
				TenMBTo100MBFiles:     usage.CurrentFiles10MBTo100MB,
				HundredMBTo1GBFiles:   usage.CurrentFiles100MBTo1GB,
				OneGBTo10GBFiles:      usage.CurrentFiles1GBTo10GB,
				TenGBTo100GBFiles:     usage.CurrentFiles10GBTo100GB,
				GreaterThan100GBFiles: usage.CurrentFilesGreaterThan100GB,
			},
		),
		Lifetime: usageStatsPeriod(
			usage.LifetimeUploads, usage.LifetimeFiles, usage.LifetimeSize,
			common.UsageFeatureStats{
				PasswordUploads:  usage.LifetimePasswordUploads,
				RemovableUploads: usage.LifetimeRemovableUploads,
				OneShotUploads:   usage.LifetimeOneShotUploads,
				StreamUploads:    usage.LifetimeStreamUploads,
				ExtendTTLUploads: usage.LifetimeExtendTTLUploads,
				E2EEUploads:      usage.LifetimeE2EEUploads,
				CommentUploads:   usage.LifetimeCommentUploads,
			},
			common.UsageTTLStats{
				NoneUploads:              usage.LifetimeUploadTTLNone,
				LessThan1HourUploads:     usage.LifetimeUploadTTLLessThan1Hour,
				OneHourToOneDayUploads:   usage.LifetimeUploadTTL1HourTo1Day,
				OneDayToSevenDaysUploads: usage.LifetimeUploadTTL1DayTo7Days,
				SevenDaysTo30DaysUploads: usage.LifetimeUploadTTL7DaysTo30Days,
				GreaterThan30DaysUploads: usage.LifetimeUploadTTLGreaterThan30Days,
			},
			common.UsageFileSizeStats{
				LessThan1MBFiles:      usage.LifetimeFilesLessThan1MB,
				OneMBTo10MBFiles:      usage.LifetimeFiles1MBTo10MB,
				TenMBTo100MBFiles:     usage.LifetimeFiles10MBTo100MB,
				HundredMBTo1GBFiles:   usage.LifetimeFiles100MBTo1GB,
				OneGBTo10GBFiles:      usage.LifetimeFiles1GBTo10GB,
				TenGBTo100GBFiles:     usage.LifetimeFiles10GBTo100GB,
				GreaterThan100GBFiles: usage.LifetimeFilesGreaterThan100GB,
			},
		),
	}
}

// The bounded-window helper that used to live here (applyDownloadWindows) was
// generalized to applyActivityWindows in stats_upload.go, which sets both the
// download and the upload count windows from one merged activity series.

// serverStatsFromUsage maps the DB server row to the admin API shape. The
// conversion keeps DB-oriented column names out of handler/UI code.
func serverStatsFromUsage(usage *common.UsageStats, anonymous *common.UsageStats, users int64) *common.ServerStats {
	if usage == nil {
		usage = &common.UsageStats{}
	}
	if anonymous == nil {
		anonymous = &common.UsageStats{}
	}
	return &common.ServerStats{
		Users:            int(users),
		Uploads:          usage.CurrentUploads,
		AnonymousUploads: anonymous.CurrentUploads,
		Files:            usage.CurrentFiles,
		TotalSize:        usage.CurrentSize,
		AnonymousSize:    anonymous.CurrentSize,
		LifetimeUsers:    usage.LifetimeUsers,
		Usage:            usageStatsResponseFromUsage(usage),
		AnonymousUsage:   usageStatsResponseFromUsage(anonymous),
	}
}

// userStatsFromUsage maps a user DB counter row to the /me/stats API shape.
func userStatsFromUsage(usage *common.UsageStats) *common.UserStats {
	if usage == nil {
		return nil
	}
	return &common.UserStats{
		Uploads:   usage.CurrentUploads,
		Files:     usage.CurrentFiles,
		TotalSize: usage.CurrentSize,
		Usage:     usageStatsResponseFromUsage(usage),
	}
}

func (b *Backend) incrementUserUsage(tx *gorm.DB, userID string, delta usageDelta) error {
	if userID == "" {
		return nil
	}
	return b.incrementUsage(tx, userID, "", delta, true)
}

func (b *Backend) incrementTokenUsage(tx *gorm.DB, token string, userID string, delta usageDelta) error {
	if token == "" {
		return nil
	}
	// Token rows are owned by token create/delete. Once a token is revoked,
	// older uploads may still be downloaded or removed, but those mutations must
	// not recreate retained stats for the revoked token.
	return b.incrementUsage(tx, userID, token, delta, false)
}

// applyUsageDelta applies one delta to every usage_stats row an upload's
// activity touches: the owning user (or the anonymous sentinel) first, then
// its token, if any. This is the canonical "user/anonymous before token" write
// pair (see the lock order documented at the top of stats_download.go),
// collapsed into one call so every stats writer gets it structurally, rather
// than by repeating both calls and userUsageStatsID(user) at each site.
func (b *Backend) applyUsageDelta(tx *gorm.DB, user string, token string, delta usageDelta) error {
	err := b.incrementUserUsage(tx, userUsageStatsID(user), delta)
	if err != nil {
		return err
	}
	return b.incrementTokenUsage(tx, token, user, delta)
}

// incrementUsage applies one signed counter delta to a scoped usage_stats row.
// The composite key is (user_id, token); token="" identifies a user, anonymous,
// or deleted-user (tombstone) row, and a non-empty token identifies a token row.
// There is no server row: server totals are summed on read over token=” rows.
func (b *Backend) incrementUsage(tx *gorm.DB, userID string, token string, delta usageDelta, createMissing bool) error {
	now := time.Now()
	// The SET map is derived from the counter registry (stats_counters.go) and
	// carries only the non-zero counters plus the bookkeeping columns.
	updates := buildIncrementUpdates(&delta, now)

	result := tx.Model(&common.UsageStats{}).Where("user_id = ? AND token = ?", userID, token).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	if !createMissing {
		return nil
	}
	return b.createUsageStatsWithDelta(tx, userID, token, delta, updates, now)
}

func (b *Backend) createUsageStatsWithDelta(tx *gorm.DB, userID string, token string, delta usageDelta, updates map[string]any, now time.Time) error {
	// The counter columns are populated from the same registry that drives the
	// increment SET map; only the bookkeeping fields are set explicitly here.
	stats := &common.UsageStats{
		UserID:       userID,
		Token:        token,
		LastUploadAt: delta.lastUploadAt,
		StartedAt:    now,
	}
	applyDeltaToUsageStats(stats, &delta)
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "token"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(stats).Error
}

// ensureBaseUsageStatsRows seeds the always-present token=” rows during
// initialization/migration without overwriting existing counters. It runs on
// every startup, so it is the creation point for fresh installs, where
// gormigrate's InitSchema records migration 0011 as applied without running its
// body (so 0011's tombstone creation never executes on a brand-new database).
//
// There is no longer a server row: server totals are summed on read over the
// token=” rows. As a safety net, any stray ("","") row left behind by an older
// build is deleted so it can never double-count into that sum.
func (b *Backend) ensureBaseUsageStatsRows(tx *gorm.DB) error {
	now := time.Now()

	err := tx.Where("user_id = ? AND token = ?", "", "").Delete(&common.UsageStats{}).Error
	if err != nil {
		return err
	}

	// Anonymous usage row: read directly by GetServerStatistics for the
	// AnonymousUploads/Size fields and included in the token='' server sum.
	err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&common.UsageStats{
		UserID:    common.AnonymousUserUsageStatsID,
		StartedAt: now,
	}).Error
	if err != nil {
		return err
	}

	// Deleted-user tombstone: receives DeleteUser folds so server lifetime totals
	// survive account deletion, and anchors the server startedAt MIN on fresh
	// installs. Its started_at never moves later, so DoNothing on conflict keeps
	// the earliest (migration-created) value on migrated instances.
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&common.UsageStats{
		UserID:    common.DeletedUserUsageStatsID,
		StartedAt: now,
	}).Error
}

// foldUsageIntoDeletedTombstone adds every registry counter of a source usage row
// into the ("__deleted__","") tombstone, so a deleted user's contribution to the
// server sum-on-read (Σ over token=” rows) survives dropping the user's own row.
// The update column set is registry-generated (buildTombstoneFoldUpdates), the
// same source of truth as the server sum, so a fold can never miss a counter a
// sum would include.
//
// started_at is pulled back to MIN(tombstone, src): the server "Stats since"
// anchor is the MIN(started_at) over the token=” rows, and dropping the folded
// row would otherwise let that MIN jump FORWARD when the deleted (or imported)
// user was the oldest scope — e.g. deleting the first-ever user, or importing a
// user/tombstone with a historical CreatedAt. The tombstone is NOT necessarily
// the earliest token=” row (an imported user can predate it), so the min is
// computed Go-side from the two loaded values and written back, rather than with
// a non-portable LEAST()/MIN() (a raw MIN also loses datetime affinity on SQLite;
// see GetServerStatistics). The tombstone exists on every migrated/initialized
// instance; the create branch is the unmigrated edge (test coverage) and seeds
// started_at from the folded row, upserting so a concurrent creator's counters
// are still added rather than lost.
func (b *Backend) foldUsageIntoDeletedTombstone(tx *gorm.DB, src *common.UsageStats) error {
	now := time.Now()
	updates := buildTombstoneFoldUpdates(src, now)

	// Read the existing tombstone model-aware so started_at scans as a datetime
	// (a raw MIN aggregate would not, on SQLite), then take the Go-side min.
	tombstone := &common.UsageStats{}
	err := tx.Where("user_id = ? AND token = ?", common.DeletedUserUsageStatsID, "").Take(tombstone).Error
	if err == gorm.ErrRecordNotFound {
		created := &common.UsageStats{
			UserID:    common.DeletedUserUsageStatsID,
			StartedAt: src.StartedAt,
		}
		if created.StartedAt.IsZero() {
			created.StartedAt = now
		}
		copyUsageCounters(created, src)
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "token"}},
			DoUpdates: clause.Assignments(updates),
		}).Create(created).Error
	}
	if err != nil {
		return err
	}

	if !src.StartedAt.IsZero() && src.StartedAt.Before(tombstone.StartedAt) {
		updates["started_at"] = src.StartedAt
	}
	return tx.Model(&common.UsageStats{}).
		Where("user_id = ? AND token = ?", common.DeletedUserUsageStatsID, "").
		Updates(updates).Error
}

// GetDeletedUsageTombstone returns the ("__deleted__","") tombstone row for
// export, or nil when it is absent (e.g. an instance that never initialized it).
func (b *Backend) GetDeletedUsageTombstone() (*common.UsageStats, error) {
	usage := &common.UsageStats{}
	err := b.db.Where("user_id = ? AND token = ?", common.DeletedUserUsageStatsID, "").Take(usage).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return usage, nil
}

// ImportDeletedUsageTombstone folds an exported tombstone record into the live
// tombstone row. The tombstone is the only usage row not rebuildable from
// metadata (a deleted user's uploads are gone), so it is exported/imported as a
// record. Import replays it exactly once and folds nothing extra: imported users
// replay through CreateUser and get their own lifetime_users, and a deleted
// user's folded counters ride solely on this record.
//
// Imports target a fresh DB: unlike every other imported object, which
// fails loudly on a primary-key conflict when replayed into an
// already-populated destination, this fold has no such guard. Replaying the
// same tombstone-bearing export a second time into a non-fresh DB silently
// double-counts its lifetime counters — the accepted, documented consequence
// of that contract (see TestBackend_ImportDeletedTombstoneTwiceDoubleCountsLifetimeCounters
// and server/ARCHITECTURE.md), not a bug.
func (b *Backend) ImportDeletedUsageTombstone(src *common.UsageStats) error {
	if src == nil {
		return nil
	}
	return b.db.Transaction(func(tx *gorm.DB) error {
		return b.foldUsageIntoDeletedTombstone(tx, src)
	})
}

// addUploadFeatureUsage accumulates one upload's feature-flag counters onto d
// (current and/or lifetime, scaled by the current/lifetime arguments). Feature
// counters are upload-level stats, not file-level stats. A nil upload is a
// no-op so callers can pass through an upload that failed to load.
func addUploadFeatureUsage(d *usageDelta, upload *common.Upload, current int, lifetime int) {
	if upload == nil {
		return
	}

	if upload.ProtectedByPassword {
		d.currentPasswordUploads += current
		d.lifetimePasswordUploads += lifetime
	}
	if upload.Removable {
		d.currentRemovableUploads += current
		d.lifetimeRemovableUploads += lifetime
	}
	if upload.OneShot {
		d.currentOneShotUploads += current
		d.lifetimeOneShotUploads += lifetime
	}
	if upload.Stream {
		d.currentStreamUploads += current
		d.lifetimeStreamUploads += lifetime
	}
	if upload.ExtendTTL {
		d.currentExtendTTLUploads += current
		d.lifetimeExtendTTLUploads += lifetime
	}
	if upload.E2EE != "" {
		d.currentE2EEUploads += current
		d.lifetimeE2EEUploads += lifetime
	}
	if strings.TrimSpace(upload.Comments) != "" {
		d.currentCommentUploads += current
		d.lifetimeCommentUploads += lifetime
	}
}

// userUsageStatsID maps anonymous uploads to the synthetic usage row. Real
// authenticated users keep their provider-qualified user id.
func userUsageStatsID(userID string) string {
	if userID == "" {
		return common.AnonymousUserUsageStatsID
	}
	return userID
}

// addUploadTTLUsage accumulates the creation-time TTL of one upload into its
// coarse histogram bucket for current and/or lifetime server stats. A nil
// upload is a no-op so callers can pass through an upload that failed to load.
func addUploadTTLUsage(d *usageDelta, upload *common.Upload, current int, lifetime int) {
	if upload == nil {
		return
	}

	// Buckets are based on the TTL selected when the upload was created. They are
	// intentionally coarse so the admin UI stays readable.
	switch {
	case upload.TTL <= 0:
		d.currentTTLNoneUploads += current
		d.lifetimeTTLNoneUploads += lifetime
	case upload.TTL < int(time.Hour.Seconds()):
		d.currentTTLLt1hUploads += current
		d.lifetimeTTLLt1hUploads += lifetime
	case upload.TTL < int((24 * time.Hour).Seconds()):
		d.currentTTL1h1dUploads += current
		d.lifetimeTTL1h1dUploads += lifetime
	case upload.TTL < int((7 * 24 * time.Hour).Seconds()):
		d.currentTTL1d7dUploads += current
		d.lifetimeTTL1d7dUploads += lifetime
	case upload.TTL <= int((30 * 24 * time.Hour).Seconds()):
		d.currentTTL7d30dUploads += current
		d.lifetimeTTL7d30dUploads += lifetime
	default:
		d.currentTTLGt30dUploads += current
		d.lifetimeTTLGt30dUploads += lifetime
	}
}

const (
	fileSize1MB   int64 = 1024 * 1024
	fileSize10MB        = 10 * fileSize1MB
	fileSize100MB       = 100 * fileSize1MB
	fileSize1GB         = 1024 * fileSize1MB
	fileSize10GB        = 10 * fileSize1GB
	fileSize100GB       = 100 * fileSize1GB
)

// addFileSizeUsage accumulates one file into the server-wide size histogram
// bucket it belongs to. The thresholds use binary MiB/GiB values to match Go
// size handling elsewhere.
func addFileSizeUsage(d *usageDelta, size int64, current int, lifetime int) {
	switch {
	case size < fileSize1MB:
		d.currentFileSizeLt1mFiles += current
		d.lifetimeFileSizeLt1mFiles += lifetime
	case size < fileSize10MB:
		d.currentFileSize1m10mFiles += current
		d.lifetimeFileSize1m10mFiles += lifetime
	case size < fileSize100MB:
		d.currentFileSize10m100mFiles += current
		d.lifetimeFileSize10m100mFiles += lifetime
	case size < fileSize1GB:
		d.currentFileSize100m1gFiles += current
		d.lifetimeFileSize100m1gFiles += lifetime
	case size < fileSize10GB:
		d.currentFileSize1g10gFiles += current
		d.lifetimeFileSize1g10gFiles += lifetime
	case size < fileSize100GB:
		d.currentFileSize10g100gFiles += current
		d.lifetimeFileSize10g100gFiles += lifetime
	default:
		d.currentFileSizeGt100gFiles += current
		d.lifetimeFileSizeGt100gFiles += lifetime
	}
}

// uploadCreationDelta builds the full usage delta for a newly created upload:
// its feature/TTL buckets, the lifetime upload/file counters, and — for a
// still-retained upload — the matching current counters. lastUploadAt is
// passed in rather than read from upload.CreatedAt so the caller can fall back
// to time.Now() for an upload created without a timestamp, and the same value
// then anchors both the returned delta and the daily rollup the caller writes
// alongside it.
//
// Upload creation moves lifetime counters immediately. Current counters are
// applied only for retained uploads, which also covers import/backfill paths
// that can create already-deleted rows.
func uploadCreationDelta(upload *common.Upload, lastUploadAt time.Time) usageDelta {
	var delta usageDelta

	addUploadFeatureUsage(&delta, upload, 0, 1)
	addUploadTTLUsage(&delta, upload, 0, 1)
	delta.lifetimeUploads = 1
	delta.downloads = upload.DownloadCount
	// Deliberately NOT delta.downloadedBytes = upload.DownloadedBytes: unlike
	// download_count (whose lifetime total has no other rebuild source and is
	// replayed into usage_stats.downloads exactly once, here), downloaded_bytes
	// already has an independent rebuild/reseed path — fakedb seeds it via the
	// separate FixtureSeedDownloadedBytes call (server/cmd/fakedb.go), and folding the
	// column value in here too would double it into usage_stats.downloaded_bytes.
	// This also matches the documented import behavior (server/ARCHITECTURE.md
	// "Migration Backfill"): usage_stats byte counters stay at 0 on import/backfill
	// even though the upload row's own downloaded_bytes survives verbatim.
	delta.lastUploadAt = &lastUploadAt
	if !upload.DeletedAt.Valid {
		delta.currentUploads = 1
		addUploadFeatureUsage(&delta, upload, 1, 0)
		addUploadTTLUsage(&delta, upload, 1, 0)
	}
	for _, file := range upload.Files {
		switch file.Status {
		case common.FileUploaded:
			if !upload.DeletedAt.Valid {
				delta.currentFiles++
				delta.currentSize += file.Size
				addFileSizeUsage(&delta, file.Size, 1, 0)
			}
			delta.lifetimeFiles++
			delta.lifetimeSize += file.Size
			addFileSizeUsage(&delta, file.Size, 0, 1)
		case common.FileRemoved, common.FileDeleted:
			delta.lifetimeFiles++
			delta.lifetimeSize += file.Size
			addFileSizeUsage(&delta, file.Size, 0, 1)
		}
	}

	return delta
}

// incrementUsageForCompletedFile records a file that successfully entered
// Plik's lifetime history. current=false is used by stream completion paths
// where the file is immediately deleted after a successful transfer.
//
// wireBytes carries the file's wire bytes received (ingress) for the live
// AddFile completion path: on a fully consumed stream that equals file.Size, so
// callers pass file.Size. It is recorded into today's upload_stats_daily bucket
// and the usage_stats.uploaded_bytes counter, in canonical lock order (rollup
// before usage rows). The importer (CreateFile) passes wireBytes=0 so replayed
// history never fabricates ingress ("since upgrade, no backfill").
func (b *Backend) incrementUsageForCompletedFile(tx *gorm.DB, file *common.File, current bool, wireBytes int64) error {
	if file == nil {
		return nil
	}

	upload, err := b.lockUploadRow(tx, file.UploadID)
	if err != nil {
		return err
	}
	if upload == nil {
		return nil
	}

	// A successful file upload always contributes to lifetime counters. Current
	// counters only move when the parent upload is still retained.
	delta := usageDelta{lifetimeFiles: 1, lifetimeSize: file.Size}
	addFileSizeUsage(&delta, file.Size, 0, 1)
	if current && !upload.DeletedAt.Valid {
		delta.currentFiles = 1
		delta.currentSize = file.Size
		addFileSizeUsage(&delta, file.Size, 1, 0)
	}
	if wireBytes > 0 {
		delta.uploadedBytes = wireBytes
	}

	// Canonical lock order: the daily rollup (upload-first lock already held by
	// the caller) is written before the usage rows below.
	if wireBytes > 0 {
		err = b.recordDailyUploads(tx, statsDay(time.Now()), upload.User, upload.Token, 0, wireBytes)
		if err != nil {
			return err
		}
	}

	return b.applyUsageDelta(tx, upload.User, upload.Token, delta)
}

// decrementUsageForUploadedFile removes retained file usage from current
// counters only. Lifetime counters intentionally never move backwards.
func (b *Backend) decrementUsageForUploadedFile(tx *gorm.DB, file *common.File) error {
	if file == nil {
		return nil
	}

	upload, err := b.lockUploadRow(tx, file.UploadID)
	if err != nil {
		return err
	}
	if upload == nil || upload.DeletedAt.Valid {
		return nil
	}

	// Deletions only decrement current counters. Lifetime counters are append-only
	// from the point the stats migration was introduced.
	delta := usageDelta{currentFiles: -1, currentSize: -file.Size}
	addFileSizeUsage(&delta, file.Size, -1, 0)
	return b.applyUsageDelta(tx, upload.User, upload.Token, delta)
}

// lockUploadRow takes the parent upload row's write lock at the start of any
// transaction that mutates that upload's file rows or usage counters. Acquiring
// this lock first everywhere establishes the canonical "upload row before file
// rows" order documented in stats_download.go: concurrent file transitions,
// downloads, and removals of the same upload then serialize on the upload row
// instead of racing on the file/usage rows beneath it, so decrement deltas read
// under the lock stay stable and can neither double-count nor leak.
//
// The upload is loaded Unscoped so cleanup/transition paths that run after the
// upload has been soft-deleted still serialize correctly; callers that must skip
// already-removed uploads inspect DeletedAt on the returned row. A missing
// upload (or empty id) returns (nil, nil).
//
// The read is dialect-guarded by applyUpdateLock: FOR UPDATE on the backends that
// support it, a plain read on SQLite.
func (b *Backend) lockUploadRow(tx *gorm.DB, uploadID string) (upload *common.Upload, err error) {
	if uploadID == "" {
		return nil, nil
	}

	upload = &common.Upload{}
	err = b.applyUpdateLock(tx.Unscoped()).Where("id = ?", uploadID).Take(upload).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return upload, nil
}

// applyUpdateLock adds a SELECT ... FOR UPDATE row lock to stmt on every backend
// that supports it, so a locking read serializes concurrent writers on the row it
// returns. SQLite has no FOR UPDATE — it would be a SQL syntax error — and is a
// single writer under WAL, so the clause is skipped there; a plain read
// serializes writers just as well. This is the one dialect guard shared by every
// locking read (lockUploadRow, the DeleteUser usage-row fold).
func (b *Backend) applyUpdateLock(stmt *gorm.DB) *gorm.DB {
	if b.db.Dialector.Name() != "sqlite" {
		stmt = stmt.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return stmt
}

// uploadedFileStatsForUploads returns retained file count and size for a set of
// uploads. It is used by migrations/backfills where application counters do not
// yet exist.
func uploadedFileStatsForUploads(tx *gorm.DB, uploadIDs any) (files int, size int64, err error) {
	err = tx.Model(&common.File{}).
		Select("count(files.id), coalesce(sum(size),0)").
		Where("upload_id IN (?)", uploadIDs).
		Where(&common.File{Status: common.FileUploaded}).
		Row().
		Scan(&files, &size)
	return files, size, err
}

// addFileSizeStatsForUploads accumulates the file-size histogram for a
// selected upload/status scope onto d, by iterating file rows and reusing the
// single Go bucketing helper (addFileSizeUsage) so the bucket thresholds live
// in exactly one place instead of being duplicated as a SQL CASE ladder.
// current and lifetime let callers reuse the same aggregate for retained
// current files and lifetime completed metadata. It is used by
// migrations/backfills and upload removal, not on the per-request hot path,
// so per-row iteration is acceptable.
func addFileSizeStatsForUploads(tx *gorm.DB, d *usageDelta, uploadIDs any, statuses []string, current int, lifetime int) error {
	if ids, ok := uploadIDs.([]string); ok && len(ids) == 0 {
		return nil
	}

	rows, err := tx.Model(&common.File{}).
		Select("size").
		Where("upload_id IN (?)", uploadIDs).
		Where("status IN ?", statuses).
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var size int64
		err = rows.Scan(&size)
		if err != nil {
			return err
		}
		addFileSizeUsage(d, size, current, lifetime)
	}

	return nil
}

// addUploadFeatureStatsForUploads accumulates aggregate server feature and TTL
// counters from upload rows onto d. It is used by migrations/backfills where we
// need exact counts over a selected set of upload IDs instead of applying one
// upload mutation at a time.
func addUploadFeatureStatsForUploads(tx *gorm.DB, d *usageDelta, uploadIDs any, current int, lifetime int) error {
	if ids, ok := uploadIDs.([]string); ok && len(ids) == 0 {
		return nil
	}

	// The caller controls current-vs-lifetime semantics through uploadIDs.
	// Keep this query unscoped so lifetime backfills can include soft-deleted uploads.
	rows, err := tx.Unscoped().Model(&common.Upload{}).
		Select("protected_by_password, removable, one_shot, stream, extend_ttl, e2ee, comments, ttl").
		Where("id IN (?)", uploadIDs).
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var protectedByPassword, removable, oneShot, stream, extendTTL sql.NullBool
		var e2ee, comments sql.NullString
		var ttl sql.NullInt64
		err = rows.Scan(&protectedByPassword, &removable, &oneShot, &stream, &extendTTL, &e2ee, &comments, &ttl)
		if err != nil {
			return err
		}

		upload := &common.Upload{
			ProtectedByPassword: protectedByPassword.Bool,
			Removable:           removable.Bool,
			OneShot:             oneShot.Bool,
			Stream:              stream.Bool,
			ExtendTTL:           extendTTL.Bool,
			E2EE:                e2ee.String,
			Comments:            comments.String,
			TTL:                 int(ttl.Int64),
		}
		addUploadFeatureUsage(d, upload, current, lifetime)
		addUploadTTLUsage(d, upload, current, lifetime)
	}

	return nil
}
