package metadata

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

// createMetadata seeds a user (with a token), one completed upload/file, and
// a setting. It returns the seeded user so callers can assert exact counters
// after an export/import roundtrip.
func createMetadata(t *testing.T, b *Backend) (user *common.User) {
	user = common.NewUser(common.ProviderLocal, "user")
	user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	file.Size = 123
	upload.User = user.ID
	upload.Token = user.Tokens[0].Token
	createUpload(t, b, upload)

	setting := &common.Setting{Key: "foo", Value: "bar"}
	err := b.CreateSetting(setting)
	require.NoError(t, err)

	return user
}

func TestBackend_Export(t *testing.T) {
	b := newTestMetadataBackend()
	defer func() {
		if b != nil {
			shutdownTestMetadataBackend(b)
		}
	}()

	user := createMetadata(t, b)

	path := "/tmp/plik.metadata.test.snappy.gob"
	err := b.Export(path)
	require.NoError(t, err, "export error %s", err)
	shutdownTestMetadataBackend(b)

	b = newTestMetadataBackend()

	err = b.Import(path, &ImportOptions{})
	require.NoError(t, err, "import error %s", err)

	// Not just "no error": the imported side must have rebuilt the completed
	// upload/file into both the user and server usage_stats rows.
	userStats, err := b.GetUserStatistics(user.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, userStats.Uploads)
	require.Equal(t, 1, userStats.Files)
	require.Equal(t, int64(123), userStats.TotalSize)
	require.Equal(t, 1, userStats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, userStats.Usage.Lifetime.Files)
	require.Equal(t, int64(123), userStats.Usage.Lifetime.TotalSize)

	serverStats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 1, serverStats.Files)
	require.Equal(t, int64(123), serverStats.TotalSize)
	require.Equal(t, 1, serverStats.LifetimeUsers)
}

func TestBackend_ExportRemovedFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	upload.NewFile() // never uploaded: status stays FileMissing

	createUpload(t, b, upload)

	// Soft delete upload
	err := b.RemoveUpload(upload.ID)
	require.NoError(t, err, "unable to delete upload")

	path := "/tmp/plik.metadata.test.snappy.gob"
	err = b.Export(path)
	require.NoError(t, err, "export error %s", err)

	shutdownTestMetadataBackend(b)
	b = newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	err = b.Import(path, &ImportOptions{})
	require.NoError(t, err, "import error %s", err)

	serverStats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 0, serverStats.Uploads, "soft-deleted upload must not be current")
	require.Equal(t, 1, serverStats.Usage.Lifetime.Uploads)
	// The documented rebuild approximation in action: a file that never
	// received a single byte (FileMissing -> FileDeleted cleanup) still gets
	// lifetime-counted by import, because a bare status cannot distinguish
	// it from a genuinely completed-then-removed file. Its size is 0, so the
	// approximation is invisible in LifetimeTotalSize but visible in LifetimeFiles.
	require.Equal(t, 1, serverStats.Usage.Lifetime.Files)
	require.Equal(t, int64(0), serverStats.Usage.Lifetime.TotalSize)
}

func TestBackend_ExportImportDownloadStatsDaily(t *testing.T) {
	b := newTestMetadataBackend()
	defer func() {
		if b != nil {
			shutdownTestMetadataBackend(b)
		}
	}()

	user := common.NewUser(common.ProviderLocal, "export-rollup-attribution-user")
	token := user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{User: user.ID, Token: token.Token, Comments: "export trending"}
	file := upload.NewFile()
	file.Name = "export-trending.txt"
	file.Size = 42
	file.Status = common.FileUploaded
	createUpload(t, b, upload)
	require.NoError(t, b.RecordFileDownload(upload, file, 1024, true))

	path := filepath.Join(t.TempDir(), "plik.metadata.test.snappy.gob")
	err := b.Export(path)
	require.NoError(t, err, "export error %s", err)
	shutdownTestMetadataBackend(b)
	b = nil

	b = newTestMetadataBackend()

	err = b.Import(path, &ImportOptions{})
	require.NoError(t, err, "import error %s", err)

	uploads, err := b.GetTrendingUploads("1d", "", 10)
	require.NoError(t, err)
	require.NotEmpty(t, uploads)
	require.Equal(t, upload.ID, uploads[0].ID)
	require.Equal(t, int64(1), uploads[0].DownloadCount)

	files, err := b.GetTrendingFiles("1d", 10)
	require.NoError(t, err)
	require.NotEmpty(t, files)
	require.Equal(t, file.ID, files[0].ID)
	require.Equal(t, int64(1), files[0].DownloadCount)

	// Daily rollup bytes survive the export/import roundtrip: the recording above
	// served 1024 bytes, which must be restored verbatim on both entity rows.
	requireDailyRollup(t, b, common.DownloadStatsEntityUpload, upload.ID, 1, 1024)
	requireDailyRollup(t, b, common.DownloadStatsEntityFile, file.ID, 1, 1024)

	// Rollup attribution (user_id/token) also survives the roundtrip: gob
	// encodes struct fields by name, so the new columns travel through export
	// and import like any other field.
	requireDailyRollupAttribution(t, b, common.DownloadStatsEntityUpload, upload.ID, user.ID, token.Token)
	requireDailyRollupAttribution(t, b, common.DownloadStatsEntityFile, file.ID, user.ID, token.Token)

	// The user daily series is rebuilt straight from the imported rollup rows,
	// so it must reflect the roundtripped attribution too.
	series, err := b.GetUserActivityStatsDaily(user.ID, 1)
	require.NoError(t, err)
	require.Len(t, series, 1)
	require.Equal(t, int64(1), series[0].Downloads)
	require.Equal(t, int64(1024), series[0].DownloadedBytes)

	// The upload row's own downloaded_bytes column survives the roundtrip
	// verbatim: gob encodes every struct field by name (like DownloadCount
	// above), and CreateUpload's plain tx.Create(upload) call during import
	// restores it as-is — there is now a per-upload byte source.
	importedUpload, err := b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1024), importedUpload.DownloadedBytes, "upload.downloaded_bytes must survive the export/import roundtrip verbatim, exactly like download_count")

	// Usage-scope downloads are rebuilt on import from upload.download_count
	// (CreateUpload's delta.downloads = upload.DownloadCount). downloaded_bytes
	// now HAS a per-upload source too (asserted above), but CreateUpload
	// deliberately does NOT fold it into the usage delta: fakedb's separate
	// FixtureSeedDownloadedBytes call (and any future bytes-only reseed helper) would
	// double-count against it if it did. So the rebuilt usage_stats row still
	// starts at 0 bytes on import — the same "bytes tracked since upgrade" /
	// backfill=0 behavior already documented for uploaded_bytes.
	requireUsageStats(t, b, user.ID, "", 1, 0)
}

// TestBackend_ExportImportUploadStatsDaily pins that upload rollups survive an
// export/import roundtrip as records (uploads AND wire bytes), and that the
// imported row is authoritative over CreateUpload's replay of the day's Uploads
// count. Old exports simply lack these rows, so the imported table starts at
// zeros for them.
func TestBackend_ExportImportUploadStatsDaily(t *testing.T) {
	b := newTestMetadataBackend()
	defer func() {
		if b != nil {
			shutdownTestMetadataBackend(b)
		}
	}()

	user := common.NewUser(common.ProviderLocal, "export-upload-rollup-user")
	token := user.NewToken()
	createUser(t, b, user)

	today := statsDay(time.Now())

	// A completed upload: CreateUpload writes the day's uploads +1 and the fused
	// completion writes the wire bytes, so the source row is {uploads:1, bytes:900}.
	upload := &common.Upload{User: user.ID, Token: token.Token}
	file := upload.NewFile()
	file.Name = "export-upload.txt"
	file.Status = common.FileMissing
	createUpload(t, b, upload)
	require.NoError(t, b.UpdateFileStatus(file, common.FileMissing, common.FileUploading))
	file.Size = 900
	file.Status = common.FileUploaded
	require.NoError(t, b.UpdateFile(file, common.FileUploading))
	requireUploadRollup(t, b, today, user.ID, token.Token, 1, 900)

	path := filepath.Join(t.TempDir(), "plik.metadata.upload-rollup.snappy.gob")
	require.NoError(t, b.Export(path))
	shutdownTestMetadataBackend(b)
	b = nil

	b = newTestMetadataBackend()
	require.NoError(t, b.Import(path, &ImportOptions{}))

	// Uploads count and wire bytes both survive: CreateUpload replays uploads=1
	// (bytes 0), then the imported row overwrites with the authoritative
	// {uploads:1, bytes:900} instead of colliding on the primary key.
	requireUploadRollup(t, b, today, user.ID, token.Token, 1, 900)

	// The user upload series is rebuilt straight from the roundtripped rollup.
	series, err := b.GetUserActivityStatsDaily(user.ID, 1)
	require.NoError(t, err)
	require.Len(t, series, 1)
	require.Equal(t, int64(1), series[0].Uploads)
	require.Equal(t, int64(900), series[0].UploadedBytes)

	// uploaded_bytes is a usage_stats counter, which is NOT exported: like
	// downloaded_bytes it is rebuilt as 0 on import (CreateFile passes wireBytes=0),
	// the documented "since upgrade / backfill=0" behavior for both byte counters.
	requireUploadedBytes(t, b, user.ID, "", 0)
}

// TestBackend_ExportImportLifetimeCounterApproximation seeds a backend with a
// realistic mix of file lifecycle outcomes:
//   - a completed, retained file
//   - a one-shot-consumed file (FileUploaded -> FileRemoved), on a tokenless
//     upload
//   - a failed-cleanup file (FileUploading -> FileRemoved via upload
//     removal) that the live path never lifetime-counts
//   - a stream-consumed file (FileUploading -> FileDeleted for a Stream
//     upload) that the live path does lifetime-count
//   - an anonymous, completed upload
//   - a recorded download and its daily rollup
//
// It exports, imports into a fresh backend, and pins the exact rebuilt
// counters. Import cannot distinguish the failed-cleanup file from a
// genuinely completed-then-removed one (see ARCHITECTURE.md "Import,
// Export, and Repair"), so the imported-side numbers intentionally include
// the failed-cleanup file's bytes — this test asserts that documented
// approximation, not the live-accumulated numbers.
func TestBackend_ExportImportLifetimeCounterApproximation(t *testing.T) {
	b := newTestMetadataBackend()
	defer func() {
		if b != nil {
			shutdownTestMetadataBackend(b)
		}
	}()

	alice := common.NewUser(common.ProviderLocal, "roundtrip-alice")
	token := alice.NewToken()
	createUser(t, b, alice)

	// Idle user: exercises LifetimeUsers counting a user with no uploads at all.
	bob := common.NewUser(common.ProviderLocal, "roundtrip-bob")
	createUser(t, b, bob)

	// Deleted-before-export user: common.User has no soft-delete column, so
	// DeleteUser hard-deletes the row and it is entirely absent from the
	// export. This pins the import-rebuild semantics for LifetimeUsers
	// documented in server/ARCHITECTURE.md ("Stats Architecture" / user
	// creation section): the live counter is append-only and never
	// decremented by DeleteUser, but the imported side rebuilds it purely by
	// replaying CreateUser once per exported (i.e. surviving) user, so a
	// deleted user's historical contribution to lifetime_users does not
	// survive an export/import roundtrip.
	carol := common.NewUser(common.ProviderLocal, "roundtrip-carol")
	createUser(t, b, carol)
	deleted, err := b.DeleteUser(carol.ID)
	require.NoError(t, err)
	require.True(t, deleted, "carol must have been deleted")

	// Before export: the live, append-only counter already reflects all three
	// creations, including carol's, because DeleteUser never touches it.
	preExportStats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 3, preExportStats.LifetimeUsers, "live lifetime_users counts every user ever created, deleted or not")

	// Completed, retained file. Downloaded once to exercise rollup survival.
	uploadCompleted := &common.Upload{User: alice.ID, Token: token.Token}
	fileCompleted := uploadCompleted.NewFile()
	fileCompleted.Size = 1000
	fileCompleted.Status = common.FileUploaded
	createUpload(t, b, uploadCompleted)
	require.NoError(t, b.RecordFileDownload(uploadCompleted, fileCompleted, 1024, true))

	// One-shot consumption: completes, then the file is removed. Tokenless
	// (web session) upload, to distinguish user-scope from token-scope.
	uploadOneShot := &common.Upload{User: alice.ID, OneShot: true}
	fileOneShot := uploadOneShot.NewFile()
	fileOneShot.Size = 2000
	fileOneShot.Status = common.FileUploaded
	createUpload(t, b, uploadOneShot)
	require.NoError(t, b.RemoveFile(fileOneShot))

	// Failed upload cleanup: the file never left FileUploading, then the
	// whole upload is removed. The live path never lifetime-counts this.
	uploadFailed := &common.Upload{User: alice.ID, Token: token.Token}
	fileFailed := uploadFailed.NewFile()
	fileFailed.Size = 3000
	fileFailed.Status = common.FileUploading
	createUpload(t, b, uploadFailed)
	require.NoError(t, b.RemoveUpload(uploadFailed.ID))

	// Stream consumption: FileUploading -> FileDeleted for a Stream upload
	// is a successful transfer, and the live path lifetime-counts it too.
	uploadStream := &common.Upload{User: alice.ID, Token: token.Token, Stream: true}
	fileStream := uploadStream.NewFile()
	fileStream.Status = common.FileMissing
	createUpload(t, b, uploadStream)
	require.NoError(t, b.UpdateFileStatus(fileStream, common.FileMissing, common.FileUploading))
	fileStream.Status = common.FileDeleted
	fileStream.Size = 4000
	require.NoError(t, b.UpdateFile(fileStream, common.FileUploading))

	// Anonymous, completed, retained file.
	uploadAnonymous := &common.Upload{}
	fileAnonymous := uploadAnonymous.NewFile()
	fileAnonymous.Size = 500
	fileAnonymous.Status = common.FileUploaded
	createUpload(t, b, uploadAnonymous)

	path := filepath.Join(t.TempDir(), "plik.metadata.roundtrip.snappy.gob")
	err = b.Export(path)
	require.NoError(t, err, "export error %s", err)
	shutdownTestMetadataBackend(b)
	b = nil

	b = newTestMetadataBackend()

	err = b.Import(path, &ImportOptions{})
	require.NoError(t, err, "import error %s", err)

	// alice's user scope aggregates every upload she owns, token'd or not:
	// completed + one-shot + failed(deleted, excluded from current) + stream.
	aliceStats, err := b.GetUserStatistics(alice.ID, nil)
	require.NoError(t, err)
	require.Equal(t, 3, aliceStats.Uploads, "current uploads: completed + one-shot + stream")
	require.Equal(t, 4, aliceStats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, aliceStats.Files, "current files: only the completed one is still retained")
	require.Equal(t, int64(1000), aliceStats.TotalSize)
	require.Equal(t, 4, aliceStats.Usage.Lifetime.Files, "approximation: completed + one-shot-removed + failed-cleanup-removed + stream-deleted")
	require.Equal(t, int64(10000), aliceStats.Usage.Lifetime.TotalSize, "approximation inflates by the failed-cleanup file's 3000 bytes")

	// Token scope excludes the tokenless one-shot upload.
	tokenStats, err := b.GetUserStatistics(alice.ID, &token.Token)
	require.NoError(t, err)
	require.Equal(t, 2, tokenStats.Uploads)
	require.Equal(t, 3, tokenStats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, tokenStats.Files)
	require.Equal(t, int64(1000), tokenStats.TotalSize)
	require.Equal(t, 3, tokenStats.Usage.Lifetime.Files, "approximation: completed + failed-cleanup-removed + stream-deleted")
	require.Equal(t, int64(8000), tokenStats.Usage.Lifetime.TotalSize)

	anonymousStats, err := b.GetUserStatistics(common.AnonymousUserUsageStatsID, nil)
	require.NoError(t, err)
	require.Equal(t, 1, anonymousStats.Uploads)
	require.Equal(t, 1, anonymousStats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, anonymousStats.Files)
	require.Equal(t, int64(500), anonymousStats.TotalSize)
	require.Equal(t, 1, anonymousStats.Usage.Lifetime.Files)
	require.Equal(t, int64(500), anonymousStats.Usage.Lifetime.TotalSize)

	serverStats, err := b.GetServerStatistics()
	require.NoError(t, err)
	require.Equal(t, 4, serverStats.Uploads)
	require.Equal(t, 5, serverStats.Usage.Lifetime.Uploads)
	require.Equal(t, 2, serverStats.Files)
	require.Equal(t, int64(1500), serverStats.TotalSize)
	require.Equal(t, 5, serverStats.Usage.Lifetime.Files)
	require.Equal(t, int64(10500), serverStats.Usage.Lifetime.TotalSize)
	require.Equal(t, 1, serverStats.AnonymousUploads)
	require.Equal(t, int64(500), serverStats.AnonymousSize)
	require.Equal(t, 1, serverStats.AnonymousUsage.Lifetime.Uploads)
	require.Equal(t, int64(500), serverStats.AnonymousUsage.Lifetime.TotalSize)
	require.Equal(t, int64(1), serverStats.Usage.Downloads.Total)

	// alice and bob replay through CreateUser (lifetime_users=1 each), and carol's
	// contribution — folded into the deleted-user tombstone on the source before
	// export — is preserved by exporting/importing the tombstone as a record. So
	// the imported LifetimeUsers is 3, matching the live pre-export value: unlike
	// the old server-row rebuild (which lost a deleted user's history), the
	// tombstone now carries it across the roundtrip. CountUsers is only 2 (carol
	// stays deleted), so LifetimeUsers > user count here, exactly as it does live.
	userCount, err := b.CountUsers("", nil)
	require.NoError(t, err)
	require.Equal(t, 2, int(userCount), "only alice and bob survive into the export")
	require.Equal(t, 3, serverStats.LifetimeUsers, "tombstone preserves the deleted user's lifetime_users across export/import")
	require.Equal(t, preExportStats.LifetimeUsers, serverStats.LifetimeUsers, "roundtrip preserves the live append-only counter")

	// Daily download rollups survive the roundtrip too (complements
	// TestBackend_ExportImportDownloadStatsDaily with a richer mix).
	trendingUploads, err := b.GetTrendingUploads("1d", "", 10)
	require.NoError(t, err)
	require.NotEmpty(t, trendingUploads)
	require.Equal(t, uploadCompleted.ID, trendingUploads[0].ID)
	require.Equal(t, int64(1), trendingUploads[0].DownloadCount)

	trendingFiles, err := b.GetTrendingFiles("1d", 10)
	require.NoError(t, err)
	require.NotEmpty(t, trendingFiles)
	require.Equal(t, fileCompleted.ID, trendingFiles[0].ID)
	require.Equal(t, int64(1), trendingFiles[0].DownloadCount)
}

// TestBackend_ImportDeletedTombstonePreservesHistoricalStartedAt pins that a
// deleted user's historical "Stats since" anchor survives an export/import
// roundtrip. On the source, deleting a user created long ago folds that user's
// started_at into the tombstone; the tombstone is exported as a record and, on
// import, folded into the fresh backend's init tombstone — which must pull the
// server StartedAt back to the historical value rather than the import-time now.
func TestBackend_ImportDeletedTombstonePreservesHistoricalStartedAt(t *testing.T) {
	src := newTestMetadataBackend()

	historical := time.Now().Add(-1000 * time.Hour).Truncate(time.Second)
	old := common.NewUser(common.ProviderLocal, "import-oldest")
	old.CreatedAt = historical
	createUser(t, src, old)
	deleted, err := src.DeleteUser(old.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	// A surviving (newer) user so the imported sum has a live contributor.
	keep := common.NewUser(common.ProviderLocal, "import-newer")
	createUser(t, src, keep)

	srcStats, err := src.GetServerStatistics()
	require.NoError(t, err)
	require.NotNil(t, srcStats.Usage.StartedAt)
	require.WithinDuration(t, historical, *srcStats.Usage.StartedAt, time.Second, "source anchor = deleted oldest user's started_at")

	path := filepath.Join(t.TempDir(), "plik.tombstone-startedat.snappy.gob")
	require.NoError(t, src.Export(path))
	shutdownTestMetadataBackend(src)

	dst := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(dst)
	require.NoError(t, dst.Import(path, &ImportOptions{}))

	dstStats, err := dst.GetServerStatistics()
	require.NoError(t, err)
	require.NotNil(t, dstStats.Usage.StartedAt)
	require.WithinDuration(t, historical, *dstStats.Usage.StartedAt, time.Second, "imported tombstone must carry the historical started_at, not the import-time now")

	// lifetime_users still survives the roundtrip via the tombstone (old + keep).
	require.Equal(t, 2, dstStats.LifetimeUsers, "deleted + surviving user both counted")
}

// TestBackend_ImportDeletedTombstoneTwiceDoubleCountsLifetimeCounters pins
// the documented consequence of ImportDeletedUsageTombstone's fold semantics
// (`col = col + src`): production imports target a fresh DB (see the "Import,
// Export, and Repair" section of server/ARCHITECTURE.md), and folding is the
// correct behavior under that contract. But replaying the SAME tombstone-bearing export a
// second time into an already-populated destination is not guarded against —
// every other imported object (the user row, in particular) fails loudly on
// its primary-key conflict, but the tombstone fold has no such guard and
// silently adds the source counters in again. This test pins that double-count
// as the accepted consequence of re-importing into a non-fresh DB, not a bug
// to fix.
func TestBackend_ImportDeletedTombstoneTwiceDoubleCountsLifetimeCounters(t *testing.T) {
	src := newTestMetadataBackend()

	// A surviving user makes the export non-trivial: it is the object that
	// demonstrates the fresh-DB contract enforcing itself on replay (its row
	// already exists in dst after the first import, so the second import's
	// CreateUser call hits a primary-key conflict) — in contrast with the
	// tombstone fold below, which has no such guard.
	keep := common.NewUser(common.ProviderLocal, "tombstone-double-import-keep")
	createUser(t, src, keep)

	deletedUser := common.NewUser(common.ProviderLocal, "tombstone-double-import-deleted")
	createUser(t, src, deletedUser)
	deleted, err := src.DeleteUser(deletedUser.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	srcTombstone, err := src.GetDeletedUsageTombstone()
	require.NoError(t, err)
	require.NotNil(t, srcTombstone)
	require.Equal(t, 1, srcTombstone.LifetimeUsers, "sanity: the deleted user folded lifetime_users=1 into the source tombstone")

	path := filepath.Join(t.TempDir(), "plik.tombstone-double-import.snappy.gob")
	require.NoError(t, src.Export(path))
	shutdownTestMetadataBackend(src)

	dst := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(dst)

	require.NoError(t, dst.Import(path, &ImportOptions{}))
	afterFirst, err := dst.GetDeletedUsageTombstone()
	require.NoError(t, err)
	require.NotNil(t, afterFirst)
	require.Equal(t, 1, afterFirst.LifetimeUsers, "first import folds the tombstone exactly once")
	countAfterFirst, err := dst.CountUsers("", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), countAfterFirst, "only the surviving user is present after the first import")

	// Re-import the SAME export into the now non-fresh dst. keep's user row
	// hits a primary-key conflict on replay — the fresh-DB contract enforcing
	// itself — so IgnoreErrors is required to let the import reach the
	// tombstone object at all; the tombstone fold itself never errors, which
	// is exactly the gap this test documents.
	require.NoError(t, dst.Import(path, &ImportOptions{IgnoreErrors: true}))
	countAfterSecond, err := dst.CountUsers("", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), countAfterSecond, "keep's user row rejects the replay as a primary-key conflict, proving the fresh-DB contract is enforced elsewhere")

	afterSecond, err := dst.GetDeletedUsageTombstone()
	require.NoError(t, err)
	require.NotNil(t, afterSecond)
	require.Equal(t, 2, afterSecond.LifetimeUsers, "re-importing the same tombstone into a non-fresh DB double-counts lifetime_users — the accepted consequence of the fresh-DB import contract, not a bug")
}
