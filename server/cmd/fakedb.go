package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/root-gg/logger"
	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadata"
)

var fakedbUsers int
var fakedbTokens int
var fakedbUploads int
var fakedbFiles int
var fakedbAnonUploads int
var fakedbOutput string

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
	fakedbCmd.Flags().StringVar(&fakedbOutput, "output", "/tmp/test-plik.db", "output SQLite database path")
}

var firstNames = []string{
	"alice", "bob", "charlie", "diana", "eve", "frank", "grace", "hector",
	"iris", "jack", "kate", "leo", "mia", "noah", "olivia", "paul",
	"quinn", "rosa", "sam", "tina", "uma", "victor", "wendy", "xander",
	"yara", "zach", "ada", "ben", "clara", "dario", "elsa", "felix",
	"gina", "hugo", "ivy", "jake", "lana", "max", "nora", "oscar",
}

var lastNames = []string{
	"smith", "jones", "brown", "wilson", "taylor", "thomas", "white",
	"harris", "martin", "garcia", "martinez", "robinson", "clark", "lewis",
	"lee", "walker", "hall", "allen", "young", "king", "wright", "scott",
	"green", "baker", "adams", "nelson", "hill", "moore", "jackson", "davis",
}

var emailDomains = []string{
	"gmail.com", "outlook.com", "yahoo.com", "proton.me", "example.org",
	"acme.co", "corp.net", "dev.io", "company.com", "test.local",
}

var fakeProviders = []string{"local", "google", "github", "ovh", "oidc"}

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

func fakedb(cmd *cobra.Command, args []string) {
	log := logger.NewLogger().SetMinLevel(logger.INFO)
	log.Infof("Generating fake database at %s", fakedbOutput)
	log.Infof("Parameters: %d users, %d tokens/user, %d uploads/user, %d files/upload, %d anonymous uploads",
		fakedbUsers, fakedbTokens, fakedbUploads, fakedbFiles, fakedbAnonUploads)

	// Remove existing file if present
	_ = os.Remove(fakedbOutput)

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
			upload := common.NewUpload(false, 16)
			upload.User = user.ID
			if len(user.Tokens) > 0 && rand.Intn(2) == 0 {
				upload.Token = user.Tokens[rand.Intn(len(user.Tokens))].Token
			}
			upload.Comments = fmt.Sprintf("upload %d by %s", u, login)
			upload.RemoteIP = fmt.Sprintf("10.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256))

			ttl := []int{0, 3600, 86400, 604800, 2592000}[rand.Intn(5)]
			upload.TTL = ttl
			if ttl > 0 {
				// Spread expiry randomly into the future (1h to 60 days)
				exp := time.Now().Add(time.Duration(rand.Intn(60*24)+1) * time.Hour)
				upload.ExpireAt = &exp
			}
			upload.CreatedAt = user.CreatedAt.Add(time.Duration(rand.Intn(30*24)) * time.Hour)

			for f := range fakedbFiles {
				file := upload.NewFile()
				ft := fileTypes[rand.Intn(len(fileTypes))]
				file.Name = fmt.Sprintf("file_%d%s", f, ft.ext)
				file.Size = int64(rand.Intn(100 * 1024 * 1024))
				file.Status = common.FileUploaded
				file.Type = ft.contentType
			}

			err := backend.CreateUpload(upload)
			if err != nil {
				fmt.Printf("Failed to create upload for %s: %s\n", login, err)
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
		upload := common.NewUpload(false, 16)
		upload.Comments = fmt.Sprintf("anonymous upload %d", i)
		upload.RemoteIP = fmt.Sprintf("192.168.%d.%d", rand.Intn(256), rand.Intn(256))
		ttl := []int{0, 3600, 86400, 604800}[rand.Intn(4)]
		upload.TTL = ttl
		if ttl > 0 {
			// Spread expiry randomly into the future (1h to 60 days)
			exp := time.Now().Add(time.Duration(rand.Intn(60*24)+1) * time.Hour)
			upload.ExpireAt = &exp
		}
		upload.CreatedAt = time.Now().Add(-time.Duration(rand.Intn(30*24)) * time.Hour)
		for f := range fakedbFiles {
			file := upload.NewFile()
			ft := fileTypes[rand.Intn(len(fileTypes))]
			file.Name = fmt.Sprintf("anon_file_%d%s", f, ft.ext)
			file.Size = int64(rand.Intn(50 * 1024 * 1024))
			file.Status = common.FileUploaded
			file.Type = ft.contentType
		}
		err := backend.CreateUpload(upload)
		if err != nil {
			fmt.Printf("Failed to create anonymous upload: %s\n", err)
			os.Exit(1)
		}
	}

	elapsed := time.Since(start)
	totalUploads := fakedbUsers*fakedbUploads + fakedbAnonUploads
	totalFiles := totalUploads * fakedbFiles

	fmt.Println()
	log.Infof("Done! Created %d users (+admin), %d tokens, %d uploads (%d anonymous), %d files in %.1fs",
		fakedbUsers,
		fakedbUsers*fakedbTokens,
		totalUploads,
		fakedbAnonUploads,
		totalFiles,
		elapsed.Seconds())

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

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
