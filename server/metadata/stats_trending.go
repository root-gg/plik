package metadata

import (
	"database/sql"
	"time"

	"github.com/root-gg/plik/server/common"
)

// This file contains admin trending queries. The "all" window reads lifetime
// counters from hot upload/file rows; bounded windows read daily rollups.

// Trending-uploads sort values. TrendingSortDownloads (the default/empty value) ranks by
// download count; TrendingSortDownloadedBytes ranks by bytes served instead.
// Both metrics are always populated on TrendingItem regardless of which one
// is selected — only the ORDER BY / zero-value omission follows sort.
const (
	TrendingSortDownloads       = "downloads"
	TrendingSortDownloadedBytes = "downloadedBytes"
)

// GetTrendingUploads returns currently available uploads ordered by the
// selected sort metric (download count or downloaded bytes). window is one of
// "", "all", "1d", "7d", "30d", sort is "" or one of the TrendingSort*
// constants, and limit is 1..100; the handler (parseTrendingWindow /
// parseTrendingSort / parseTrendingLimit, server/handlers/misc.go) is the
// single validation site, so this method trusts its inputs.
func (b *Backend) GetTrendingUploads(window string, sort string, limit int) ([]*common.TrendingItem, error) {
	return b.getTrendingUploads(nil, window, sort, limit)
}

// GetUserTrendingUploads is the user-scoped variant of GetTrendingUploads,
// restricted to uploads owned by userID (follows GetUserActivityStatsDaily's
// precedent: a public GetUserXxx(userID, ...) wrapper over a shared method
// that takes an optional *string scope). Backs GET /me/stats/trending/uploads
// — there is deliberately no user-scoped trending FILES variant, since
// file-grain byte trending is out of scope.
func (b *Backend) GetUserTrendingUploads(userID string, window string, sort string, limit int) ([]*common.TrendingItem, error) {
	return b.getTrendingUploads(&userID, window, sort, limit)
}

func (b *Backend) getTrendingUploads(userID *string, window string, sort string, limit int) ([]*common.TrendingItem, error) {
	if window == "" || window == "all" {
		return b.getTrendingUploadsAll(userID, sort, limit)
	}
	return b.getTrendingUploadsWindow(userID, trendingSince(window), sort, limit)
}

// GetTrendingFiles returns currently available files ordered by download count.
// It follows the same window semantics as GetTrendingUploads and likewise trusts
// its (handler-validated) inputs.
func (b *Backend) GetTrendingFiles(window string, limit int) ([]*common.TrendingItem, error) {
	if window == "" || window == "all" {
		return b.getTrendingFilesAll(limit)
	}
	return b.getTrendingFilesWindow(trendingSince(window), limit)
}

// trendingSince converts a public window token into the oldest UTC daily bucket
// included in the query. The current day counts as day one, so 7d includes today
// plus the previous six UTC days. The handler validates the window before this is
// reached, so an unexpected value falls back to the single-day window.
func trendingSince(window string) time.Time {
	today := statsDay(time.Now())
	switch window {
	case "7d":
		return today.AddDate(0, 0, -6)
	case "30d":
		return today.AddDate(0, 0, -29)
	default:
		return today
	}
}

// populateServerActivityWindows sets the server usage download AND upload count
// windows from the merged activity series. Download windows use only upload-entity
// download rows so direct and archive downloads both count once, matching
// usage_stats.downloads. It reuses GetServerActivityStatsDaily's dense 30-day
// series so the bounded windows and the per-day chart series can never drift apart.
func (b *Backend) populateServerActivityWindows(stats *common.ServerStats) error {
	if stats == nil || stats.Usage == nil {
		return nil
	}

	points, err := b.GetServerActivityStatsDaily(activityWindowDays)
	if err != nil {
		return err
	}

	applyActivityWindows(stats.Usage, points)
	return nil
}

// trendingUploadsAllColumn maps a sort value to the lifetime upload column it
// ranks (and requires positive) in the "all" window. Both columns are always
// selected regardless of which one is chosen — see GetTrendingUploads.
func trendingUploadsAllColumn(sort string) string {
	if sort == TrendingSortDownloadedBytes {
		return "uploads.downloaded_bytes"
	}
	return "uploads.download_count"
}

// getTrendingUploadsAll ranks retained uploads by their lifetime download
// counter or lifetime downloaded bytes (per sort). Uploads with a zero value
// for the CHOSEN metric are intentionally omitted from trending. userID
// optionally restricts results to one owner (GetUserTrendingUploads); nil
// means unscoped (GetTrendingUploads).
func (b *Backend) getTrendingUploadsAll(userID *string, sort string, limit int) (items []*common.TrendingItem, err error) {
	// The "all" window can read directly from the uploads row. Join a compact
	// per-upload file aggregate so the admin UI can show file count and
	// retained size without loading every file row.
	fileStats := b.db.Table("files").
		Select("upload_id, count(id) as files, coalesce(sum(size),0) as size").
		Where("status = ?", common.FileUploaded).
		Group("upload_id")

	column := trendingUploadsAllColumn(sort)

	stmt := b.db.Model(&common.Upload{}).
		Select("uploads.id, uploads.comments, uploads.user, uploads.download_count, uploads.downloaded_bytes, uploads.last_downloaded_at, coalesce(file_stats.files,0) as files, coalesce(file_stats.size,0) as size").
		Joins("left join (?) as file_stats on file_stats.upload_id = uploads.id", fileStats).
		Where(column + " > 0").
		Order(column + " desc, uploads.last_downloaded_at desc, uploads.id asc").
		Limit(limit)
	if userID != nil {
		stmt = stmt.Where("uploads.user = ?", *userID)
	}

	rows, err := stmt.Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanTrendingUploads(rows)
}

// scanTrendingUploads drains rows shaped like an upload-trending query result —
// id, comments, owner, lifetime download count, lifetime downloaded bytes,
// last-downloaded time, then the joined file count/size aggregate, in that
// column order — into TrendingItem values. Both trending-uploads query
// variants (lifetime "all" and rollup-windowed) select their columns in this
// order regardless of which metric they rank by, so they share this scan
// instead of repeating it.
func scanTrendingUploads(rows *sql.Rows) (items []*common.TrendingItem, err error) {
	for rows.Next() {
		item := &common.TrendingItem{Type: common.DownloadStatsEntityUpload}
		err = rows.Scan(&item.ID, &item.Comments, &item.User, &item.DownloadCount, &item.DownloadedBytes, &item.LastDownloadedAt, &item.Files, &item.Size)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// getTrendingFilesAll ranks retained files by their lifetime download counter.
// Removed/deleted files and files under soft-deleted uploads are hidden so the
// admin UI never links to unavailable content.
func (b *Backend) getTrendingFilesAll(limit int) (items []*common.TrendingItem, err error) {
	// Only currently retained files are eligible for trending. The parent upload
	// join filters out soft-deleted uploads and gives the UI the owner id.
	rows, err := b.db.Table("files").
		Select("files.id, files.upload_id, files.name, uploads.user, files.size, files.download_count, files.last_downloaded_at").
		Joins("join uploads on uploads.id = files.upload_id and uploads.deleted_at is null").
		Where("files.status = ?", common.FileUploaded).
		Where("files.download_count > 0").
		Order("files.download_count desc, files.last_downloaded_at desc, files.id asc").
		Limit(limit).
		Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanTrendingFiles(rows)
}

// scanTrendingFiles drains rows shaped like a file-trending query result — id,
// upload id, name, owner, size, download count, last-downloaded time, in that
// column order — into TrendingItem values. Both trending-files query variants
// (lifetime "all" and rollup-windowed) select their columns in this order, so
// they share this scan instead of repeating it.
func scanTrendingFiles(rows *sql.Rows) (items []*common.TrendingItem, err error) {
	for rows.Next() {
		item := &common.TrendingItem{Type: common.DownloadStatsEntityFile}
		err = rows.Scan(&item.ID, &item.UploadID, &item.Name, &item.User, &item.Size, &item.DownloadCount, &item.LastDownloadedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// trendingUploadsWindowColumn maps a sort value to the summed-rollup column
// (of the download_stats subquery aliased below) it ranks and requires
// positive in a bounded window.
func trendingUploadsWindowColumn(sort string) string {
	if sort == TrendingSortDownloadedBytes {
		return "download_stats.bytes"
	}
	return "download_stats.downloads"
}

// getTrendingUploadsWindow ranks retained uploads by summed daily rollups
// (downloads or bytes served, per sort) in a bounded window. File count and
// size are recomputed from retained file rows for display, not from
// historical rollups. userID optionally restricts results to one owner
// (GetUserTrendingUploads); nil means unscoped (GetTrendingUploads). User
// scoping filters on uploads.user via the live-uploads join below — NOT
// on the rollup's own user_id — so only currently available uploads count.
func (b *Backend) getTrendingUploadsWindow(userID *string, since time.Time, sort string, limit int) (items []*common.TrendingItem, err error) {
	// Windowed upload trending starts from daily rollups because the hot upload row
	// only stores lifetime counters. Grouping by entity_id yields one row per
	// upload for the selected 1d/7d/30d window, summing both metrics so either
	// can be selected below without a second query.
	downloadStats := b.db.Table("download_stats_daily").
		Select("entity_id, sum(downloads) as downloads, sum(bytes) as bytes").
		Where("entity_type = ?", common.DownloadStatsEntityUpload).
		Where("day >= ?", since).
		Group("entity_id")
	// File totals are kept in a second subquery so uploads with no retained files
	// still appear, but deleted/missing files are not counted in the display.
	fileStats := b.db.Table("files").
		Select("upload_id, count(id) as files, coalesce(sum(size),0) as size").
		Where("status = ?", common.FileUploaded).
		Group("upload_id")

	column := trendingUploadsWindowColumn(sort)

	stmt := b.db.Table("(?) as download_stats", downloadStats).
		Select("uploads.id, uploads.comments, uploads.user, download_stats.downloads, download_stats.bytes, uploads.last_downloaded_at, coalesce(file_stats.files,0) as files, coalesce(file_stats.size,0) as size").
		Joins("join uploads on uploads.id = download_stats.entity_id and uploads.deleted_at is null").
		Joins("left join (?) as file_stats on file_stats.upload_id = uploads.id", fileStats).
		Where(column + " > 0").
		Order(column + " desc, uploads.last_downloaded_at desc, uploads.id asc").
		Limit(limit)
	if userID != nil {
		stmt = stmt.Where("uploads.user = ?", *userID)
	}

	rows, err := stmt.Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanTrendingUploads(rows)
}

// getTrendingFilesWindow ranks retained files by summed daily file rollups in a
// bounded window.
func (b *Backend) getTrendingFilesWindow(since time.Time, limit int) (items []*common.TrendingItem, err error) {
	// Windowed file trending sums only daily file rollups in the requested range.
	// The joins deliberately require both the file and its parent upload to still
	// be available so admin links never point at expired/deleted content.
	rows, err := b.db.Table("download_stats_daily").
		Select("files.id, files.upload_id, files.name, uploads.user, files.size, sum(download_stats_daily.downloads) as downloads, files.last_downloaded_at").
		Joins("join files on files.id = download_stats_daily.entity_id and files.status = ?", common.FileUploaded).
		Joins("join uploads on uploads.id = files.upload_id and uploads.deleted_at is null").
		Where("download_stats_daily.entity_type = ?", common.DownloadStatsEntityFile).
		Where("download_stats_daily.day >= ?", since).
		Group("files.id, files.upload_id, files.name, uploads.user, files.size, files.last_downloaded_at").
		Order("downloads desc, files.last_downloaded_at desc, files.id asc").
		Limit(limit).
		Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanTrendingFiles(rows)
}
