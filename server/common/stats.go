package common

import (
	"encoding/json"
	"time"
)

const DownloadStatsEntityUpload = "upload"
const DownloadStatsEntityFile = "file"
const AnonymousUserUsageStatsID = "__anonymous__"

// DeletedUserUsageStatsID is the reserved usage_stats scope key that receives a
// deleted user's counters on DeleteUser (the "tombstone" row). Under sum-on-read
// server totals — Σ over every token=” row — deleting a user's row would shrink
// server lifetime totals; folding the row into this tombstone before deleting it
// preserves those append-only totals. It joins "__anonymous__" as a reserved
// sentinel user id.
const DeletedUserUsageStatsID = "__deleted__"

// ServerStats is the admin API snapshot returned by GET /stats.
//
// The legacy flat fields (users/uploads/anonymousUploads/files/totalSize/
// anonymousTotalSize) are kept verbatim for compatibility. Everything else moved
// into the canonical nested Usage/AnonymousUsage objects. LifetimeUsers is the
// server-only lifetime user counter (append-only since Usage.StartedAt).
type ServerStats struct {
	// Current retained usage (legacy flat compatibility fields).
	Users            int   `json:"users"`
	Uploads          int   `json:"uploads"`
	AnonymousUploads int   `json:"anonymousUploads"`
	Files            int   `json:"files"`
	TotalSize        int64 `json:"totalSize"`
	AnonymousSize    int64 `json:"anonymousTotalSize"`

	// LifetimeUsers counts user creations since stats tracking started and is
	// never decremented on user deletion.
	LifetimeUsers int `json:"lifetimeUsers"`

	// Usage is the canonical scoped stats payload for the server scope, with
	// download windows populated. AnonymousUsage is the anonymous-scope payload
	// (downloads total+bytes only, no windows).
	Usage          *UsageStatsResponse `json:"usage,omitempty"`
	AnonymousUsage *UsageStatsResponse `json:"anonymousUsage,omitempty"`
}

// UserStats is the authenticated user API snapshot returned by GET /me/stats
// and embedded as the per-user stats attachment in admin user lists. The legacy
// current trio (uploads/files/totalSize) is kept for compatibility; everything
// else lives in the canonical nested Usage object.
type UserStats struct {
	Uploads   int   `json:"uploads"`
	Files     int   `json:"files"`
	TotalSize int64 `json:"totalSize"`

	// Usage is the canonical scoped stats payload. Download windows are populated
	// only for the user scope of GET /me/stats (no ?token= filter).
	Usage *UsageStatsResponse `json:"usage,omitempty"`
}

// CleaningStats cleaning statistics
type CleaningStats struct {
	RemovedUploads            int
	DeletedFiles              int
	DeletedUploads            int
	OrphanFilesCleaned        int
	OrphanTokensCleaned       int
	DownloadStatsDailyCleaned int
	UploadStatsDailyCleaned   int
}

// UsageStatsResponse is THE canonical API shape for one usage_stats row, reused
// for server, anonymous, user, and token scopes. lastUploadAt is exposed here
// and nowhere else (it is the single canonical location for the token scope too).
type UsageStatsResponse struct {
	StartedAt    *time.Time         `json:"startedAt,omitempty"`
	LastUploadAt *time.Time         `json:"lastUploadAt,omitempty"`
	Downloads    UsageDownloadStats `json:"downloads"`
	Uploads      UsageUploadStats   `json:"uploads"`
	Current      UsageStatsPeriod   `json:"current"`
	Lifetime     UsageStatsPeriod   `json:"lifetime"`
}

// UsageDownloadStats groups download counters for one scope. Total and Bytes are
// always present (all-time events and all-time bytes served). The bounded windows
// are pointers that are set only for scopes backed by a daily rollup series
// (server usage, user usage) and omitted everywhere else (token, anonymous,
// user-list attachments).
type UsageDownloadStats struct {
	Total      int64  `json:"total"`
	Bytes      int64  `json:"bytes"`
	Today      *int64 `json:"today,omitempty"`
	Last7Days  *int64 `json:"last7Days,omitempty"`
	Last30Days *int64 `json:"last30Days,omitempty"`
}

// UsageUploadStats is the symmetric upload counterpart of UsageDownloadStats.
// Total is the lifetime upload count (usage_stats.lifetime_uploads) and Bytes is
// the lifetime wire bytes received (usage_stats.uploaded_bytes, tracked since the
// upgrade that introduced it; backfilled rows start at 0). The bounded windows
// count uploads (not bytes) and, like the download windows, are set only for the
// server and user scopes that have a daily rollup series.
type UsageUploadStats struct {
	Total      int64  `json:"total"`
	Bytes      int64  `json:"bytes"`
	Today      *int64 `json:"today,omitempty"`
	Last7Days  *int64 `json:"last7Days,omitempty"`
	Last30Days *int64 `json:"last30Days,omitempty"`
}

// UsageStatsPeriod groups counters that exist both for current retained
// metadata and lifetime metadata retained since StartedAt.
type UsageStatsPeriod struct {
	Uploads   int   `json:"uploads"`
	Files     int   `json:"files"`
	TotalSize int64 `json:"totalSize"`

	Features  UsageFeatureStats  `json:"features"`
	TTL       UsageTTLStats      `json:"ttl"`
	FileSizes UsageFileSizeStats `json:"fileSizes"`
}

// UsageFeatureStats counts uploads using each optional feature.
type UsageFeatureStats struct {
	PasswordUploads  int `json:"passwordUploads"`
	RemovableUploads int `json:"removableUploads"`
	OneShotUploads   int `json:"oneShotUploads"`
	StreamUploads    int `json:"streamUploads"`
	ExtendTTLUploads int `json:"extendTTLUploads"`
	E2EEUploads      int `json:"e2eeUploads"`
	CommentUploads   int `json:"commentUploads"`
}

// UsageTTLStats counts uploads by their creation-time TTL bucket.
type UsageTTLStats struct {
	NoneUploads              int `json:"noneUploads"`
	LessThan1HourUploads     int `json:"lessThan1HourUploads"`
	OneHourToOneDayUploads   int `json:"oneHourToOneDayUploads"`
	OneDayToSevenDaysUploads int `json:"oneDayToSevenDaysUploads"`
	SevenDaysTo30DaysUploads int `json:"sevenDaysTo30DaysUploads"`
	GreaterThan30DaysUploads int `json:"greaterThan30DaysUploads"`
}

// UsageFileSizeStats counts files by file-size bucket, not bytes.
type UsageFileSizeStats struct {
	LessThan1MBFiles      int `json:"lessThan1MBFiles"`
	OneMBTo10MBFiles      int `json:"oneMBTo10MBFiles"`
	TenMBTo100MBFiles     int `json:"tenMBTo100MBFiles"`
	HundredMBTo1GBFiles   int `json:"hundredMBTo1GBFiles"`
	OneGBTo10GBFiles      int `json:"oneGBTo10GBFiles"`
	TenGBTo100GBFiles     int `json:"tenGBTo100GBFiles"`
	GreaterThan100GBFiles int `json:"greaterThan100GBFiles"`
}

// UsageStats is the single materialized stats row used for user, anonymous,
// token, and deleted-user (tombstone) scopes. There is no server row: server
// totals are a sum-on-read over every token=” row. The scope key uses reserved
// sentinels: (user_id="__anonymous__", token="") is anonymous usage,
// (user_id="__deleted__", token="") is the deleted-user tombstone,
// (user_id="<id>", token="") is a user row, and (user_id="<owner>",
// token="<token>") is token usage.
type UsageStats struct {
	UserID string `json:"userID" gorm:"primaryKey;size:256;not null;default:''"`
	Token  string `json:"token" gorm:"primaryKey;size:256;not null;default:'';index:idx_usage_stats_token"`

	// Current counters are decremented when retained uploads/files are removed.
	CurrentUploads int   `json:"currentUploads" gorm:"column:current_uploads"`
	CurrentFiles   int   `json:"currentFiles" gorm:"column:current_files"`
	CurrentSize    int64 `json:"currentSize" gorm:"column:current_size"`

	// Lifetime counters are append-only from StartedAt forward.
	LifetimeUploads int   `json:"lifetimeUploads" gorm:"column:lifetime_uploads"`
	LifetimeFiles   int   `json:"lifetimeFiles" gorm:"column:lifetime_files"`
	LifetimeSize    int64 `json:"lifetimeSize" gorm:"column:lifetime_size"`

	// Downloads counts logical download events, not downloaded bytes.
	Downloads int64 `json:"downloads" gorm:"column:downloads"`

	// DownloadedBytes counts bytes actually served (egress) on download paths.
	// Unlike Downloads, it accumulates on every response that streams file bytes,
	// including mid-range GETs that are not download events, and reflects the
	// bytes actually written (partial on client disconnect). It is only tracked
	// from the upgrade that introduced it forward; backfilled rows start at 0.
	DownloadedBytes int64 `json:"downloadedBytes" gorm:"column:downloaded_bytes"`

	// UploadedBytes counts wire bytes actually received from clients (ingress) on
	// the AddFile path. Symmetric with DownloadedBytes: it accumulates the file
	// bytes read off the wire (multipart framing excluded) on both successful
	// completions and partial/failed transfers, and is only tracked from the
	// upgrade that introduced it forward; backfilled/imported rows start at 0.
	UploadedBytes int64 `json:"uploadedBytes" gorm:"column:uploaded_bytes"`

	// LifetimeUsers is a server-only lifetime counter: it counts user creations
	// since StartedAt and is never decremented on user deletion. Other scopes
	// (user/anonymous/token rows) leave this at its zero value.
	LifetimeUsers int `json:"lifetimeUsers" gorm:"column:lifetime_users"`

	// Current and lifetime usage of upload features selected at upload creation.
	CurrentPasswordUploads   int `json:"currentPasswordUploads" gorm:"column:current_password_uploads"`
	LifetimePasswordUploads  int `json:"lifetimePasswordUploads" gorm:"column:lifetime_password_uploads"`
	CurrentRemovableUploads  int `json:"currentRemovableUploads" gorm:"column:current_removable_uploads"`
	LifetimeRemovableUploads int `json:"lifetimeRemovableUploads" gorm:"column:lifetime_removable_uploads"`
	CurrentOneShotUploads    int `json:"currentOneShotUploads" gorm:"column:current_one_shot_uploads"`
	LifetimeOneShotUploads   int `json:"lifetimeOneShotUploads" gorm:"column:lifetime_one_shot_uploads"`
	CurrentStreamUploads     int `json:"currentStreamUploads" gorm:"column:current_stream_uploads"`
	LifetimeStreamUploads    int `json:"lifetimeStreamUploads" gorm:"column:lifetime_stream_uploads"`
	CurrentExtendTTLUploads  int `json:"currentExtendTTLUploads" gorm:"column:current_extend_ttl_uploads"`
	LifetimeExtendTTLUploads int `json:"lifetimeExtendTTLUploads" gorm:"column:lifetime_extend_ttl_uploads"`
	CurrentE2EEUploads       int `json:"currentE2EEUploads" gorm:"column:current_e2ee_uploads"`
	LifetimeE2EEUploads      int `json:"lifetimeE2EEUploads" gorm:"column:lifetime_e2ee_uploads"`
	CurrentCommentUploads    int `json:"currentCommentUploads" gorm:"column:current_comment_uploads"`
	LifetimeCommentUploads   int `json:"lifetimeCommentUploads" gorm:"column:lifetime_comment_uploads"`

	// TTL bucket fields use human-readable Go names and explicit DB column tags.
	CurrentUploadTTLNone               int `json:"currentUploadTTLNone" gorm:"column:current_ttl_none_uploads"`
	LifetimeUploadTTLNone              int `json:"lifetimeUploadTTLNone" gorm:"column:lifetime_ttl_none_uploads"`
	CurrentUploadTTLLessThan1Hour      int `json:"currentUploadTTLLessThan1Hour" gorm:"column:current_ttl_lt1h_uploads"`
	LifetimeUploadTTLLessThan1Hour     int `json:"lifetimeUploadTTLLessThan1Hour" gorm:"column:lifetime_ttl_lt1h_uploads"`
	CurrentUploadTTL1HourTo1Day        int `json:"currentUploadTTL1HourTo1Day" gorm:"column:current_ttl_1h1d_uploads"`
	LifetimeUploadTTL1HourTo1Day       int `json:"lifetimeUploadTTL1HourTo1Day" gorm:"column:lifetime_ttl_1h1d_uploads"`
	CurrentUploadTTL1DayTo7Days        int `json:"currentUploadTTL1DayTo7Days" gorm:"column:current_ttl_1d7d_uploads"`
	LifetimeUploadTTL1DayTo7Days       int `json:"lifetimeUploadTTL1DayTo7Days" gorm:"column:lifetime_ttl_1d7d_uploads"`
	CurrentUploadTTL7DaysTo30Days      int `json:"currentUploadTTL7DaysTo30Days" gorm:"column:current_ttl_7d30d_uploads"`
	LifetimeUploadTTL7DaysTo30Days     int `json:"lifetimeUploadTTL7DaysTo30Days" gorm:"column:lifetime_ttl_7d30d_uploads"`
	CurrentUploadTTLGreaterThan30Days  int `json:"currentUploadTTLGreaterThan30Days" gorm:"column:current_ttl_gt30d_uploads"`
	LifetimeUploadTTLGreaterThan30Days int `json:"lifetimeUploadTTLGreaterThan30Days" gorm:"column:lifetime_ttl_gt30d_uploads"`

	// File-size bucket counters use binary MiB/GiB thresholds. Current counters
	// track retained uploaded files; lifetime counters keep completed metadata history.
	CurrentFilesLessThan1MB       int `json:"currentFilesLessThan1MB" gorm:"column:current_file_size_lt1m_files"`
	LifetimeFilesLessThan1MB      int `json:"lifetimeFilesLessThan1MB" gorm:"column:lifetime_file_size_lt1m_files"`
	CurrentFiles1MBTo10MB         int `json:"currentFiles1MBTo10MB" gorm:"column:current_file_size_1m10m_files"`
	LifetimeFiles1MBTo10MB        int `json:"lifetimeFiles1MBTo10MB" gorm:"column:lifetime_file_size_1m10m_files"`
	CurrentFiles10MBTo100MB       int `json:"currentFiles10MBTo100MB" gorm:"column:current_file_size_10m100m_files"`
	LifetimeFiles10MBTo100MB      int `json:"lifetimeFiles10MBTo100MB" gorm:"column:lifetime_file_size_10m100m_files"`
	CurrentFiles100MBTo1GB        int `json:"currentFiles100MBTo1GB" gorm:"column:current_file_size_100m1g_files"`
	LifetimeFiles100MBTo1GB       int `json:"lifetimeFiles100MBTo1GB" gorm:"column:lifetime_file_size_100m1g_files"`
	CurrentFiles1GBTo10GB         int `json:"currentFiles1GBTo10GB" gorm:"column:current_file_size_1g10g_files"`
	LifetimeFiles1GBTo10GB        int `json:"lifetimeFiles1GBTo10GB" gorm:"column:lifetime_file_size_1g10g_files"`
	CurrentFiles10GBTo100GB       int `json:"currentFiles10GBTo100GB" gorm:"column:current_file_size_10g100g_files"`
	LifetimeFiles10GBTo100GB      int `json:"lifetimeFiles10GBTo100GB" gorm:"column:lifetime_file_size_10g100g_files"`
	CurrentFilesGreaterThan100GB  int `json:"currentFilesGreaterThan100GB" gorm:"column:current_file_size_gt100g_files"`
	LifetimeFilesGreaterThan100GB int `json:"lifetimeFilesGreaterThan100GB" gorm:"column:lifetime_file_size_gt100g_files"`

	LastUploadAt *time.Time `json:"lastUploadAt,omitempty"`
	StartedAt    time.Time  `json:"startedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (UsageStats) TableName() string {
	return "usage_stats"
}

// DownloadStatsDaily stores compact daily download rollups by entity.
//
// The primary key is (Day, EntityType, EntityID). Day is normalized to UTC
// midnight. EntityType is "upload" or "file". These rows power bounded
// trending windows without recomputing them from raw download events.
type DownloadStatsDaily struct {
	Day        time.Time `json:"day" gorm:"primaryKey;index:idx_download_stats_daily_user_day,priority:2"`
	EntityType string    `json:"entityType" gorm:"primaryKey;size:16"`
	EntityID   string    `json:"entityID" gorm:"primaryKey;size:256"`
	Downloads  int64     `json:"downloads"`
	// Bytes is the egress (bytes served) rolled up for this day/entity. It is
	// incremented alongside Downloads for download events and on its own for
	// bytes-only recordings (mid-range GETs). Tracked from upgrade forward only.
	Bytes int64 `json:"bytes"`

	// UserID and Token attribute this row to the upload that produced it,
	// copied verbatim from upload.User / upload.Token at record time. Unlike
	// usage_stats, there is no anonymous sentinel here: an anonymous upload's
	// User is already "" and is stored as-is. Attribution is written once on
	// INSERT and is immutable on conflict (repeated downloads on the same day
	// only increment Downloads/Bytes), so it survives upload deletion — rollup
	// rows are retained for 31 days regardless of whether the upload still
	// exists — and powers per-user daily series (GetUserActivityStatsDaily)
	// even for deleted uploads.
	UserID string `json:"userID" gorm:"column:user_id;size:256;index:idx_download_stats_daily_user_day,priority:1"`
	Token  string `json:"token" gorm:"column:token;size:256"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func (DownloadStatsDaily) TableName() string {
	return "download_stats_daily"
}

// UploadStatsDaily stores compact daily upload rollups per attribution pair.
//
// The primary key is (Day, UserID, Token). Day is normalized to UTC midnight.
// Unlike download rollups there is no per-entity dimension: there is exactly one
// row per UTC day per (user_id, token) attribution pair. Uploads counts upload
// creations for that day; Bytes is the wire bytes received (ingress) for that
// day. These rows power the bounded upload trending windows and the daily chart.
type UploadStatsDaily struct {
	Day time.Time `json:"day" gorm:"primaryKey;index:idx_upload_stats_daily_user_day,priority:2"`

	// UserID and Token attribute this row to the upload(s) that produced it,
	// copied verbatim from upload.User / upload.Token at record time (an anonymous
	// upload's User is already "" and is stored as-is; there is no sentinel).
	// Because attribution is part of the primary key it is immutable for the life
	// of the bucket, and the user-scoped daily series (GetUserActivityStatsDaily)
	// reads it directly, so it survives upload deletion. The composite index below
	// serves both the per-user series (user_id = ?) and the server-wide series
	// (group by day across all rows).
	UserID string `json:"userID" gorm:"primaryKey;column:user_id;size:256;index:idx_upload_stats_daily_user_day,priority:1"`
	Token  string `json:"token" gorm:"primaryKey;column:token;size:256"`

	// Uploads counts upload creations for the day. Bytes is the wire bytes received
	// (ingress) for the day; it is tracked from upgrade forward only.
	Uploads int64 `json:"uploads"`
	Bytes   int64 `json:"bytes"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func (UploadStatsDaily) TableName() string {
	return "upload_stats_daily"
}

// ActivityDailyPoint is one day of the merged activity series: the summed
// downloads / bytes served and uploads / bytes received for one UTC day.
// Returned by GetUserActivityStatsDaily and GetServerActivityStatsDaily as a
// dense, oldest-first series — one point per requested day, zero-filled when a
// day has no recorded activity — so chart consumers never have to reconcile
// gaps. One query set feeds all four measures, so the windows derived from it
// (applyActivityWindows) can never drift from the chart.
type ActivityDailyPoint struct {
	Day             time.Time `json:"day"`
	Downloads       int64     `json:"downloads"`
	DownloadedBytes int64     `json:"downloadedBytes"`
	Uploads         int64     `json:"uploads"`
	UploadedBytes   int64     `json:"uploadedBytes"`
}

// MarshalJSON renders the daily series point with Day as a "YYYY-MM-DD" UTC
// calendar-day string, the stable public contract of the activity-series
// endpoints. Day is already normalized to UTC midnight by the query producing it.
func (p ActivityDailyPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Day             string `json:"day"`
		Downloads       int64  `json:"downloads"`
		DownloadedBytes int64  `json:"downloadedBytes"`
		Uploads         int64  `json:"uploads"`
		UploadedBytes   int64  `json:"uploadedBytes"`
	}{
		Day:             p.Day.UTC().Format("2006-01-02"),
		Downloads:       p.Downloads,
		DownloadedBytes: p.DownloadedBytes,
		Uploads:         p.Uploads,
		UploadedBytes:   p.UploadedBytes,
	})
}

// TrendingItem is returned by admin (and, for uploads, self-scoped) trending
// endpoints for current retained uploads/files only. DownloadCount is either
// lifetime ("all") or the selected daily-rollup window, depending on the
// endpoint query; DownloadedBytes mirrors it (bytes served instead of a
// count) and is populated on trending UPLOADS regardless of which sort was
// requested. Trending FILES leave DownloadedBytes at its zero value: there is
// no lifetime per-file byte column, so file-grain byte trending is out of
// scope.
type TrendingItem struct {
	ID               string     `json:"id"`
	Type             string     `json:"type"`
	UploadID         string     `json:"uploadID,omitempty"`
	Name             string     `json:"name,omitempty"`
	Comments         string     `json:"comments,omitempty"`
	User             string     `json:"user,omitempty"`
	Size             int64      `json:"size,omitempty"`
	Files            int        `json:"files,omitempty"`
	DownloadCount    int64      `json:"downloadCount"`
	DownloadedBytes  int64      `json:"downloadedBytes"`
	LastDownloadedAt *time.Time `json:"lastDownloadedAt,omitempty"`
}
