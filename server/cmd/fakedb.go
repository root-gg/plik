package cmd

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"sort"
	"time"
	"unicode"

	"github.com/root-gg/logger"
	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadata"
)

// fakedb* variables are bound to cobra flags and read once by the command
// handler. They stay package-level to match the other plikd command files.
var fakedbUsers int
var fakedbTokens int
var fakedbUploads int
var fakedbFiles int
var fakedbAnonUploads int
var fakedbDownloadedUploads int
var fakedbDownloads int
var fakedbOutput string

// fakedbAdminUploads is the (fixed, not flag-configurable) number of uploads
// attributed to the admin demo login so its own Home dashboard has real data
// out of the box — see the fakedb() call site for why these are force-seeded
// rather than left to the random per-upload download draw.
const fakedbAdminUploads = 6

// fakedbCmd generates a standalone SQLite metadata database for local UI,
// stats, and performance testing.
var fakedbCmd = &cobra.Command{
	Use:   "fakedb",
	Short: "Generate a fake SQLite database populated with test data",
	Long: `Generate a fake Plik SQLite database with randomised users, tokens,
uploads, and files. Useful for UI testing and performance benchmarks.

An admin user (login: admin, password: plikplik) is always created.`,
	Run: fakedb,
}

func init() {
	rootCmd.AddCommand(fakedbCmd)

	fakedbCmd.Flags().IntVar(&fakedbUsers, "users", 1000, "number of users to create")
	fakedbCmd.Flags().IntVar(&fakedbTokens, "tokens", 5, "tokens per user")
	fakedbCmd.Flags().IntVar(&fakedbUploads, "uploads", 100, "uploads per user")
	fakedbCmd.Flags().IntVar(&fakedbFiles, "files", 10, "files per upload")
	fakedbCmd.Flags().IntVar(&fakedbAnonUploads, "anon-uploads", 100, "anonymous uploads (no user)")
	fakedbCmd.Flags().IntVar(&fakedbDownloadedUploads, "downloaded-uploads", 1000, "number of uploads with fake download activity")
	fakedbCmd.Flags().IntVar(&fakedbDownloads, "downloads", 250, "maximum fake downloads per downloaded upload (0 disables fake downloads)")
	fakedbCmd.Flags().StringVar(&fakedbOutput, "output", "/tmp/test-plik.db", "output SQLite database path")
}

// firstNames is a small identity pool used to make fake users readable in the
// UI while keeping login/email generation unique via the loop index.
var firstNames = []string{
	"alice", "bob", "charlie", "diana", "eve", "frank", "grace", "hector",
	"iris", "jack", "kate", "leo", "mia", "noah", "olivia", "paul",
	"quinn", "rosa", "sam", "tina", "uma", "victor", "wendy", "xander",
	"yara", "zach", "ada", "ben", "clara", "dario", "elsa", "felix",
	"gina", "hugo", "ivy", "jake", "lana", "max", "nora", "oscar",
}

// lastNames pairs with firstNames for readable fake display names and logins.
var lastNames = []string{
	"smith", "jones", "brown", "wilson", "taylor", "thomas", "white",
	"harris", "martin", "garcia", "martinez", "robinson", "clark", "lewis",
	"lee", "walker", "hall", "allen", "young", "king", "wright", "scott",
	"green", "baker", "adams", "nelson", "hill", "moore", "jackson", "davis",
}

// emailDomains gives generated users varied but non-sensitive fake addresses.
var emailDomains = []string{
	"gmail.com", "outlook.com", "yahoo.com", "proton.me", "example.org",
	"acme.co", "corp.net", "dev.io", "company.com", "test.local",
}

// fakeProviders exercises provider filters and display paths without requiring
// external OAuth fixtures.
var fakeProviders = []string{"local", "google", "github", "ovh", "oidc"}

// fakeUploadTTLBuckets mirrors representative server TTL policy buckets used by
// stats and filter UIs.
var fakeUploadTTLBuckets = []int{0, 1800, 3600, 86400, 604800, 2592000, 5184000}

// fileTypes covers common preview/download MIME categories used by the web UI.
var fileTypes = []struct {
	ext         string
	contentType string
}{
	{".txt", "text/plain; charset=utf-8"},
	{".pdf", "application/pdf"},
	{".jpg", "image/jpeg"},
	{".png", "image/png"},
	{".gif", "image/gif"},
	{".bmp", "image/bmp"},
	{".webp", "image/webp"},
	{".ico", "image/x-icon"},
	{".zip", "application/zip"},
	{".tar.gz", "application/gzip"},
	{".go", "text/plain; charset=utf-8"},
	{".js", "text/plain; charset=utf-8"},
	{".csv", "text/plain; charset=utf-8"},
	{".json", "application/json"},
	{".xml", "text/xml; charset=utf-8"},
	{".md", "text/plain; charset=utf-8"},
	{".py", "text/plain; charset=utf-8"},
	{".sh", "text/plain; charset=utf-8"},
	{".log", "text/plain; charset=utf-8"},
	{".doc", "application/msword"},
	{".ps", "application/postscript"},
	{".bin", "application/octet-stream"},
}

// fakedb is the cobra handler for `plikd fakedb`; it intentionally creates only
// metadata rows, not data-backend objects, because the command is for UI and
// stats fixtures.
func fakedb(cmd *cobra.Command, args []string) {
	log := logger.NewLogger().SetMinLevel(logger.INFO)
	if err := validateFakeDBCounts(fakeDBCounts{
		users:             fakedbUsers,
		tokens:            fakedbTokens,
		uploads:           fakedbUploads,
		files:             fakedbFiles,
		anonymousUploads:  fakedbAnonUploads,
		downloadedUploads: fakedbDownloadedUploads,
		downloads:         fakedbDownloads,
	}); err != nil {
		fmt.Printf("Invalid fakedb parameters: %s\n", err)
		os.Exit(1)
	}

	log.Infof("Generating fake database at %s", fakedbOutput)
	log.Infof("Parameters: %d users, %d tokens/user, %d uploads/user, %d files/upload, %d anonymous uploads, %d downloaded uploads, up to %d downloads/upload",
		fakedbUsers, fakedbTokens, fakedbUploads, fakedbFiles, fakedbAnonUploads, fakedbDownloadedUploads, fakedbDownloads)

	downloadedUploads := fakedbDownloadedUploads
	downloads := fakedbDownloads
	if fakedbFiles == 0 && fakedbDownloads > 0 && fakedbDownloadedUploads > 0 {
		log.Warningf("fake downloads disabled because --files=0 leaves no files to receive download counters")
		downloadedUploads = 0
		downloads = 0
	}

	if err := removeExistingFakeDB(fakedbOutput); err != nil {
		fmt.Printf("Failed to remove existing database: %s\n", err)
		os.Exit(1)
	}

	cfg := &metadata.Config{
		Driver:           "sqlite3",
		ConnectionString: fakedbOutput,
	}

	backend, err := metadata.NewBackend(cfg, log)
	if err != nil {
		fmt.Printf("Failed to open database: %s\n", err)
		os.Exit(1)
	}
	defer func() { _ = backend.Shutdown() }()

	start := time.Now()
	totalUploads := fakedbUsers*fakedbUploads + fakedbAnonUploads
	downloadSeeder := newFakeDownloadSeeder(totalUploads, downloadedUploads, downloads)
	if downloadSeeder.target != fakedbDownloadedUploads {
		log.Infof("Effective fake downloaded upload target: %d (requested %d)", downloadSeeder.target, fakedbDownloadedUploads)
	}

	// Create the admin user so we can log in
	adminUser := common.NewUser(common.ProviderLocal, "admin")
	adminUser.Login = "admin"
	adminUser.Name = "Admin"
	adminUser.Email = "admin@plik.root.gg"
	adminUser.IsAdmin = true
	hash, err := common.HashPassword("plikplik")
	if err != nil {
		fmt.Printf("Failed to hash admin password: %s\n", err)
		os.Exit(1)
	}
	adminUser.Password = hash
	err = backend.CreateUser(adminUser)
	if err != nil {
		fmt.Printf("Failed to create admin user: %s\n", err)
		os.Exit(1)
	}
	log.Infof("Created admin user (login: admin, password: plikplik)")

	fakeUploadPasswordHash, err := common.HashUploadPassword("plik", "plikplik")
	if err != nil {
		fmt.Printf("Failed to hash fake upload password: %s\n", err)
		os.Exit(1)
	}

	// Attribute a handful of the fake upload stream to the admin user, so the
	// demo login's own Home dashboard (activity, trending, uploads list) shows
	// real data instead of the empty/zero state a fresh account would
	// otherwise see. Reuses the exact same generators as the rest of the fake
	// upload stream (lifecycle, features, files) and the exact same
	// download/rollup/bytes pipeline — only the per-upload download seeding
	// is forced (forceSeedUpload) rather than left to the random per-upload
	// draw newFakeDownloadSeeder.seedUpload otherwise makes, since a demo
	// dashboard should not depend on chance.
	for u := range fakedbAdminUploads {
		upload := common.NewUpload()
		upload.User = adminUser.ID
		comment := fmt.Sprintf("upload %d by admin", u)
		upload.RemoteIP = fmt.Sprintf("10.0.0.%d", u+1)
		applyFakeUploadLifecycle(upload, time.Now().Add(-30*24*time.Hour), time.Now())
		applyFakeUploadFeatures(upload, comment, fakeUploadPasswordHash)
		addFakeUploadFiles(upload, fakedbFiles, "file", 100*1024*1024)

		var fileBytes map[string]int64
		if fakedbFiles > 0 && downloadSeeder.maxDownloads > 0 {
			fileBytes = downloadSeeder.forceSeedUpload(upload)
		}
		err := backend.CreateUpload(upload)
		if err != nil {
			fmt.Printf("Failed to create admin upload: %s\n", err)
			os.Exit(1)
		}
		err = createFakeDownloadRollups(backend, upload, fileBytes)
		if err != nil {
			fmt.Printf("Failed to create fake download stats for admin upload %s: %s\n", upload.ID, err)
			os.Exit(1)
		}
		err = seedFakeUploadedBytes(backend, upload)
		if err != nil {
			fmt.Printf("Failed to seed fake upload bytes for admin upload %s: %s\n", upload.ID, err)
			os.Exit(1)
		}
	}
	log.Infof("Created %d admin-owned uploads for the demo Home dashboard", fakedbAdminUploads)

	// Create randomised users
	for i := range fakedbUsers {
		first := firstNames[rand.Intn(len(firstNames))]
		last := lastNames[rand.Intn(len(lastNames))]
		provider := fakeProviders[rand.Intn(len(fakeProviders))]

		login := fmt.Sprintf("%s.%s%d", first, last, i)
		user := common.NewUser(provider, login)
		user.Login = login
		user.Name = fmt.Sprintf("%s %s", capitalize(first), capitalize(last))
		user.Email = fmt.Sprintf("%s.%s%d@%s", first, last, i, emailDomains[rand.Intn(len(emailDomains))])
		user.IsAdmin = rand.Intn(10) == 0 // ~10% admins
		user.CreatedAt = time.Now().Add(-time.Duration(rand.Intn(365*24)) * time.Hour)

		for j := range fakedbTokens {
			token := common.NewToken()
			token.Comment = fmt.Sprintf("token-%d", j)
			token.UserID = user.ID
			user.Tokens = append(user.Tokens, token)
		}

		err := backend.CreateUser(user)
		if err != nil {
			fmt.Printf("Failed to create user %s: %s\n", login, err)
			os.Exit(1)
		}

		for u := range fakedbUploads {
			upload := common.NewUpload()
			upload.User = user.ID
			if len(user.Tokens) > 0 && rand.Intn(2) == 0 {
				upload.Token = user.Tokens[rand.Intn(len(user.Tokens))].Token
			}
			comment := fmt.Sprintf("upload %d by %s", u, login)
			upload.RemoteIP = fmt.Sprintf("10.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256))
			applyFakeUploadLifecycle(upload, user.CreatedAt, time.Now())
			applyFakeUploadFeatures(upload, comment, fakeUploadPasswordHash)
			addFakeUploadFiles(upload, fakedbFiles, "file", 100*1024*1024)

			fileBytes := downloadSeeder.seedUpload(upload)
			err := backend.CreateUpload(upload)
			if err != nil {
				fmt.Printf("Failed to create upload for %s: %s\n", login, err)
				os.Exit(1)
			}
			err = createFakeDownloadRollups(backend, upload, fileBytes)
			if err != nil {
				fmt.Printf("Failed to create fake download stats for %s: %s\n", upload.ID, err)
				os.Exit(1)
			}
			err = seedFakeUploadedBytes(backend, upload)
			if err != nil {
				fmt.Printf("Failed to seed fake upload bytes for %s: %s\n", upload.ID, err)
				os.Exit(1)
			}
		}

		if (i+1)%100 == 0 {
			elapsed := time.Since(start)
			log.Infof("Created %d/%d users (%.1fs elapsed)", i+1, fakedbUsers, elapsed.Seconds())
		}
	}

	// Create anonymous uploads (no user, no token)
	for i := range fakedbAnonUploads {
		upload := common.NewUpload()
		comment := fmt.Sprintf("anonymous upload %d", i)
		upload.RemoteIP = fmt.Sprintf("192.168.%d.%d", rand.Intn(256), rand.Intn(256))
		applyFakeUploadLifecycle(upload, time.Now().Add(-30*24*time.Hour), time.Now())
		applyFakeUploadFeatures(upload, comment, fakeUploadPasswordHash)
		addFakeUploadFiles(upload, fakedbFiles, "anon_file", 50*1024*1024)
		fileBytes := downloadSeeder.seedUpload(upload)
		err := backend.CreateUpload(upload)
		if err != nil {
			fmt.Printf("Failed to create anonymous upload: %s\n", err)
			os.Exit(1)
		}
		err = createFakeDownloadRollups(backend, upload, fileBytes)
		if err != nil {
			fmt.Printf("Failed to create fake download stats for %s: %s\n", upload.ID, err)
			os.Exit(1)
		}
		err = seedFakeUploadedBytes(backend, upload)
		if err != nil {
			fmt.Printf("Failed to seed fake upload bytes for %s: %s\n", upload.ID, err)
			os.Exit(1)
		}
	}

	elapsed := time.Since(start)
	totalFiles := totalUploads * fakedbFiles
	stats, statsErr := backend.GetServerStatistics()

	fmt.Println()
	log.Infof("Done! Created %d users (+admin), %d tokens, %d uploads (%d anonymous), %d files, %d uploads with fake downloads in %.1fs",
		fakedbUsers,
		fakedbUsers*fakedbTokens,
		totalUploads,
		fakedbAnonUploads,
		totalFiles,
		downloadSeeder.selected,
		elapsed.Seconds())
	if statsErr == nil && stats.Usage != nil {
		features := stats.Usage.Current.Features
		ttl := stats.Usage.Current.TTL
		log.Infof("Fake stats: %d downloads, features: password=%d removable=%d one-shot=%d stream=%d extendTTL=%d e2ee=%d comments=%d",
			stats.Usage.Downloads.Total,
			features.PasswordUploads,
			features.RemovableUploads,
			features.OneShotUploads,
			features.StreamUploads,
			features.ExtendTTLUploads,
			features.E2EEUploads,
			features.CommentUploads)
		log.Infof("Fake TTL stats: none=%d lt1h=%d 1h1d=%d 1d7d=%d 7d30d=%d gt30d=%d",
			ttl.NoneUploads,
			ttl.LessThan1HourUploads,
			ttl.OneHourToOneDayUploads,
			ttl.OneDayToSevenDaysUploads,
			ttl.SevenDaysTo30DaysUploads,
			ttl.GreaterThan30DaysUploads)
	}

	fmt.Println()
	fmt.Println("To use this database, start plikd with:")
	fmt.Println()
	fmt.Printf("  PLIKD_METADATA_BACKEND_CONFIG='{\"Driver\":\"sqlite3\",\"ConnectionString\":\"%s\"}' \\\n", fakedbOutput)
	fmt.Println("  PLIKD_FEATURE_AUTHENTICATION=enabled \\")
	fmt.Println("  ./plikd")
	fmt.Println()
	fmt.Println("Login with:  admin / plikplik")
	fmt.Println()
}

// capitalize uppercases only the first rune so generated display names stay
// readable without corrupting non-ASCII fake identity data.
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// randomFakeUploadTTL picks one representative TTL bucket for stats/UI coverage.
func randomFakeUploadTTL() int {
	return fakeUploadTTLBuckets[rand.Intn(len(fakeUploadTTLBuckets))]
}

// applyFakeUploadLifecycle assigns a coherent creation/expiration pair. Finite
// TTL uploads are always generated as currently available, with ExpireAt exactly
// matching CreatedAt + TTL.
func applyFakeUploadLifecycle(upload *common.Upload, minCreatedAt time.Time, now time.Time) {
	applyFakeUploadLifecycleWithTTL(upload, minCreatedAt, now, randomFakeUploadTTL())
}

// applyFakeUploadLifecycleWithTTL is the deterministic test seam for lifecycle
// generation; production fakedb calls applyFakeUploadLifecycle for random TTLs.
func applyFakeUploadLifecycleWithTTL(upload *common.Upload, minCreatedAt time.Time, now time.Time, ttl int) {
	upload.TTL = ttl
	upload.CreatedAt = fakeUploadCreatedAt(minCreatedAt, now, ttl)
	if ttl == 0 {
		upload.ExpireAt = nil
		return
	}
	expireAt := upload.CreatedAt.Add(time.Duration(ttl) * time.Second)
	upload.ExpireAt = &expireAt
}

// fakeUploadCreatedAt chooses a creation time between minCreatedAt and now.
// For finite TTLs, it raises the lower bound enough to guarantee CreatedAt+TTL
// is still in the future, so generated uploads are current when fakedb finishes.
func fakeUploadCreatedAt(minCreatedAt time.Time, now time.Time, ttl int) time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	if minCreatedAt.IsZero() {
		minCreatedAt = now.Add(-30 * 24 * time.Hour)
	}
	if minCreatedAt.After(now) {
		minCreatedAt = now
	}

	lowerBound := minCreatedAt
	if ttl > 0 {
		nonExpiredLowerBound := now.Add(-time.Duration(ttl) * time.Second).Add(time.Nanosecond)
		if lowerBound.Before(nonExpiredLowerBound) {
			lowerBound = nonExpiredLowerBound
		}
	}
	if !lowerBound.Before(now) {
		return now
	}
	span := now.Sub(lowerBound)
	return lowerBound.Add(time.Duration(rand.Int63n(int64(span) + 1)))
}

// addFakeUploadFiles creates already-uploaded file metadata and aligns file
// creation with the parent upload so fake downloads cannot predate file rows.
func addFakeUploadFiles(upload *common.Upload, count int, prefix string, maxSize int64) {
	for f := range count {
		file := upload.NewFile()
		ft := fileTypes[rand.Intn(len(fileTypes))]
		file.Name = fmt.Sprintf("%s_%d%s", prefix, f, ft.ext)
		file.Size = fakeFileSize(maxSize)
		file.Status = common.FileUploaded
		file.Type = ft.contentType
		file.CreatedAt = upload.CreatedAt
	}
}

// fakeFileSize returns a non-negative size below maxSize. A non-positive bound
// is treated as zero so tests and future callers cannot panic rand.Int63n.
func fakeFileSize(maxSize int64) int64 {
	if maxSize <= 0 {
		return 0
	}
	return rand.Int63n(maxSize)
}

// applyFakeUploadFeatures gives fake uploads a broad feature mix so the admin
// stats UI exercises more than the default/comment-only path.
func applyFakeUploadFeatures(upload *common.Upload, comment string, passwordHash string) {
	if randPercent(70) {
		upload.Comments = comment
	}
	if randPercent(12) {
		upload.ProtectedByPassword = true
		upload.Login = "plik"
		upload.Password = passwordHash
	}
	if randPercent(35) {
		upload.Removable = true
	}
	if upload.TTL > 0 && randPercent(20) {
		upload.ExtendTTL = true
	}
	if randPercent(10) {
		upload.E2EE = "age"
	}
}

// randPercent returns true approximately percent times out of one hundred.
func randPercent(percent int) bool {
	return rand.Intn(100) < percent
}

// fakeDBCounts mirrors the user-facing fakedb count flags so validation can
// reject impossible inputs before any database or file generation starts.
type fakeDBCounts struct {
	users             int
	tokens            int
	uploads           int
	files             int
	anonymousUploads  int
	downloadedUploads int
	downloads         int
}

// validateFakeDBCounts rejects impossible count flags before generation starts.
func validateFakeDBCounts(counts fakeDBCounts) error {
	values := map[string]int{
		"--users":              counts.users,
		"--tokens":             counts.tokens,
		"--uploads":            counts.uploads,
		"--files":              counts.files,
		"--anon-uploads":       counts.anonymousUploads,
		"--downloaded-uploads": counts.downloadedUploads,
		"--downloads":          counts.downloads,
	}
	for flag, value := range values {
		if value < 0 {
			return fmt.Errorf("%s must be greater than or equal to 0", flag)
		}
	}
	return nil
}

// removeExistingFakeDB removes the SQLite database and sidecar files left by
// earlier runs. Missing files are fine; other remove errors stop generation so
// fakedb does not accidentally append to stale state.
func removeExistingFakeDB(path string) error {
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		err := os.Remove(candidate)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", candidate, err)
		}
	}
	return nil
}

// fakeDownloadSeeder selects a bounded number of uploads to receive fake
// downloads while iterating through generated uploads only once.
type fakeDownloadSeeder struct {
	// remainingUploads is the number of generated uploads still unseen by the
	// one-pass selector, including the upload currently being considered.
	remainingUploads int
	// target is the number of uploads that should receive fake downloads after
	// clamping the CLI value to the number of generated uploads.
	target int
	// maxDownloads is the inclusive upper bound for the random download count
	// assigned to each selected upload.
	maxDownloads int
	// selected tracks how many uploads have successfully received at least one
	// file download counter.
	selected int
}

// newFakeDownloadSeeder clamps CLI inputs so impossible targets degrade into a
// valid deterministic selection budget.
func newFakeDownloadSeeder(totalUploads int, target int, maxDownloads int) *fakeDownloadSeeder {
	if target > totalUploads {
		target = totalUploads
	}
	if target < 0 {
		target = 0
	}
	if maxDownloads < 0 {
		maxDownloads = 0
	}
	if maxDownloads == 0 {
		target = 0
	}
	return &fakeDownloadSeeder{
		remainingUploads: totalUploads,
		target:           target,
		maxDownloads:     maxDownloads,
	}
}

// seedUpload assigns lifetime download counters before the upload is
// persisted, and returns a plausible bytes-served total per file (keyed by
// file ID) for createFakeDownloadRollups to seed onto the daily rollups
// afterward. common.Upload.DownloadedBytes gets the per-upload total directly
// (mirroring DownloadCount, since it is a plain column on the upload row now),
// but bytes served has no column on common.File — that stays tracked only in
// download_stats_daily and usage_stats — so per-file bytes still cannot be
// stashed on the file structs and are threaded through as a side channel
// return value instead. Daily rollups are created separately once the
// metadata rows exist. Stream and one-shot uploads are skipped because fakedb
// does not model those lifecycles.
func (s *fakeDownloadSeeder) seedUpload(upload *common.Upload) map[string]int64 {
	if s.maxDownloads == 0 || s.remainingUploads == 0 {
		return nil
	}
	if upload.OneShot || upload.Stream {
		s.remainingUploads--
		return nil
	}

	remainingTarget := s.target - s.selected
	seed := remainingTarget > 0 && rand.Intn(s.remainingUploads) < remainingTarget
	s.remainingUploads--
	if !seed {
		return nil
	}

	return s.forceSeedUpload(upload)
}

// forceSeedUpload applies seedUpload's download/bytes generation tail
// unconditionally — a random nonzero lifetime download count (1..maxDownloads)
// and bytes-served total spread across upload.Files — bypassing seedUpload's
// probabilistic per-upload selection gate entirely. It shares every generator
// seedUpload itself uses (fakeDownloadTime, seedFakeFileDownloads) so callers
// that need a GUARANTEED-seeded upload (e.g. the admin fixture uploads below,
// which must reliably show real demo data regardless of the random draw) get
// the exact same counter/bytes coherence as the rest of the fake download
// pipeline, instead of a bespoke parallel path. Returns nil (upload/s.selected
// untouched) if seeding produced nothing — e.g. the upload has no files.
func (s *fakeDownloadSeeder) forceSeedUpload(upload *common.Upload) map[string]int64 {
	downloads := int64(rand.Intn(s.maxDownloads) + 1)
	lastDownloadedAt := fakeDownloadTime(upload.CreatedAt)

	// Keep fake detail views intuitive: the upload download counter is the sum of
	// the generated file counters shown in the same upload.
	seededDownloads, fileBytes := seedFakeFileDownloads(upload.Files, downloads, lastDownloadedAt)
	if seededDownloads == 0 {
		return nil
	}
	s.selected++
	upload.DownloadCount = seededDownloads
	upload.LastDownloadedAt = &lastDownloadedAt
	// upload.DownloadedBytes is set on the struct here, before CreateUpload
	// persists it via a plain Create — like DownloadCount, NOT via the usage
	// delta (CreateUpload deliberately does not fold DownloadedBytes into
	// usage_stats; see the comment there). createFakeDownloadRollups below
	// still separately seeds usage_stats.downloaded_bytes via
	// FixtureSeedDownloadedBytes from this same fileBytes total, so setting it
	// here too does not double it into usage_stats — it only reaches the
	// upload row.
	for _, b := range fileBytes {
		upload.DownloadedBytes += b
	}
	return fileBytes
}

// seedFakeFileDownloads spreads one upload's fake download total across its
// files while preserving the exact aggregate, and returns a plausible
// bytes-served total per file (keyed by file ID): each simulated download
// transfers a randomized fraction of the chosen file's size (see
// fakeDownloadBytes), so bytes correlate with both download counts and file
// sizes instead of always being zero.
func seedFakeFileDownloads(files []*common.File, downloads int64, lastDownloadedAt time.Time) (seeded int64, fileBytes map[string]int64) {
	if downloads <= 0 || len(files) == 0 {
		return 0, nil
	}

	fileBytes = make(map[string]int64, len(files))
	for range downloads {
		file := files[rand.Intn(len(files))]
		file.DownloadCount++
		file.LastDownloadedAt = &lastDownloadedAt
		fileBytes[file.ID] += fakeDownloadBytes(file.Size)
		seeded++
	}
	return seeded, fileBytes
}

// fakeDownloadBytes returns a plausible egress size for one simulated
// download: a randomized 50-100% fraction of the file's declared size, so
// seeded bytes vary across downloads instead of always exactly equalling
// file.Size (a full, un-ranged GET) or 0 (the pre-seeding behavior this
// replaces). A non-positive size seeds 0 bytes.
func fakeDownloadBytes(size int64) int64 {
	if size <= 0 {
		return 0
	}
	fraction := 0.5 + rand.Float64()*0.5 // [0.5, 1.0)
	return int64(float64(size) * fraction)
}

// fakeDownloadTime keeps fake downloads recent enough to populate the 30-day
// trending windows while still respecting the upload creation date.
func fakeDownloadTime(createdAt time.Time) time.Time {
	now := time.Now()
	oldest := now.Add(-30 * 24 * time.Hour)
	if createdAt.IsZero() || createdAt.After(now) {
		createdAt = now
	}
	if createdAt.Before(oldest) {
		createdAt = oldest
	}
	span := now.Sub(createdAt)
	if span <= 0 {
		return now
	}
	return createdAt.Add(time.Duration(rand.Int63n(int64(span))))
}

// createFakeDownloadRollups mirrors the seeded lifetime counters — and, via
// fileBytes, the seeded bytes served per file — into daily buckets so fakedb
// also populates windowed trending views and the chart's Traffic mode / bytes
// tiles, which otherwise read 0 for every fake download.
//
// The upload-entity rollup — the only entity type the Activity chart/window
// queries read (getActivityStatsDailySeries filters entity_type=upload) — is
// spread across a pseudo-random handful of days via spreadFakeDownloadDays
// instead of dumped entirely onto LastDownloadedAt's single day: piling every
// download onto one day produced one giant "today" bar dwarfing 29 flat days
// in the demo chart, because many fake uploads' LastDownloadedAt lands
// recently (fakeDownloadTime skews toward "now" for short-TTL uploads).
// File-entity rollups are NOT read by the chart, so they keep the simpler
// single-day shape (one row per file, on the file's own LastDownloadedAt day)
// — spreading them too would add correlation complexity for no visible payoff.
//
// fileBytes is keyed by file ID (seedFakeFileDownloads' return value) and may
// be nil, in which case every rollup simply seeds 0 bytes, matching the
// pre-bytes-seeding behavior. The upload-entity rollup's total bytes is the
// sum of every file's seeded bytes, mirroring how a real day's upload rollup
// accumulates the same day's per-file download bytes (RecordFileDownload
// attributes one download's bytes to both its file rollup and its upload
// rollup) — spreadFakeDownloadDays then splits that total proportionally
// across the days it picks, so the per-day split still sums to the exact
// lifetime total (rollup sums stay consistent with the seeded counters).
//
// Rollups are attributed to upload.User / upload.Token verbatim, exactly like
// the production recordDailyDownloads path (server/metadata/stats_download.go)
// attributes real downloads. Without this, GetUserActivityStatsDaily's
// `user_id = ?` filter never matches any fake row, so /me/stats always reports
// empty download windows for fake users even though lifetime counters exist.
func createFakeDownloadRollups(backend *metadata.Backend, upload *common.Upload, fileBytes map[string]int64) error {
	if upload.DownloadCount == 0 || upload.LastDownloadedAt == nil {
		return nil
	}

	var uploadBytes int64
	for _, bytes := range fileBytes {
		uploadBytes += bytes
	}

	for _, chunk := range spreadFakeDownloadDays(upload.ID, upload.DownloadCount, uploadBytes, upload.CreatedAt, time.Now()) {
		err := createFakeDownloadRollup(backend, common.DownloadStatsEntityUpload, upload.ID, upload.User, upload.Token, chunk.downloads, chunk.bytes, chunk.day)
		if err != nil {
			return err
		}
	}

	for _, file := range upload.Files {
		if file.DownloadCount == 0 || file.LastDownloadedAt == nil {
			continue
		}
		err := createFakeDownloadRollup(backend, common.DownloadStatsEntityFile, file.ID, upload.User, upload.Token, file.DownloadCount, fileBytes[file.ID], *file.LastDownloadedAt)
		if err != nil {
			return err
		}
	}

	return backend.FixtureSeedDownloadedBytes(upload, uploadBytes)
}

// fakeDownloadDayChunk is one UTC-day bucket produced by spreadFakeDownloadDays:
// a portion of an entity's total fake downloads/bytes attributed to that day.
type fakeDownloadDayChunk struct {
	day       time.Time
	downloads int64
	bytes     int64
}

// maxFakeDownloadSpreadDays bounds how many distinct days one upload's fake
// downloads are spread across. A handful of days (not all 30) keeps the shape
// visually uneven — a realistic-looking curve — rather than flattening the
// chart into a plateau.
const maxFakeDownloadSpreadDays = 8

// entitySeed derives a deterministic RNG seed from an entity ID so a given
// fakedb run's per-upload download spread (spreadFakeDownloadDays) is
// reproducible independent of the shared global math/rand source's call order
// elsewhere in generation.
func entitySeed(entityID string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(entityID))
	return int64(h.Sum64())
}

// truncateUTCDay returns the UTC midnight instant of the day containing t,
// matching the day key used by download_stats_daily (statsDay in
// server/metadata/stats_download.go, unexported and not reachable from this
// package).
func truncateUTCDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// spreadFakeDownloadDays deterministically distributes one entity's total fake
// downloads (and the proportional bytes) across a pseudo-random handful of
// distinct UTC days, instead of attributing the entire lifetime total to a
// single day (see createFakeDownloadRollups for why that used to skew the
// demo Activity chart). downloads <= 0 returns nil (nothing to spread).
//
// The spread window is [max(createdAt's day, today - 29), today] — anchored to
// `now` (the chart/window's actual "today"), NOT to the entity's own
// LastDownloadedAt day. An earlier draft anchored to LastDownloadedAt and
// always included its day; that still let "today" dominate the aggregate
// chart, because fakeDownloadTime already skews many entities' own
// LastDownloadedAt toward "now" (short-TTL uploads must have been created
// recently to still be current, which also compresses their possible download
// span toward today) — anchoring every entity's spread to its own recent day
// and forcing that day's inclusion piled all of those entities back onto the
// same calendar day, just spread thinner. Anchoring to a day common to every
// entity and picking freely within it (no forced day) decorrelates entities'
// choices instead, so only entities whose window happens to reach a given day
// contribute to it — the fix that actually flattens the aggregate.
//
// The split is seeded from entityID (entitySeed), not the shared global rand
// source, so re-running fakedb with the same generated IDs reproduces the same
// shape. Every returned day gets at least 1 download — a fractional download
// would misrepresent the exact lifetime total the rollups must keep summing
// to — and the chunks' downloads/bytes always sum to exactly the inputs.
func spreadFakeDownloadDays(entityID string, downloads int64, bytes int64, createdAt time.Time, now time.Time) []fakeDownloadDayChunk {
	if downloads <= 0 {
		return nil
	}

	today := truncateUTCDay(now)
	earliest := today.AddDate(0, 0, -29)
	if createdDay := truncateUTCDay(createdAt); createdDay.After(earliest) {
		earliest = createdDay
	}
	if earliest.After(today) {
		earliest = today
	}
	spanDays := int(today.Sub(earliest).Hours()/24) + 1

	maxDays := min(spanDays, maxFakeDownloadSpreadDays)
	if int64(maxDays) > downloads {
		maxDays = int(downloads)
	}
	if maxDays < 1 {
		maxDays = 1
	}

	rng := rand.New(rand.NewSource(entitySeed(entityID)))
	numDays := 1
	if maxDays > 1 {
		numDays = 1 + rng.Intn(maxDays)
	}

	// Pick numDays distinct day offsets from today in [0, spanDays). No day is
	// forced — see the decorrelation rationale above.
	offsets := map[int]bool{}
	for len(offsets) < numDays {
		offsets[rng.Intn(spanDays)] = true
	}
	sortedOffsets := make([]int, 0, len(offsets))
	for o := range offsets {
		sortedOffsets = append(sortedOffsets, o)
	}
	sort.Ints(sortedOffsets)

	n := len(sortedOffsets)

	// Random positive weights, independent of day position, so no single day —
	// in particular offset 0 (the entity's own most recent day) — is
	// systematically favored with the largest share: an earlier draft consumed
	// a wide random chunk of the remaining total starting from offset 0, which
	// still left "today" dominating the aggregate chart whenever many
	// entities' own most-recent day happened to be the same calendar day.
	// Independent weights avoid that positional bias.
	downloadAlloc := make([]int64, n)
	if n == 1 {
		downloadAlloc[0] = downloads
	} else {
		weights := make([]float64, n)
		var weightSum float64
		for i := range weights {
			weights[i] = 0.3 + rng.Float64()
			weightSum += weights[i]
		}
		var allocated int64
		for i, w := range weights {
			share := max(int64(float64(downloads)*w/weightSum), 1)
			downloadAlloc[i] = share
			allocated += share
		}
		// Correct rounding drift by nudging randomly chosen days (never always
		// the same slot) so the total lands on exactly `downloads`; every
		// day's floor of 1 is preserved (numDays <= downloads guarantees a
		// valid all-decremented-to-1 state exists, so this always terminates).
		diff := downloads - allocated
		for diff != 0 {
			idx := rng.Intn(n)
			if diff > 0 {
				downloadAlloc[idx]++
				diff--
			} else if downloadAlloc[idx] > 1 {
				downloadAlloc[idx]--
				diff++
			}
		}
	}

	// Bytes follow each day's share of downloads (more downloads that day ->
	// proportionally more bytes). int64() truncates toward zero on these
	// non-negative floats, so the per-day floors always sum to <= bytes —
	// diff is never negative, and the remainder is handed to randomly chosen
	// days (again, no positional bias) until the total is exact.
	byteAlloc := make([]int64, n)
	var allocatedBytes int64
	for i, d := range downloadAlloc {
		b := int64(float64(bytes) * float64(d) / float64(downloads))
		byteAlloc[i] = b
		allocatedBytes += b
	}
	for diff := bytes - allocatedBytes; diff > 0; diff-- {
		byteAlloc[rng.Intn(n)]++
	}

	chunks := make([]fakeDownloadDayChunk, n)
	for i, offset := range sortedOffsets {
		chunks[i] = fakeDownloadDayChunk{
			day:       today.AddDate(0, 0, -offset),
			downloads: downloadAlloc[i],
			bytes:     byteAlloc[i],
		}
	}
	return chunks
}

// seedFakeUploadedBytes mirrors the wire-byte ingress of a fake upload into the
// upload's creation-day rollup bucket and the usage_stats.uploaded_bytes
// counters, so the Uploads / Uploaded-data metrics show data on dev instances.
// CreateUpload already recorded the day's Uploads +1 (server/metadata/upload.go);
// this adds the matching bytes (the sum of the upload's completed file sizes,
// which for a fully received transfer equals the wire bytes) via the
// fixture-only FixtureSeedUploadedBytes seam. It is a no-op for a byte-less
// upload.
func seedFakeUploadedBytes(backend *metadata.Backend, upload *common.Upload) error {
	var uploadBytes int64
	for _, file := range upload.Files {
		switch file.Status {
		case common.FileUploaded, common.FileRemoved, common.FileDeleted:
			uploadBytes += file.Size
		}
	}
	return backend.FixtureSeedUploadedBytes(upload, uploadBytes)
}

// createFakeDownloadRollup writes one UTC-day bucket matching the production
// download_stats_daily identity: day, entity type, and entity id. userID and
// token are copied verbatim from the owning upload (empty strings for
// anonymous uploads), matching how recordDailyDownloads attributes real rows.
// day is truncated to its UTC midnight instant here too, so callers may pass
// either a raw timestamp or an already-truncated day (spreadFakeDownloadDays
// returns the latter; the idempotent truncation is a no-op for it).
func createFakeDownloadRollup(backend *metadata.Backend, entityType string, entityID string, userID string, token string, downloads int64, bytes int64, day time.Time) error {
	day = truncateUTCDay(day)
	return backend.CreateDownloadStatsDaily(&common.DownloadStatsDaily{
		Day:        day,
		EntityType: entityType,
		EntityID:   entityID,
		UserID:     userID,
		Token:      token,
		Downloads:  downloads,
		Bytes:      bytes,
		UpdatedAt:  time.Now(),
	})
}
