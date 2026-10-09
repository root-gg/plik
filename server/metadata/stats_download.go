package metadata

import (
	"fmt"
	"sort"
	"time"

	"github.com/root-gg/plik/server/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// This file contains download counter writes and daily rollup persistence. The
// hot upload/file rows hold lifetime counters; download_stats_daily holds the
// bounded windows used by trending endpoints.
//
// Canonical stats-write lock order
// --------------------------------
// Every transaction that writes download stats acquires row locks in one fixed
// order so concurrent writers can never form an AB-BA cycle and deadlock:
//
//	1. uploads              - the parent upload row.
//	2. files                - file rows, in ascending file ID order when several.
//	3. download_stats_daily - daily rollups: the upload bucket first, then file
//	                          buckets in ascending file ID order.
//	4. usage_stats          - usage rows via incrementDownloadUsage, in the order
//	                          user/anonymous, then token. There is no server row:
//	                          server totals are summed on read over token='' rows.
//
// Because a download always locks its parent upload row first, all downloads of
// the same upload serialize on that row instead of racing on the shared file,
// rollup, and usage rows beneath it. RecordFileDownload and RecordArchiveDownload
// (and any future multi-row stats writer added here) MUST follow this order. It
// is also documented in server/ARCHITECTURE.md, section "Stats Architecture".
//
// This order is now universal: every fused metadata+counter path takes the
// upload-row lock first via Backend.lockUploadRow before it touches file rows or
// usage counters — the download recorders here, the file status transitions
// (UpdateFile / UpdateFileStatus), and the removal/purge paths (RemoveUpload,
// RemoveUserUploads, DeleteRemovedUploads). Because every writer is upload-first,
// no two of them can acquire an upload row and a file row in opposite orders, so
// AB-BA deadlocks are impossible; decrement deltas are read under the upload lock
// and therefore cannot double-count or leak.
//
// Download recording is best-effort: callers log a failure and let the download
// succeed anyway. Its transactions run single-shot and simply give up on any
// error. Every fused metadata+counter mutation (upload create/remove, file
// status transitions, ...) also runs as a plain single-shot transaction: there
// is no retry apparatus. The canonical lock order above is what makes that safe
// — with no shared hot server row left and every writer taking the upload row
// first, conflicts are ordered lock waits, not deadlocks or serialization
// aborts, so there is nothing transient to retry.
//
// Bytes served (egress) vs download events
// ----------------------------------------
// Two distinct quantities are recorded here. A download *event* is a logical
// intent count and follows the counting policy in server/ARCHITECTURE.md (full
// GET and byte-0 range for direct downloads; any GET for the one-shot/stream
// branch; upload + per-file for archives). *Bytes served* accumulate on every
// response that streams file bytes — including mid-range GETs, which serve bytes
// but are NOT events — and reflect the bytes actually written, so a client that
// disconnects mid-stream records exactly its egress. A bytes-only recording
// (downloads == 0, bytes > 0) now locks and updates the upload row too (its
// downloaded_bytes column accumulates every response that streams file bytes,
// event or not, mirroring usage_stats.downloaded_bytes), but still does not
// touch the hot file row — there is no per-file byte column — nor the upload's
// own download_count/last_downloaded_at, which stay event-gated. That is the
// upload row plus a strict suffix of the remaining canonical order
// (download_stats_daily, then usage_stats), so it cannot invert against any
// other writer and stays deadlock-free.

// recordDailyDownloads atomically increments one daily rollup bucket's download
// and/or byte counters. The day must already be normalized with statsDay.
// Only the non-zero counters are incremented, so a bytes-only recording bumps
// bytes without touching downloads and vice versa.
//
// userID and token attribute the bucket to the upload that produced it,
// stored verbatim (an anonymous upload's "" User is not translated to any
// sentinel). They are only set when this call creates the row: the ON
// CONFLICT clause below deliberately omits user_id/token from its update
// list, so attribution is written once, on first insert, and is immutable for
// the lifetime of the bucket.
func (b *Backend) recordDailyDownloads(tx *gorm.DB, day time.Time, entityType string, entityID string, userID string, token string, downloads int64, bytes int64) error {
	if downloads <= 0 && bytes <= 0 {
		return nil
	}

	// Daily rows power bounded trending windows. The unique
	// (day, entity_type, entity_id) key lets all instances increment the same
	// bucket atomically without creating duplicate rows.
	now := time.Now()
	stat := &common.DownloadStatsDaily{
		Day:        day,
		EntityType: entityType,
		EntityID:   entityID,
		UserID:     userID,
		Token:      token,
		Downloads:  downloads,
		Bytes:      bytes,
		UpdatedAt:  now,
	}

	updates := map[string]any{"updated_at": now}
	if downloads > 0 {
		updates["downloads"] = incrementColumn("download_stats_daily", "downloads", downloads)
	}
	if bytes > 0 {
		updates["bytes"] = incrementColumn("download_stats_daily", "bytes", bytes)
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "day"},
			{Name: "entity_type"},
			{Name: "entity_id"},
		},
		DoUpdates: clause.Assignments(updates),
	}).Create(stat).Error
}

// CreateDownloadStatsDaily persists one daily download rollup during import or
// tests. Normal download paths use recordDailyDownloads so repeated events
// increment existing buckets.
func (b *Backend) CreateDownloadStatsDaily(stats *common.DownloadStatsDaily) error {
	return b.db.Create(stats).Error
}

// ForEachDownloadStatsDaily executes f for every daily download stats row.
// Export uses this to stream rollups without loading the whole table.
func (b *Backend) ForEachDownloadStatsDaily(f func(stats *common.DownloadStatsDaily) error) error {
	rows, err := b.db.Model(&common.DownloadStatsDaily{}).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		stats := &common.DownloadStatsDaily{}
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

// statsDay normalizes timestamps to the UTC day key shared by
// download_stats_daily and upload_stats_daily (the merged activity series).
// Using UTC keeps rollup windows stable across instances running in different
// local time zones.
func statsDay(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

// The dense per-day download series and its bounded-window summation used to
// live here (GetUserDownloadStatsDaily / GetServerDownloadStatsDaily /
// getDownloadStatsDailySeries / sumDownloadStatsDailyPoints). They were folded
// into the merged activity series in stats_upload.go
// (getActivityStatsDailySeries + applyActivityWindows), which feeds all four
// download/upload measures from one query set so the windows and the chart can
// never drift apart.

// DeleteExpiredDownloadStatsDaily prunes old daily rollups. The UI/API only
// need 30-day windows, so cleanup keeps today's UTC bucket plus the previous
// 30 UTC buckets and deletes anything older.
func (b *Backend) DeleteExpiredDownloadStatsDaily() (int, error) {
	cutoff := statsDay(time.Now()).AddDate(0, 0, -30)
	result := b.db.
		Where("day < ?", cutoff).
		Delete(&common.DownloadStatsDaily{})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

// RecordFileDownload atomically records the egress (bytes served) of one file
// download and, when countEvent is true, the download event on the file and its
// parent upload. countEvent carries the pre-stream download-counting policy
// decision (see server/ARCHITECTURE.md): full GETs and byte-0 ranges count an
// event, mid-range GETs do not. bytes is the number of body bytes actually
// written to the client (0 for HEAD, tiny for range-error bodies, partial on
// disconnect). When there is neither an event nor any bytes there is nothing to
// record.
func (b *Backend) RecordFileDownload(upload *common.Upload, file *common.File, bytes int64, countEvent bool) error {
	if upload == nil || file == nil {
		return fmt.Errorf("missing upload or file")
	}
	if !countEvent && bytes <= 0 {
		// Nothing to record: no event and no egress (e.g. HEAD).
		return nil
	}

	downloads := int64(0)
	if countEvent {
		downloads = 1
	}

	now := time.Now()
	day := statsDay(now)
	// Single-shot best-effort transaction: on any error the counters are simply
	// skipped, the caller logs and the download still succeeds.
	return b.db.Transaction(func(tx *gorm.DB) error {
		// The file row's download_count/last_downloaded_at move only for a counted
		// event; a bytes-only recording (mid-range GET) skips the hot file row —
		// there is no per-file byte column.
		recordFileRows := func(tx *gorm.DB) error {
			return tx.Model(&common.File{}).
				Where(&common.File{ID: file.ID}).
				Updates(map[string]any{"download_count": gorm.Expr("download_count + ?", 1), "last_downloaded_at": now}).Error
		}
		// A single file download attributes its egress to its one file rollup too,
		// mirroring the upload rollup: both get (downloads, bytes), so a bytes-only
		// recording still bumps the file rollup's bytes with downloads 0.
		recordFileRollups := func(tx *gorm.DB) error {
			return b.recordDailyDownloads(tx, day, common.DownloadStatsEntityFile, file.ID, upload.User, upload.Token, downloads, bytes)
		}
		return b.recordUploadDownload(tx, upload, downloads, bytes, now, day, recordFileRows, recordFileRollups)
	})
}

// RecordArchiveDownload atomically records an archive download's egress and,
// when countEvent is true, one download event on the upload and on every
// included file. bytes is the total egress of the zip stream actually written
// to the client; it is attributed to the upload rollup and the usage rows (the
// archive is a single stream, so it cannot be split across the per-file
// rollups — those only ever get the event, bytes 0). countEvent mirrors
// RecordFileDownload's event-vs-bytes split: the caller (get_archive.go) calls
// this with countEvent=false on a mid-stream failure, so the bytes already
// streamed to the client before the failure are still recorded as a
// bytes-only event (downloads=0) instead of being lost — matching the
// "a client that disconnects mid-stream records exactly its egress" policy
// documented above and in server/ARCHITECTURE.md. When countEvent is false,
// only the upload row's downloaded_bytes, the upload's daily rollup bytes, and
// the usage rows' bytes are touched — the file rows, the per-file rollups, and
// the upload's download_count/last_downloaded_at all stay untouched, exactly
// like RecordFileDownload's bytes-only path.
func (b *Backend) RecordArchiveDownload(upload *common.Upload, files []*common.File, bytes int64, countEvent bool) error {
	if upload == nil {
		return fmt.Errorf("missing upload")
	}
	if len(files) == 0 {
		return nil
	}
	if !countEvent && bytes <= 0 {
		// Nothing to record: no event and no egress.
		return nil
	}

	downloads := int64(0)
	if countEvent {
		downloads = 1
	}

	now := time.Now()
	day := statsDay(now)
	fileIDs := make([]string, 0, len(files))
	for _, file := range files {
		if file != nil {
			fileIDs = append(fileIDs, file.ID)
		}
	}
	if len(fileIDs) == 0 {
		return nil
	}
	// Sort so the per-file rollup loop below locks file rollup rows in a fixed
	// ascending order, matching the canonical lock order documented at the top
	// of this file. This only orders that sequential loop: the bulk file
	// UPDATE just below is one statement (WHERE id IN fileIDs), and its
	// internal row-lock acquisition order is decided by the database, not by
	// this slice's order.
	sort.Strings(fileIDs)

	// Single-shot best-effort transaction: on any error the counters are simply
	// skipped, the caller logs and the download still succeeds.
	return b.db.Transaction(func(tx *gorm.DB) error {
		// Archive downloads count as one upload download and one logical download
		// for each included file, matching the download view's aggregate display.
		// A bytes-only recording (countEvent=false) skips the hot file rows.
		recordFileRows := func(tx *gorm.DB) error {
			return tx.Model(&common.File{}).
				Where("id IN ?", fileIDs).
				Updates(map[string]any{"download_count": gorm.Expr("download_count + ?", 1), "last_downloaded_at": now}).Error
		}
		// The whole-archive egress is attributed to the upload rollup (in the
		// shared tail); the per-file rollups get the event only (bytes 0) because a
		// single zip stream cannot be split across its files. A bytes-only
		// recording skips them entirely: there is no event to attribute.
		recordFileRollups := func(tx *gorm.DB) error {
			if downloads <= 0 {
				return nil
			}
			for _, fileID := range fileIDs {
				if err := b.recordDailyDownloads(tx, day, common.DownloadStatsEntityFile, fileID, upload.User, upload.Token, 1, 0); err != nil {
					return err
				}
			}
			return nil
		}
		return b.recordUploadDownload(tx, upload, downloads, bytes, now, day, recordFileRows, recordFileRollups)
	})
}

// recordUploadDownload writes the parent upload row's download counters and the
// upload-scoped daily rollup and usage increments shared by both download
// recorders. Steps run in the canonical lock order documented at the top of this
// file: the upload row first, then (for a counted event) the file rows via
// recordFileRows, then the upload daily rollup, then the file rollups via
// recordFileRollups, then the usage rows. downloads is 1 for a counted event and
// 0 for a bytes-only recording; bytes is added to the upload's downloaded_bytes
// on every call, while download_count/last_downloaded_at and the file rows move
// only for an event. recordFileRows and recordFileRollups carry the per-recorder
// file writes (a single file vs every archived file).
func (b *Backend) recordUploadDownload(tx *gorm.DB, upload *common.Upload, downloads int64, bytes int64, now time.Time, day time.Time, recordFileRows func(tx *gorm.DB) error, recordFileRollups func(tx *gorm.DB) error) error {
	// The upload row's downloaded_bytes accumulates every response that streams
	// file bytes, event or not — mirroring usage_stats.downloaded_bytes — so it is
	// updated unconditionally (the callers guarantee an event or bytes > 0).
	// download_count/last_downloaded_at, and the file rows, move only for an event.
	uploadUpdates := map[string]any{"downloaded_bytes": gorm.Expr("downloaded_bytes + ?", bytes)}
	if downloads > 0 {
		uploadUpdates["download_count"] = gorm.Expr("download_count + ?", 1)
		uploadUpdates["last_downloaded_at"] = now
	}
	result := tx.Model(&common.Upload{}).
		Where(&common.Upload{ID: upload.ID}).
		Updates(uploadUpdates)
	if result.Error != nil {
		return result.Error
	}
	// This UPDATE is soft-delete-scoped, so it matches no row when the upload was
	// deleted mid-download. Recording is best-effort and runs post-stream from a
	// pre-stream upload object, so give up here rather than record for an upload
	// that no longer exists: continuing would write orphan daily rollups and,
	// worse, resurrect a deleted user's (user,"") usage row via the createMissing
	// insert in the usage increment below. Mirrors incrementUsageForCompletedFile /
	// decrementUsageForUploadedFile, which likewise bail when the upload is gone.
	if result.RowsAffected == 0 {
		return nil
	}

	if downloads > 0 {
		if err := recordFileRows(tx); err != nil {
			return err
		}
	}

	if err := b.recordDailyDownloads(tx, day, common.DownloadStatsEntityUpload, upload.ID, upload.User, upload.Token, downloads, bytes); err != nil {
		return err
	}

	if err := recordFileRollups(tx); err != nil {
		return err
	}

	return b.incrementDownloadUsage(tx, upload, downloads, bytes)
}

// incrementDownloadUsage applies a download event count and byte egress to the
// user/anonymous and token usage rows in the canonical lock order. downloads may
// be 0 for a bytes-only recording (mid-range GET) while bytes is still
// incremented. There is no server row; server download/byte totals are summed on
// read over the token=” rows.
func (b *Backend) incrementDownloadUsage(tx *gorm.DB, upload *common.Upload, downloads int64, bytes int64) error {
	delta := usageDelta{downloads: downloads, downloadedBytes: bytes}
	return b.applyUsageDelta(tx, upload.User, upload.Token, delta)
}

// FixtureSeedDownloadedBytes adds bytes-only usage (no download event, no
// rollup write) to the user/anonymous and token usage_stats rows owning
// upload (there is no server row; server bytes are summed on read). It exists
// for fixture/backfill code paths that already seed daily
// rollups directly (bypassing RecordFileDownload/RecordArchiveDownload — e.g.
// `plikd fakedb`, see server/cmd/fakedb.go) and need the usage_stats lifetime
// byte totals to stay consistent with those rollups, without also replaying a
// download event that was already accounted for elsewhere. It must not be
// used on the live request path: real downloads always go through
// RecordFileDownload/RecordArchiveDownload so rollups and usage_stats update
// atomically in one transaction.
func (b *Backend) FixtureSeedDownloadedBytes(upload *common.Upload, bytes int64) error {
	if upload == nil || bytes <= 0 {
		return nil
	}
	return b.db.Transaction(func(tx *gorm.DB) error {
		return b.incrementDownloadUsage(tx, upload, 0, bytes)
	})
}
