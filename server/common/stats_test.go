package common

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// These are shape-lock tests: they marshal fully-populated stats structs and
// assert the EXACT set of JSON keys at every level of the canonical nested
// contract. They fail if a field is added, removed, or renamed — the whole point
// is that the public stats shape (the spec for the webapp/clients) cannot drift
// silently. api.md must be updated in lockstep with any intentional change here.

// jsonKeys marshals v and returns the sorted top-level object keys.
func jsonKeys(t *testing.T, v any) []string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &obj))
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// keysAt returns the sorted object keys of the nested field path inside v.
func keysAt(t *testing.T, v any, path ...string) []string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	cur := raw
	for _, p := range path {
		var obj map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(cur, &obj), "path %v", path)
		next, ok := obj[p]
		require.Truef(t, ok, "missing key %q on path %v", p, path)
		cur = next
	}
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(cur, &obj))
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedStrings(s ...string) []string {
	sort.Strings(s)
	return s
}

// fullPeriod returns a UsageStatsPeriod with every counter set non-zero.
func fullPeriod() UsageStatsPeriod {
	return UsageStatsPeriod{
		Uploads: 1, Files: 2, TotalSize: 3,
		Features:  UsageFeatureStats{1, 2, 3, 4, 5, 6, 7},
		TTL:       UsageTTLStats{1, 2, 3, 4, 5, 6},
		FileSizes: UsageFileSizeStats{1, 2, 3, 4, 5, 6, 7},
	}
}

// fullUsageResponse returns a usage response with windows populated (server/user
// scope) so every possible key is present.
func fullUsageResponse() *UsageStatsResponse {
	now := time.Now()
	today := int64(1)
	last7 := int64(2)
	last30 := int64(3)
	return &UsageStatsResponse{
		StartedAt:    &now,
		LastUploadAt: &now,
		Downloads:    UsageDownloadStats{Total: 10, Bytes: 20, Today: &today, Last7Days: &last7, Last30Days: &last30},
		Uploads:      UsageUploadStats{Total: 30, Bytes: 40, Today: &today, Last7Days: &last7, Last30Days: &last30},
		Current:      fullPeriod(),
		Lifetime:     fullPeriod(),
	}
}

func assertPeriodShape(t *testing.T, v any, period string) {
	t.Helper()
	require.Equal(t, sortedStrings("uploads", "files", "totalSize", "features", "ttl", "fileSizes"),
		keysAt(t, v, "usage", period), "%s period keys", period)
	require.Equal(t, sortedStrings("passwordUploads", "removableUploads", "oneShotUploads", "streamUploads", "extendTTLUploads", "e2eeUploads", "commentUploads"),
		keysAt(t, v, "usage", period, "features"), "%s features keys", period)
	require.Equal(t, sortedStrings("noneUploads", "lessThan1HourUploads", "oneHourToOneDayUploads", "oneDayToSevenDaysUploads", "sevenDaysTo30DaysUploads", "greaterThan30DaysUploads"),
		keysAt(t, v, "usage", period, "ttl"), "%s ttl keys", period)
	require.Equal(t, sortedStrings("lessThan1MBFiles", "oneMBTo10MBFiles", "tenMBTo100MBFiles", "hundredMBTo1GBFiles", "oneGBTo10GBFiles", "tenGBTo100GBFiles", "greaterThan100GBFiles"),
		keysAt(t, v, "usage", period, "fileSizes"), "%s fileSizes keys", period)
}

func TestServerStatsJSONShape(t *testing.T) {
	stats := &ServerStats{
		Users: 1, Uploads: 2, AnonymousUploads: 3, Files: 4, TotalSize: 5, AnonymousSize: 6,
		LifetimeUsers: 7,
		Usage:         fullUsageResponse(),
		AnonymousUsage: &UsageStatsResponse{
			StartedAt: nil, // omitted
			Downloads: UsageDownloadStats{Total: 1, Bytes: 2},
			Uploads:   UsageUploadStats{Total: 3, Bytes: 4},
			Current:   fullPeriod(),
			Lifetime:  fullPeriod(),
		},
	}
	require.Equal(t,
		sortedStrings("users", "uploads", "anonymousUploads", "files", "totalSize", "anonymousTotalSize", "lifetimeUsers", "usage", "anonymousUsage"),
		jsonKeys(t, stats), "ServerStats top-level keys")

	// usage carries windows; anonymousUsage does not (both downloads and uploads).
	require.Equal(t, sortedStrings("total", "bytes", "today", "last7Days", "last30Days"),
		keysAt(t, stats, "usage", "downloads"), "server usage downloads keys (windows present)")
	require.Equal(t, sortedStrings("total", "bytes", "today", "last7Days", "last30Days"),
		keysAt(t, stats, "usage", "uploads"), "server usage uploads keys (windows present)")
	require.Equal(t, sortedStrings("total", "bytes"),
		keysAt(t, stats, "anonymousUsage", "downloads"), "anonymous usage downloads keys (windows omitted)")
	require.Equal(t, sortedStrings("total", "bytes"),
		keysAt(t, stats, "anonymousUsage", "uploads"), "anonymous usage uploads keys (windows omitted)")

	require.Equal(t, sortedStrings("startedAt", "lastUploadAt", "downloads", "uploads", "current", "lifetime"),
		keysAt(t, stats, "usage"), "usage object keys")
	assertPeriodShape(t, stats, "current")
	assertPeriodShape(t, stats, "lifetime")
}

func TestUserStatsJSONShape(t *testing.T) {
	stats := &UserStats{Uploads: 1, Files: 2, TotalSize: 3, Usage: fullUsageResponse()}
	require.Equal(t, sortedStrings("uploads", "files", "totalSize", "usage"),
		jsonKeys(t, stats), "UserStats top-level keys")
	require.Equal(t, sortedStrings("startedAt", "lastUploadAt", "downloads", "uploads", "current", "lifetime"),
		keysAt(t, stats, "usage"), "usage object keys")
	require.Equal(t, sortedStrings("total", "bytes", "today", "last7Days", "last30Days"),
		keysAt(t, stats, "usage", "uploads"), "user usage uploads keys (windows present)")
	assertPeriodShape(t, stats, "current")
	assertPeriodShape(t, stats, "lifetime")
}

func TestTokenStatsJSONShape(t *testing.T) {
	// token.stats wraps the usage object in the same usage{} envelope as the
	// server and user scopes (token.stats.usage.current / .lifetime); token
	// scope omits download windows.
	now := time.Now()
	stats := &TokenStats{
		Usage: &UsageStatsResponse{
			StartedAt:    &now,
			LastUploadAt: &now,
			Downloads:    UsageDownloadStats{Total: 1, Bytes: 2},
			Uploads:      UsageUploadStats{Total: 3, Bytes: 4},
			Current:      fullPeriod(),
			Lifetime:     fullPeriod(),
		},
	}
	require.Equal(t, sortedStrings("usage"), jsonKeys(t, stats), "TokenStats top-level keys")
	require.Equal(t, sortedStrings("startedAt", "lastUploadAt", "downloads", "uploads", "current", "lifetime"),
		keysAt(t, stats, "usage"), "token stats usage keys")
	require.Equal(t, sortedStrings("total", "bytes"),
		keysAt(t, stats, "usage", "downloads"), "token downloads keys (windows omitted)")
	require.Equal(t, sortedStrings("total", "bytes"),
		keysAt(t, stats, "usage", "uploads"), "token uploads keys (windows omitted)")
	assertPeriodShape(t, stats, "current")
	assertPeriodShape(t, stats, "lifetime")
}

func TestUsageResponseOmitsNilTimestamps(t *testing.T) {
	// startedAt/lastUploadAt are omitempty: absent when nil.
	stats := &UsageStatsResponse{Downloads: UsageDownloadStats{}, Uploads: UsageUploadStats{}, Current: fullPeriod(), Lifetime: fullPeriod()}
	require.Equal(t, sortedStrings("downloads", "uploads", "current", "lifetime"), jsonKeys(t, stats))
}

func TestTrendingItemJSONShape(t *testing.T) {
	now := time.Now()
	item := &TrendingItem{
		ID: "upload-id", Type: DownloadStatsEntityUpload, UploadID: "u1", Name: "file.txt",
		Comments: "a comment", User: "local:alice", Size: 42, Files: 3,
		DownloadCount: 7, DownloadedBytes: 1024, LastDownloadedAt: &now,
	}
	require.Equal(t,
		sortedStrings("id", "type", "uploadID", "name", "comments", "user", "size", "files", "downloadCount", "downloadedBytes", "lastDownloadedAt"),
		jsonKeys(t, item), "TrendingItem top-level keys")
}

func TestActivityDailyPointJSONShape(t *testing.T) {
	p := ActivityDailyPoint{
		Day:             time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Downloads:       3,
		DownloadedBytes: 1024,
		Uploads:         2,
		UploadedBytes:   2048,
	}
	require.Equal(t, sortedStrings("day", "downloads", "downloadedBytes", "uploads", "uploadedBytes"), jsonKeys(t, p))

	raw, err := json.Marshal(p)
	require.NoError(t, err)
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &obj))
	require.JSONEq(t, `"2026-05-04"`, string(obj["day"]), "day must be a YYYY-MM-DD date string")
}
