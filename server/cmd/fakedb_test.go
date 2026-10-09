package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/root-gg/logger"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadata"
)

// newFakeDBTestBackend opens a throwaway SQLite metadata backend for
// fakedb_test.go's rollup-persistence tests, mirroring how fakedb() itself
// opens its backend (same driver, same NewBackend call).
func newFakeDBTestBackend(t *testing.T) *metadata.Backend {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "fakedb-test.db")
	backend, err := metadata.NewBackend(&metadata.Config{Driver: "sqlite3", ConnectionString: dbPath}, logger.NewLogger())
	require.NoError(t, err)
	t.Cleanup(func() { _ = backend.Shutdown() })
	return backend
}

// collectDownloadStatsDaily drains every persisted daily rollup row via the
// backend's public streaming accessor (no direct DB handle is exposed).
func collectDownloadStatsDaily(t *testing.T, backend *metadata.Backend) []*common.DownloadStatsDaily {
	t.Helper()
	var rows []*common.DownloadStatsDaily
	require.NoError(t, backend.ForEachDownloadStatsDaily(func(stats *common.DownloadStatsDaily) error {
		rows = append(rows, stats)
		return nil
	}))
	return rows
}

func TestValidateFakeDBCountsRejectsNegativeCounts(t *testing.T) {
	tests := []struct {
		name   string
		counts fakeDBCounts
		flag   string
	}{
		{"users", fakeDBCounts{users: -1}, "--users"},
		{"tokens", fakeDBCounts{tokens: -1}, "--tokens"},
		{"uploads", fakeDBCounts{uploads: -1}, "--uploads"},
		{"files", fakeDBCounts{files: -1}, "--files"},
		{"anonymous uploads", fakeDBCounts{anonymousUploads: -1}, "--anon-uploads"},
		{"downloaded uploads", fakeDBCounts{downloadedUploads: -1}, "--downloaded-uploads"},
		{"downloads", fakeDBCounts{downloads: -1}, "--downloads"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFakeDBCounts(tt.counts)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.flag)
		})
	}
}

func TestValidateFakeDBCountsAcceptsZeroCounts(t *testing.T) {
	require.NoError(t, validateFakeDBCounts(fakeDBCounts{}))
}

func TestSeedFakeFileDownloadsPreservesAggregate(t *testing.T) {
	upload := common.NewUpload()
	for range 10 {
		file := upload.NewFile()
		file.Status = common.FileUploaded
		file.Size = 1000
	}

	lastDownloadedAt := time.Now()
	seeded, fileBytes := seedFakeFileDownloads(upload.Files, 228, lastDownloadedAt)

	var total int64
	var totalBytes int64
	for _, file := range upload.Files {
		total += file.DownloadCount
		if file.DownloadCount > 0 {
			require.NotNil(t, file.LastDownloadedAt)
			require.Equal(t, lastDownloadedAt, *file.LastDownloadedAt)
			require.Greater(t, fileBytes[file.ID], int64(0), "a downloaded non-empty file must seed non-zero bytes")
		}
		totalBytes += fileBytes[file.ID]
	}

	require.Equal(t, int64(228), seeded)
	require.Equal(t, int64(228), total)
	// Each of the 228 simulated downloads transfers a randomized 50-100% of a
	// 1000-byte file (fakeDownloadBytes), so the aggregate must land within
	// that same bound.
	require.GreaterOrEqual(t, totalBytes, int64(228*500), "seeded bytes must not fall below the 50% floor")
	require.Less(t, totalBytes, int64(228*1000), "seeded bytes must stay below a full-size transfer for every download")
}

// TestFakeDownloadBytesWithinBounds pins fakeDownloadBytes' contract: a
// randomized 50-100% (inclusive/exclusive) fraction of a positive size, and 0
// for a non-positive size.
func TestFakeDownloadBytesWithinBounds(t *testing.T) {
	require.Zero(t, fakeDownloadBytes(0))
	require.Zero(t, fakeDownloadBytes(-1))

	for range 200 {
		bytes := fakeDownloadBytes(1000)
		require.GreaterOrEqual(t, bytes, int64(500), "fakeDownloadBytes must not go below the 50%% floor")
		require.Less(t, bytes, int64(1000), "fakeDownloadBytes must stay below the full file size")
	}
}

// TestCreateFakeDownloadRollupsAttributesUploadOwner pins the fakedb handoff
// regression: seeded daily rollups must carry the same user_id/token
// attribution the production recordDailyDownloads path writes, or
// GetUserActivityStatsDaily's `user_id = ?` filter never matches a fake row
// and every fake user's Home dashboard shows empty download windows. The
// upload-entity rollup may be spread across several days, so this asserts
// attribution on every row plus the per-entity downloads sum, rather than an
// exact row count.
func TestCreateFakeDownloadRollupsAttributesUploadOwner(t *testing.T) {
	backend := newFakeDBTestBackend(t)

	upload := common.NewUpload()
	upload.User = "user-42"
	upload.Token = "token-42"
	file := upload.NewFile()
	file.Status = common.FileUploaded

	lastDownloadedAt := time.Now()
	upload.DownloadCount = 5
	upload.LastDownloadedAt = &lastDownloadedAt
	file.DownloadCount = 5
	file.LastDownloadedAt = &lastDownloadedAt

	require.NoError(t, createFakeDownloadRollups(backend, upload, nil))

	rows := collectDownloadStatsDaily(t, backend)
	require.NotEmpty(t, rows)

	var uploadDownloads, fileDownloads int64
	var sawUpload, sawFile bool
	for _, row := range rows {
		require.Equal(t, "user-42", row.UserID, "rollup %s/%s must be attributed to the upload owner", row.EntityType, row.EntityID)
		require.Equal(t, "token-42", row.Token, "rollup %s/%s must be attributed to the upload's token", row.EntityType, row.EntityID)
		switch row.EntityType {
		case common.DownloadStatsEntityUpload:
			require.Equal(t, upload.ID, row.EntityID)
			uploadDownloads += row.Downloads
			sawUpload = true
		case common.DownloadStatsEntityFile:
			require.Equal(t, file.ID, row.EntityID)
			fileDownloads += row.Downloads
			sawFile = true
		}
	}
	require.True(t, sawUpload, "expected at least one upload-entity rollup row")
	require.True(t, sawFile, "expected a file-entity rollup row")
	require.Equal(t, int64(5), uploadDownloads, "spread upload-entity rollups must sum to the seeded lifetime count")
	require.Equal(t, int64(5), fileDownloads, "file-entity rollup keeps the single-day shape (not spread)")
}

// TestCreateFakeDownloadRollupsAnonymousUploadHasEmptyAttribution ensures
// anonymous uploads (no owner, no token) keep their empty User/Token verbatim
// rather than acquiring some sentinel value — matching the production
// contract documented on common.DownloadStatsDaily.
func TestCreateFakeDownloadRollupsAnonymousUploadHasEmptyAttribution(t *testing.T) {
	backend := newFakeDBTestBackend(t)

	upload := common.NewUpload()
	file := upload.NewFile()
	file.Status = common.FileUploaded

	lastDownloadedAt := time.Now()
	upload.DownloadCount = 3
	upload.LastDownloadedAt = &lastDownloadedAt
	file.DownloadCount = 3
	file.LastDownloadedAt = &lastDownloadedAt

	require.NoError(t, createFakeDownloadRollups(backend, upload, nil))

	rows := collectDownloadStatsDaily(t, backend)
	require.NotEmpty(t, rows)
	for _, row := range rows {
		require.Empty(t, row.UserID)
		require.Empty(t, row.Token)
	}
}

// TestCreateFakeDownloadRollupsSkipsUndownloadedUploads guards the early
// return: an upload with no fake download activity must not create any
// rollup row at all (not even an empty-attribution one).
func TestCreateFakeDownloadRollupsSkipsUndownloadedUploads(t *testing.T) {
	backend := newFakeDBTestBackend(t)

	upload := common.NewUpload()
	upload.User = "user-1"

	require.NoError(t, createFakeDownloadRollups(backend, upload, nil))

	rows := collectDownloadStatsDaily(t, backend)
	require.Empty(t, rows)
}

// TestCreateFakeDownloadRollupsSeedsBytes pins the fakedb bytes-seeding fix:
// fake download activity must seed plausible bytes onto both the daily
// rollups (so the webapp chart's Traffic mode is non-zero) and the
// usage_stats lifetime byte totals (so the bytes tiles are non-zero), not
// just download counts. The upload-entity rollup's bytes must sum (across
// however many days it's spread over) to the total of every file's seeded
// bytes, mirroring how a real day's upload rollup
// accumulates that day's per-file download bytes. File-entity rollups keep
// the single-day shape, so their exact per-row bytes are still pinned.
func TestCreateFakeDownloadRollupsSeedsBytes(t *testing.T) {
	backend := newFakeDBTestBackend(t)

	upload := common.NewUpload()
	upload.User = "user-99"
	fileA := upload.NewFile()
	fileA.Status = common.FileUploaded
	fileB := upload.NewFile()
	fileB.Status = common.FileUploaded

	lastDownloadedAt := time.Now()
	upload.DownloadCount = 8
	upload.LastDownloadedAt = &lastDownloadedAt
	fileA.DownloadCount = 5
	fileA.LastDownloadedAt = &lastDownloadedAt
	fileB.DownloadCount = 3
	fileB.LastDownloadedAt = &lastDownloadedAt

	fileBytes := map[string]int64{fileA.ID: 5000, fileB.ID: 1500}

	require.NoError(t, backend.CreateUpload(upload))
	require.NoError(t, createFakeDownloadRollups(backend, upload, fileBytes))

	rows := collectDownloadStatsDaily(t, backend)
	require.NotEmpty(t, rows)

	var uploadBytes int64
	for _, row := range rows {
		switch {
		case row.EntityType == common.DownloadStatsEntityUpload:
			uploadBytes += row.Bytes
		case row.EntityID == fileA.ID:
			require.Equal(t, int64(5000), row.Bytes)
		case row.EntityID == fileB.ID:
			require.Equal(t, int64(1500), row.Bytes)
		default:
			t.Fatalf("unexpected rollup row: %+v", row)
		}
	}
	require.Equal(t, int64(6500), uploadBytes, "upload rollup bytes must sum to every file's seeded bytes")

	stats, err := backend.GetUserStatistics(upload.User, nil)
	require.NoError(t, err)
	require.Equal(t, int64(6500), stats.Usage.Downloads.Bytes, "usage_stats lifetime bytes must reflect the seeded rollup total too")
}

// TestSpreadFakeDownloadDaysReturnsNilForNonPositive pins the early return:
// nothing to spread when there are no downloads to attribute.
func TestSpreadFakeDownloadDaysReturnsNilForNonPositive(t *testing.T) {
	now := time.Now()
	require.Nil(t, spreadFakeDownloadDays("e", 0, 100, now, now))
	require.Nil(t, spreadFakeDownloadDays("e", -1, 100, now, now))
}

// TestSpreadFakeDownloadDaysSumsExactly pins the core spread-days contract:
// however many days one entity's fake downloads are spread across, the
// chunks' downloads and bytes must always sum to exactly the inputs (rollup
// sums must stay consistent with the seeded lifetime counters), and every day
// must fall within the entity's creation day and today (inclusive). Sampled
// across many entity IDs and totals since the split is randomized (seeded by
// entity ID).
func TestSpreadFakeDownloadDaysSumsExactly(t *testing.T) {
	now := time.Now()
	createdAt := now.Add(-20 * 24 * time.Hour)

	for i := range 200 {
		entityID := fmt.Sprintf("entity-%d", i)
		downloads := int64(1 + i%97)
		bytes := int64(1000 + i*37)

		chunks := spreadFakeDownloadDays(entityID, downloads, bytes, createdAt, now)
		require.NotEmpty(t, chunks, "entity %s", entityID)
		require.LessOrEqual(t, len(chunks), maxFakeDownloadSpreadDays, "entity %s", entityID)

		var totalDownloads, totalBytes int64
		today := truncateUTCDay(now)
		earliestDay := truncateUTCDay(createdAt)
		for _, c := range chunks {
			require.GreaterOrEqual(t, c.downloads, int64(1), "every spread day must get at least 1 download (entity %s)", entityID)
			require.False(t, c.day.After(today), "chunk day must not be after today (entity %s)", entityID)
			require.False(t, c.day.Before(earliestDay), "chunk day must not predate the entity's creation day (entity %s)", entityID)
			totalDownloads += c.downloads
			totalBytes += c.bytes
		}
		require.Equal(t, downloads, totalDownloads, "spread downloads must sum to the input total (entity %s)", entityID)
		require.Equal(t, bytes, totalBytes, "spread bytes must sum to the input total (entity %s)", entityID)
	}
}

// TestSpreadFakeDownloadDaysDecorrelatesAcrossEntities pins the aggregation
// fix: entities must NOT all be forced to include the same anchor day (an
// earlier draft always included the entity's own most-recent day, which still
// let "today" dominate the aggregate chart whenever many entities' own recent
// day happened to coincide — see the rationale on spreadFakeDownloadDays).
// With a wide-open window (old creation date) and many entities, at least one
// sampled entity must NOT include today among its spread days.
func TestSpreadFakeDownloadDaysDecorrelatesAcrossEntities(t *testing.T) {
	now := time.Now()
	createdAt := now.Add(-90 * 24 * time.Hour)
	today := truncateUTCDay(now)

	sawEntityWithoutToday := false
	for i := range 100 {
		chunks := spreadFakeDownloadDays(fmt.Sprintf("decorrelate-entity-%d", i), 5, 1000, createdAt, now)
		includesToday := false
		for _, c := range chunks {
			if c.day.Equal(today) {
				includesToday = true
				break
			}
		}
		if !includesToday {
			sawEntityWithoutToday = true
			break
		}
	}
	require.True(t, sawEntityWithoutToday, "expected at least one sampled entity's spread to NOT include today")
}

// TestSpreadFakeDownloadDaysProducesMultipleDays proves the actual fix: with a
// download total comfortably larger than maxFakeDownloadSpreadDays and a full
// 30-day window available, at least some entities must spread across more
// than one day (the pre-fix behavior always produced exactly one day). Not
// every entity is guaranteed to roll more than 1 day (numDays is itself
// randomized), so this samples many entity IDs and requires the spread to
// show up somewhere — vanishingly unlikely to fail if the fix is in place,
// certain to fail if spreadFakeDownloadDays regressed to always-1-day.
func TestSpreadFakeDownloadDaysProducesMultipleDays(t *testing.T) {
	now := time.Now()
	createdAt := now.Add(-90 * 24 * time.Hour) // older than the 30-day window, so the full window is available

	sawMultipleDays := false
	for i := range 100 {
		chunks := spreadFakeDownloadDays(fmt.Sprintf("wide-entity-%d", i), 250, 100_000, createdAt, now)
		if len(chunks) > 1 {
			sawMultipleDays = true
			break
		}
	}
	require.True(t, sawMultipleDays, "expected at least one sampled entity to spread its downloads across multiple days")
}

// TestSpreadFakeDownloadDaysIsReproducible pins that the split is seeded from
// the entity ID (not the shared global math/rand source), so re-running
// fakedb with the same generated IDs reproduces the same shape.
func TestSpreadFakeDownloadDaysIsReproducible(t *testing.T) {
	now := time.Now()
	createdAt := now.Add(-30 * 24 * time.Hour)

	first := spreadFakeDownloadDays("stable-id", 42, 9000, createdAt, now)
	second := spreadFakeDownloadDays("stable-id", 42, 9000, createdAt, now)
	require.Equal(t, first, second)
}

// TestSpreadFakeDownloadDaysNeverPredatesCreation pins the window's lower
// bound: a very recently created entity (created well within the 30-day
// window) must not have any rollup day before its creation day, even though
// the window would otherwise reach back 29 days.
func TestSpreadFakeDownloadDaysNeverPredatesCreation(t *testing.T) {
	now := time.Now()
	createdAt := now.Add(-2 * 24 * time.Hour) // created 2 days ago

	for i := range 50 {
		chunks := spreadFakeDownloadDays(fmt.Sprintf("recent-entity-%d", i), 10, 5000, createdAt, now)
		earliestDay := truncateUTCDay(createdAt)
		for _, c := range chunks {
			require.False(t, c.day.Before(earliestDay), "chunk day must not predate creation")
		}
	}
}

func TestApplyFakeUploadLifecycleWithFiniteTTL(t *testing.T) {
	upload := common.NewUpload()
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	minCreatedAt := now.Add(-24 * time.Hour)
	ttl := 3600

	applyFakeUploadLifecycleWithTTL(upload, minCreatedAt, now, ttl)

	require.Equal(t, ttl, upload.TTL)
	require.False(t, upload.CreatedAt.Before(minCreatedAt))
	require.False(t, upload.CreatedAt.After(now))
	require.NotNil(t, upload.ExpireAt)
	require.Equal(t, upload.CreatedAt.Add(time.Duration(ttl)*time.Second), *upload.ExpireAt)
	require.True(t, upload.ExpireAt.After(now))
}

func TestApplyFakeUploadLifecycleWithUnlimitedTTL(t *testing.T) {
	upload := common.NewUpload()
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	minCreatedAt := now.Add(-24 * time.Hour)

	applyFakeUploadLifecycleWithTTL(upload, minCreatedAt, now, 0)

	require.Zero(t, upload.TTL)
	require.False(t, upload.CreatedAt.Before(minCreatedAt))
	require.False(t, upload.CreatedAt.After(now))
	require.Nil(t, upload.ExpireAt)
}

func TestAddFakeUploadFilesUsesUploadCreatedAt(t *testing.T) {
	upload := common.NewUpload()
	upload.CreatedAt = time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	addFakeUploadFiles(upload, 3, "file", 1024)

	require.Len(t, upload.Files, 3)
	for _, file := range upload.Files {
		require.Equal(t, upload.CreatedAt, file.CreatedAt)
		require.Equal(t, common.FileUploaded, file.Status)
		require.NotEmpty(t, file.Name)
		require.NotEmpty(t, file.Type)
	}
}

func TestFakeDownloadSeederDoesNotSeedUploadWithoutFiles(t *testing.T) {
	upload := common.NewUpload()
	seeder := newFakeDownloadSeeder(1, 1, 250)

	seeder.seedUpload(upload)

	require.Zero(t, upload.DownloadCount)
	require.Nil(t, upload.LastDownloadedAt)
	require.Zero(t, seeder.selected)
}

func TestFakeDownloadSeederDoesNotSeedLifecycleSpecialUploads(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*common.Upload)
	}{
		{"one shot", func(upload *common.Upload) { upload.OneShot = true }},
		{"stream", func(upload *common.Upload) { upload.Stream = true }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upload := common.NewUpload()
			tt.setup(upload)
			upload.NewFile().Status = common.FileUploaded
			seeder := newFakeDownloadSeeder(1, 1, 250)

			seeder.seedUpload(upload)

			require.Zero(t, upload.DownloadCount)
			require.Nil(t, upload.LastDownloadedAt)
			require.Zero(t, upload.Files[0].DownloadCount)
			require.Nil(t, upload.Files[0].LastDownloadedAt)
			require.Zero(t, seeder.selected)
		})
	}
}

// TestFakeDownloadSeederSetsUploadDownloadedBytes pins that seedUpload sets
// upload.DownloadedBytes to exactly the sum of the fileBytes map it returns —
// the same total createFakeDownloadRollups later feeds into
// usage_stats.downloaded_bytes via FixtureSeedDownloadedBytes — so the upload row and
// the usage counters agree on fakedb-generated data.
func TestFakeDownloadSeederSetsUploadDownloadedBytes(t *testing.T) {
	upload := common.NewUpload()
	file := upload.NewFile()
	file.Status = common.FileUploaded
	file.Size = 1000
	// totalUploads=1, target=1: remainingTarget stays > 0 and rand.Intn(1) is
	// always 0, so selection is deterministic.
	seeder := newFakeDownloadSeeder(1, 1, 5)

	fileBytes := seeder.seedUpload(upload)

	require.NotZero(t, upload.DownloadCount, "seedUpload must assign a positive download count when selected")
	require.NotNil(t, upload.LastDownloadedAt)
	require.NotEmpty(t, fileBytes)

	var wantBytes int64
	for _, b := range fileBytes {
		wantBytes += b
	}
	require.Equal(t, wantBytes, upload.DownloadedBytes, "upload.DownloadedBytes must equal the sum of the returned per-file bytes")
	require.NotZero(t, upload.DownloadedBytes, "file.Size=1000 guarantees at least one non-zero fakeDownloadBytes sample")
}

// TestForceSeedUploadAlwaysSeedsRegardlessOfTarget pins forceSeedUpload's
// whole point: it bypasses seedUpload's probabilistic per-upload
// selection gate entirely, so a caller with a fully exhausted/zero target
// (e.g. every upload already selected, or target=0) still gets a guaranteed
// nonzero download count/bytes total — used by fakedb's admin-owned demo
// uploads, which must show real data regardless of the random draw.
func TestForceSeedUploadAlwaysSeedsRegardlessOfTarget(t *testing.T) {
	upload := common.NewUpload()
	file := upload.NewFile()
	file.Status = common.FileUploaded
	file.Size = 1000

	// target=0 (maxDownloads=5 keeps the seeder itself viable): seedUpload
	// would never select this upload, but forceSeedUpload must not care.
	seeder := newFakeDownloadSeeder(1, 0, 5)

	fileBytes := seeder.forceSeedUpload(upload)

	require.NotZero(t, upload.DownloadCount, "forceSeedUpload must assign a positive download count unconditionally")
	require.NotNil(t, upload.LastDownloadedAt)
	require.NotEmpty(t, fileBytes)
	require.Equal(t, 1, seeder.selected, "forceSeedUpload must still track the seeded count for logging")

	var wantBytes int64
	for _, b := range fileBytes {
		wantBytes += b
	}
	require.Equal(t, wantBytes, upload.DownloadedBytes)
}

// TestForceSeedUploadWithoutFilesReturnsNilAndLeavesSelectedUntouched mirrors
// TestFakeDownloadSeederDoesNotSeedUploadWithoutFiles for the forced path: an
// upload with no files has nothing to seed, so forceSeedUpload must not
// pretend it succeeded.
func TestForceSeedUploadWithoutFilesReturnsNilAndLeavesSelectedUntouched(t *testing.T) {
	upload := common.NewUpload()
	seeder := newFakeDownloadSeeder(1, 1, 5)

	fileBytes := seeder.forceSeedUpload(upload)

	require.Nil(t, fileBytes)
	require.Zero(t, upload.DownloadCount)
	require.Nil(t, upload.LastDownloadedAt)
	require.Zero(t, seeder.selected)
}

func TestApplyFakeUploadFeaturesDoesNotGenerateStreamOrOneShotUploads(t *testing.T) {
	passwordHash := "hash"
	for range 1000 {
		upload := common.NewUpload()
		upload.TTL = 3600

		applyFakeUploadFeatures(upload, "comment", passwordHash)

		require.False(t, upload.Stream)
		require.False(t, upload.OneShot)
	}
}

func TestCapitalize(t *testing.T) {
	require.Equal(t, "", capitalize(""))
	require.Equal(t, "Alice", capitalize("alice"))
	require.Equal(t, "Élodie", capitalize("élodie"))
}

func TestRemoveExistingFakeDBRemovesSQLiteSidecars(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test-plik.db")
	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		require.NoError(t, os.WriteFile(path, []byte("stale"), 0o600))
	}

	require.NoError(t, removeExistingFakeDB(dbPath))

	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		_, err := os.Stat(path)
		require.True(t, os.IsNotExist(err), "expected %s to be removed", path)
	}
}

func TestRemoveExistingFakeDBReportsRemoveErrors(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db")
	require.NoError(t, os.Mkdir(dbPath, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dbPath, "child"), []byte("stale"), 0o600))

	err := removeExistingFakeDB(dbPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "remove ")
}
