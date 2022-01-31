package metadata

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func getTestBackend() string {
	backend := os.Getenv("BACKEND")
	if backend == "" {
		return "sqlite3"
	}
	return backend
}

func generateTestData(t *testing.T, b *Backend) {
	setting := &common.Setting{Key: "key1", Value: "val1"}
	err := b.CreateSetting(setting)
	require.NoError(t, err, "unable to create setting")

	admin := common.NewUser(common.ProviderLocal, "admin")
	admin.IsAdmin = true
	admin.Login = "admin"
	admin.Password = "$2a$14$s103BdAMxYV96BunH9hefOEpXnmMzHBmif6tcsQHZkioFeoeHiuRu" // p@ssw0rd
	admin.Name = "Plik Admin"
	admin.Email = "admin@root.gg"
	admin.MaxTTL = 86400 * 365
	admin.MaxFileSize = 100 * 1e9
	err = b.CreateUser(admin)
	require.NoError(t, err, "unable to create admin user")

	adminToken := admin.NewToken()
	adminToken.Token = "e78415ed-883e-4d0b-5d0e-fe2d03757520"
	adminToken.Comment = "admin token"
	err = b.CreateToken(adminToken)
	require.NoError(t, err, "unable to create admin token")

	user := common.NewUser(common.ProviderGoogle, "googleuser")
	user.Login = "user@root.gg"
	user.Email = "user@root.gg"
	user.Name = "Plik User"
	err = b.CreateUser(user)
	require.NoError(t, err, "unable to create admin user")

	userToken := user.NewToken()
	userToken.Token = "8cbaeacd-6a3e-4636-4200-607a6e240688"
	userToken.Comment = "user token"
	err = b.CreateToken(userToken)
	require.NoError(t, err, "unable to create admin token")

	// Anonymous Upload
	upload := &common.Upload{}
	upload.ID = "UPLOAD1XXXXXXXXX"
	upload.UploadToken = "UPLOADTOKENXXXXXXXXXXXXXXXXXXXXX"
	upload.RemoteIP = "1.3.3.7"
	upload.DownloadDomain = "https://download.domain"
	upload.IsAdmin = true
	upload.OneShot = true
	upload.Removable = true
	upload.Comments = "愛 الحب 사랑 αγάπη любовь प्यार Սեր माया"
	upload.Login = "foo"
	upload.Password = "bar"
	upload.TTL = 3600
	upload.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	deadline := time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC)
	upload.ExpireAt = &deadline

	file := upload.NewFile()
	file.ID = "FILE1XXXXXXXXXXX"
	file.Size = 42
	file.Md5 = "ccea80b85af4f156af9d4d3b94e91a5e"
	file.Name = "愛愛愛"
	file.BackendDetails = "{foo:\"bar\"}"
	file.Reference = "1"
	file.Type = "application/awesome"
	file.Status = common.FileUploaded

	err = b.CreateUpload(upload)
	require.NoError(t, err, "unable to save upload metadata")

	// User Upload
	upload2 := &common.Upload{}
	upload2.ID = "UPLOAD2XXXXXXXXX"
	upload2.UploadToken = "UPLOADTOKENXXXXXXXXXXXXXXXXXXXXX"
	upload2.User = user.ID
	upload.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	file2 := upload2.NewFile()
	file2.ID = "FILE2XXXXXXXXXXX"
	file2.Name = "filename"

	err = b.CreateUpload(upload2)
	require.NoError(t, err, "unable to save upload metadata")

	// User Token Upload
	upload3 := &common.Upload{}
	upload3.ID = "UPLOAD3XXXXXXXXX"
	upload3.UploadToken = "UPLOADTOKENXXXXXXXXXXXXXXXXXXXXX"
	upload3.User = user.ID
	upload3.Token = userToken.Token
	file3 := upload3.NewFile()
	file3.ID = "FILE3XXXXXXXXXXX"
	file3.Name = "filename"

	err = b.CreateUpload(upload3)
	require.NoError(t, err, "unable to save upload metadata")

	// Deleted upload
	upload4 := &common.Upload{}
	upload4.ID = "UPLOAD4XXXXXXXXX"
	upload4.UploadToken = "UPLOADTOKENXXXXXXXXXXXXXXXXXXXXX"
	upload.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC)
	upload.DeletedAt = gorm.DeletedAt{Time: deletedAt, Valid: true}
	file4 := upload2.NewFile()
	file4.ID = "FILE4XXXXXXXXXXX"
	file4.Name = "filename"

	err = b.CreateUpload(upload4)
	require.NoError(t, err, "unable to save upload metadata")
}

func loadSQLDump(t *testing.T, path string) {
	fmt.Printf("Loading SQL dump %s\n", path)

	testConfig := &Config{}
	*testConfig = *metadataBackendConfig
	testConfig.EraseFirst = true
	testConfig.noMigrations = true
	testConfig.Debug = false
	b, err := NewBackend(testConfig, logger.NewLogger())
	require.NoError(t, err, "unable to create metadata backend")
	defer shutdownTestMetadataBackend(b)

	sqlDB, err := b.db.DB()
	require.NoError(t, err, "unable to get db handle")

	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	sqldump, err := ioutil.ReadAll(f)
	require.NoError(t, err, "unable to get read sqldump")
	_, err = sqlDB.Exec(string(sqldump))
	require.NoError(t, err, "unable to get load sqldump")
}

func TestGenerateExport(t *testing.T) {
	b := newTestMetadataBackend()

	migrations := b.getMigrations()
	lastMigrationName := migrations[len(migrations)-1].ID
	generateTestData(t, b)

	path := "dumps/export/" + lastMigrationName + ".dump"

	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		return
	}

	require.True(t, os.IsNotExist(err), "Can stat %s : %S", path, err)

	fmt.Printf("Missing metadata export dump %s\n", path)

	err = b.Export(path)
	require.NoError(t, err, "unable to export metadata")
}

func TestLoadExports(t *testing.T) {
	exportDirectory := "dumps/export"
	_, err := os.Stat(exportDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
	}

	files, err := ioutil.ReadDir(exportDirectory)
	require.NoError(t, err, "unable to list metadata exports")

	for _, file := range files {
		if file.Name() == "gorm-v1-1.3.2.dump" {
			// gob: wrong type (gorm.DeletedAt) for received field Upload.DeletedAt
			// The field Upload.DeletedAt type was changed from *time.Time to gorm.DeletedAt when migrating to GormV2
			// This was a mistakes, it's impossible to load GormV1 (<1.3.3) metadata dumps without adding some complex logic
			// For a working fix see : https://github.com/root-gg/plik/pull/414/commits/d71302a2515624bc68cfa6311ffb1afaf37f2541
			continue
		}

		b := newTestMetadataBackend()
		path := exportDirectory + "/" + file.Name()
		fmt.Printf("Importing metadata dump %s\n", path)

		err = b.Import(path, &ImportOptions{})
		require.NoError(t, err, "unable to load metadata export")
	}
}

// Generate a PGSql dump from the testing/test-backends.sh docker image
func PgSQLDump(metadataBackendConfig *Config) (dump string, err error) {
	fmt.Println("Generate postgresql dump")

	// postgres://postgres:password@localhost:2602/postgres?sslmode=disable
	u, err := url.Parse(metadataBackendConfig.ConnectionString)
	if err != nil {
		return "", err
	}

	docker := "plik.postgres"

	var cmd *exec.Cmd
	if docker != "" {
		cmd = exec.Command("docker", "exec", docker, "pg_dump")
	} else {
		cmd = exec.Command("pg_dump")
	}

	if u.User != nil {
		if u.User.Username() != "" {
			cmd.Args = append(cmd.Args, "--username", u.User.Username())
		}

		if pass, ok := u.User.Password(); ok {
			cmd.Env = append(cmd.Env, "PGPASSWORD="+pass)
		}
	}

	if docker == "" {
		cmd.Args = append(cmd.Args, "--host", u.Hostname())

		if u.Port() != "" {
			cmd.Args = append(cmd.Args, "--port", u.Port())
		}
	}

	cmd.Args = append(cmd.Args, "--dbname", strings.TrimPrefix(u.Path, "/"))
	cmd.Args = append(cmd.Args, "--inserts")

	fmt.Println(cmd.String())

	out, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			fmt.Println(string(e.Stderr))
		}
		return "", err
	}

	return string(out), nil
}

// Generate a MySQL dump from the testing/test-backends.sh docker image
func MySQLDump(metadataBackendConfig *Config) (dump string, err error) {
	fmt.Println("Generate mysql dump")

	// plik:password@tcp(127.0.0.1:2606)/plik?charset=utf8mb4&parseTime=True&loc=Local
	var re = regexp.MustCompile(`tcp\((.*)\)`)
	connectionString := re.ReplaceAllString(metadataBackendConfig.ConnectionString, "$1")
	if !strings.Contains(connectionString, "://") {
		connectionString = "mysql://" + connectionString
	}

	u, err := url.Parse(connectionString)
	if err != nil {
		return "", err
	}

	docker := "plik." + getTestBackend()

	var cmd *exec.Cmd
	if docker != "" {
		cmd = exec.Command("docker", "exec", docker, "mysqldump")
	} else {
		cmd = exec.Command("mysqldump")
	}

	if u.User != nil {
		if u.User.Username() != "" {
			cmd.Args = append(cmd.Args, "-u", u.User.Username())
		}

		if pass, ok := u.User.Password(); ok {
			cmd.Args = append(cmd.Args, fmt.Sprintf("--password=%s", pass))
		}
	}

	if docker == "" {
		cmd.Args = append(cmd.Args, "-h", u.Hostname())

		if u.Port() != "" {
			cmd.Args = append(cmd.Args, "-P", u.Port())
		}
	}

	cmd.Args = append(cmd.Args, strings.TrimPrefix(u.Path, "/"))
	fmt.Println(cmd.String())

	out, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			fmt.Println(string(e.Stderr))
		}
		return "", err
	}

	return string(out), nil
}

// Generate a SQLite3 dump, requires sqlite3 client installed
func Sqlite3Dump(metadataBackendConfig *Config) (dump string, err error) {
	fmt.Println("Generate sqlite3 dump")

	cmd := exec.Command("sqlite3", metadataBackendConfig.ConnectionString, ".dump")
	fmt.Println(cmd.String())

	out, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			fmt.Println(string(e.Stderr))
		}
		return "", err
	}

	return string(out), nil
}

func TestGenerateSQLDump(t *testing.T) {
	b := newTestMetadataBackend()

	migrations := b.getMigrations()
	lastMigrationName := migrations[len(migrations)-1].ID
	generateTestData(t, b)

	backend := getTestBackend()
	path := "dumps/" + backend + "/" + lastMigrationName + "." + backend + ".dump"

	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		return
	}

	require.True(t, os.IsNotExist(err), "Can stat %s : %S", path, err)

	fmt.Printf("Missing SQL dump %s\n", path)

	var dump string
	switch backend {
	case "sqlite3":
		dump, err = Sqlite3Dump(metadataBackendConfig)
		require.NoError(t, err, "unable to generate sqlite3 dump")
	case "postgres":
		dump, err = PgSQLDump(metadataBackendConfig)
		require.NoError(t, err, "unable to generate pgsql dump")
	case "mariadb", "mysql":
		dump, err = MySQLDump(metadataBackendConfig)
		require.NoError(t, err, "unable to generate mysql dump")
	}

	if len(dump) > 0 {
		fmt.Print(dump)
		fmt.Printf("Saving sql dump to %s\n", path)

		err = os.MkdirAll("dumps/"+backend, os.ModePerm)
		require.NoError(t, err, "unable to create sql dump directory")

		f, err := os.Create(path)
		require.NoError(t, err, "unable to create sql dump file")
		defer func() { _ = f.Close() }()

		_, err = f.WriteString(dump)
		require.NoError(t, err, "unable to write sql dump file")
	}
}

func TestAllMigrationsOnEmptyDB(t *testing.T) {
	testConfig := &Config{}
	*testConfig = *metadataBackendConfig
	testConfig.disableSchemaInit = true
	b, err := NewBackend(testConfig, logger.NewLogger())
	require.NoError(t, err, "unable to create metadata backend")

	// Test to generate data
	generateTestData(t, b)
	shutdownTestMetadataBackend(b)

	// Test to reopen the DB with the generated data
	*testConfig = *metadataBackendConfig
	testConfig.EraseFirst = false
	b, err = NewBackend(testConfig, logger.NewLogger())
	require.NoError(t, err, "unable to create metadata backend")
	shutdownTestMetadataBackend(b)
}

func TestMigrationsFromSQLDumps(t *testing.T) {
	sqlDumpDirectory := "dumps/" + getTestBackend()
	_, err := os.Stat(sqlDumpDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
	}

	files, err := ioutil.ReadDir(sqlDumpDirectory)
	require.NoError(t, err, "unable to list SQL dumps")

	for _, file := range files {
		loadSQLDump(t, sqlDumpDirectory+"/"+file.Name())

		testConfig := &Config{}
		*testConfig = *metadataBackendConfig
		testConfig.Debug = false
		testConfig.EraseFirst = false
		b, err := NewBackend(testConfig, logger.NewLogger())
		require.NoError(t, err, "unable to create metadata backend")
		shutdownTestMetadataBackend(b)
	}
}
