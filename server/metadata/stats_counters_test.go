package metadata

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/root-gg/plik/server/common"
)

// gormColumnName extracts the DB column name from a gorm struct tag, e.g.
// "column:current_uploads" -> "current_uploads". It returns "" when the tag
// declares no explicit column (keys and timestamps in common.UsageStats).
func gormColumnName(tag string) string {
	for part := range strings.SplitSeq(tag, ";") {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(part, "column:"); ok {
			return after
		}
	}
	return ""
}

// TestUsageCountersRegistryMatchesSchema is the drift tripwire: it reflects over
// common.UsageStats and asserts the set of int/int64 counter columns (fields
// carrying a gorm:"column:..." tag) is exactly the set of columns declared in
// the usageCounters registry. If a counter is added to the struct without a
// registry entry (or vice versa), this test fails — the whole point of the
// registry is that the two can never silently diverge. Reflection is confined to
// this test; production paths iterate the registry directly.
//
// This only checks that the column *names* match set-for-set; it says nothing
// about whether each counterSpec's get/set closures are wired to the right
// usageDelta/common.UsageStats field. That correctness is covered elsewhere, by
// the agreement/exactness tests (e.g. TestBackend_ImportBackfillLifetimeCounterAgreement
// and the various "...IsExact" tests in stats_test.go) that exercise real
// increment/backfill paths end-to-end.
func TestUsageCountersRegistryMatchesSchema(t *testing.T) {
	// Columns declared by the registry (also assert none is declared twice).
	registry := make(map[string]bool, len(usageCounters))
	for _, c := range usageCounters {
		require.False(t, registry[c.column], "duplicate column %q in usageCounters registry", c.column)
		registry[c.column] = true
	}

	// Counter columns declared by the schema: every int/int64 field on
	// common.UsageStats carrying an explicit gorm column tag. The type filter
	// excludes the string keys (user_id/token) and the time.Time bookkeeping
	// fields (started_at/created_at/updated_at/last_upload_at), none of which are
	// signed counters.
	schema := make(map[string]bool)
	typ := reflect.TypeFor[common.UsageStats]()
	for field := range typ.Fields() {
		if field.Type.Kind() != reflect.Int && field.Type.Kind() != reflect.Int64 {
			continue
		}
		column := gormColumnName(field.Tag.Get("gorm"))
		if column == "" {
			continue
		}
		schema[column] = true
	}

	require.NotEmpty(t, schema, "reflection found no counter columns on common.UsageStats")

	for column := range schema {
		require.Truef(t, registry[column],
			"schema counter column %q has no counterSpec entry; add one to usageCounters in stats_counters.go", column)
	}
	for column := range registry {
		require.Truef(t, schema[column],
			"registry column %q has no matching int/int64 counter field on common.UsageStats", column)
	}
	require.Equal(t, len(schema), len(registry), "registry and schema counter column sets differ in size")
}

// TestBuildIncrementUpdatesEmitsOnlyNonZeroCounters pins the non-zero-only SET
// map contract: buildIncrementUpdates emits an assignment for a counter only
// when its delta is non-zero, always emits updated_at, and emits last_upload_at
// only when the delta carries one.
func TestBuildIncrementUpdatesEmitsOnlyNonZeroCounters(t *testing.T) {
	now := time.Now()

	// A single non-zero counter -> that column plus updated_at, nothing else.
	single := buildIncrementUpdates(&usageDelta{lifetimeUploads: 1}, now)
	require.ElementsMatch(t, []string{"lifetime_uploads", "updated_at"}, mapKeys(single),
		"a single-counter delta must produce a single counter assignment plus updated_at")

	// last_upload_at is bookkeeping that rides along only when set.
	uploadAt := now
	withUploadAt := buildIncrementUpdates(&usageDelta{lifetimeUploads: 1, lastUploadAt: &uploadAt}, now)
	require.ElementsMatch(t, []string{"lifetime_uploads", "updated_at", "last_upload_at"}, mapKeys(withUploadAt))

	// An all-zero delta writes no counters at all, only the updated_at bookkeeping.
	zero := buildIncrementUpdates(&usageDelta{}, now)
	require.ElementsMatch(t, []string{"updated_at"}, mapKeys(zero),
		"a zero delta must not emit any counter assignment")

	// A mixed delta emits exactly its non-zero counters (int and int64 fields).
	mixed := buildIncrementUpdates(&usageDelta{currentUploads: 1, currentSize: 42, downloads: 3}, now)
	require.ElementsMatch(t,
		[]string{"current_uploads", "current_size", "downloads", "updated_at"},
		mapKeys(mixed))
}

// TestIncrementUpdatesDryRunSingleAssignment proves the non-zero SET map end to
// end: rendering the UPDATE for a single-counter delta produces exactly one
// "col = col + <placeholder>" increment (plus the always-present updated_at
// bookkeeping, which is a plain assignment, not an increment). incrementColumn
// renders each increment through the raw expression "usage_stats.<col> + ?", so
// the unquoted "usage_stats." table prefix appears exactly once per incremented
// counter on every dialect (SQLite/MySQL backtick-quote and PostgreSQL uses $N
// placeholders, but neither rewrites the raw expression prefix). Counting it is
// therefore dialect-agnostic, which matters because this suite also runs against
// PostgreSQL and MySQL.
func TestIncrementUpdatesDryRunSingleAssignment(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	now := time.Now()
	updates := buildIncrementUpdates(&usageDelta{lifetimeUploads: 1}, now)

	stmt := b.db.Session(&gorm.Session{DryRun: true}).
		Model(&common.UsageStats{}).
		Where("user_id = ? AND token = ?", "", "").
		Updates(updates).Statement
	sql := stmt.SQL.String()

	require.True(t, strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "UPDATE"),
		"expected an UPDATE statement, got: %s", sql)
	require.Equal(t, 1, strings.Count(sql, "usage_stats."),
		"exactly one counter column must be incremented, got SQL: %s", sql)
	require.Contains(t, sql, "lifetime_uploads", "the incremented column must be the one carried by the delta")
}

// TestServerSumAndFoldCoverExactlyRegistry is the drift tripwire for the two
// registry-generated server-scope generators introduced with sum-on-read: the
// server sum-select (serverUsageSumSelect) and the DeleteUser tombstone fold
// (buildTombstoneFoldUpdates). Each must cover exactly the registry's counter
// columns — no more, no less — so a counter the sum includes can never be one the
// fold silently drops (which would leak server lifetime totals on user deletion),
// and vice versa. It also asserts every counterSpec wires a non-nil getStats (the
// closure both generators depend on) and that set/getStats address the same field.
func TestServerSumAndFoldCoverExactlyRegistry(t *testing.T) {
	// getStats wired + set/getStats round-trip, per counter, on a fresh row so
	// columns cannot contaminate each other.
	for i, c := range usageCounters {
		require.NotNilf(t, c.getStats, "counterSpec %d (%s) is missing a getStats closure", i, c.column)
		s := &common.UsageStats{}
		c.set(s, int64(i+1))
		require.Equalf(t, int64(i+1), c.getStats(s), "set/getStats disagree for column %s", c.column)
	}

	// Sum-select: exactly one COALESCE(SUM(col),0) AS col per registry column.
	sel := serverUsageSumSelect()
	for _, c := range usageCounters {
		require.Containsf(t, sel, "COALESCE(SUM("+c.column+"),0) AS "+c.column, "sum-select is missing column %s", c.column)
	}
	require.Equal(t, len(usageCounters), strings.Count(sel, "COALESCE(SUM("),
		"sum-select must emit exactly one SUM per registry column")

	// Fold: a source row with every counter non-zero emits exactly the registry
	// columns plus updated_at (mirrors buildIncrementUpdates' contract).
	src := &common.UsageStats{}
	for i, c := range usageCounters {
		c.set(src, int64(i+1))
	}
	updates := buildTombstoneFoldUpdates(src, time.Now())
	want := make([]string, 0, len(usageCounters)+1)
	for _, c := range usageCounters {
		want = append(want, c.column)
	}
	want = append(want, "updated_at")
	require.ElementsMatch(t, want, mapKeys(updates), "fold must update exactly the registry columns plus updated_at")

	// copyUsageCounters (tombstone create path) carries every counter across.
	dst := &common.UsageStats{}
	copyUsageCounters(dst, src)
	for _, c := range usageCounters {
		require.Equalf(t, c.getStats(src), c.getStats(dst), "copyUsageCounters dropped column %s", c.column)
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
