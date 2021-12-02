package metadata

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/root-gg/logger"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/root-gg/plik/server/common"
)

// Default config
var metadataBackendConfig = &Config{Driver: "sqlite3", ConnectionString: "/tmp/plik.test.db", EraseFirst: true, Debug: false}

func TestMain(m *testing.M) {

	// Override config from env
	testConfigPath := os.Getenv("PLIKD_CONFIG")
	if testConfigPath != "" {
		fmt.Println("loading test config : " + testConfigPath)
		testConfig, err := common.LoadConfiguration(testConfigPath)
		if err != nil {
			fmt.Printf("Unable to load test configuration : %s\n", err)
			os.Exit(1)
		}
		metadataBackendConfig = NewConfig(testConfig.MetadataBackendConfig)
		metadataBackendConfig.EraseFirst = true
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func newTestMetadataBackend() *Backend {
	b, err := NewBackend(metadataBackendConfig, logger.NewLogger())
	if err != nil {
		panic(fmt.Sprintf("unable to create metadata backend : %s", err))
	}

	return b
}

func shutdownTestMetadataBackend(b *Backend) {
	err := b.Shutdown()
	if err != nil {
		fmt.Printf("Unable to shutdown metadata backend : %s\n", err)
	}
}

func TestNewConfig(t *testing.T) {
	params := make(map[string]interface{})
	params["Driver"] = "driver"
	params["ConnectionString"] = "connection string"
	params["EraseFirst"] = true

	config := NewConfig(params)
	require.Equal(t, "driver", config.Driver, "invalid driver")
	require.Equal(t, "connection string", config.ConnectionString, "invalid connection string")
	require.True(t, config.EraseFirst, "invalid erase first")
}

func TestMetadata(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	uploadID := "azertiop"
	upload := &common.Upload{ID: uploadID}

	err := b.db.Create(upload).Error
	require.NoError(t, err, "unable to create upload")

	file := &common.File{ID: "1234567890", UploadID: uploadID}
	upload.Files = append(upload.Files, file)

	err = b.db.Save(&upload).Error
	require.NoError(t, err, "unable to update upload")

	upload = &common.Upload{}
	err = b.db.Preload("Files").Take(upload, "id = ?", uploadID).Error
	require.NoError(t, err, "unable to fetch upload")

	err = b.Shutdown()
	require.NoError(t, err, "close db error")
}

func TestMetadataInvalidBackend(t *testing.T) {
	metadataBackendConfig := &Config{Driver: "invalid"}
	b, err := NewBackend(metadataBackendConfig, logger.NewLogger())
	require.Error(t, err)
	require.Nil(t, b)
}

func TestMetadataBackendDebug(t *testing.T) {
	debugMetadataBackendConfig := *metadataBackendConfig
	debugMetadataBackendConfig.Debug = true
	b, err := NewBackend(&debugMetadataBackendConfig, logger.NewLogger())
	require.NoError(t, err)
	require.NotNil(t, b)
	_ = b.Shutdown()
}

func TestMetadataSlowQueryThreshold(t *testing.T) {
	slowQueryThresholdMetadataBackendConfig := *metadataBackendConfig
	slowQueryThresholdMetadataBackendConfig.SlowQueryThreshold = "60s"
	b, err := NewBackend(&slowQueryThresholdMetadataBackendConfig, logger.NewLogger())
	require.NoError(t, err)
	require.NotNil(t, b)
}

func TestMetadataInvalidSlowQueryThreshold(t *testing.T) {
	slowQueryThresholdMetadataBackendConfig := *metadataBackendConfig
	slowQueryThresholdMetadataBackendConfig.SlowQueryThreshold = "blah"
	b, err := NewBackend(&slowQueryThresholdMetadataBackendConfig, logger.NewLogger())
	require.Error(t, err)
	require.Nil(t, b)
}

func TestMetadataInvalidConnectionString(t *testing.T) {
	metadataBackendConfig := &Config{Driver: "mysql", ConnectionString: "!fo{o}b@r"}
	b, err := NewBackend(metadataBackendConfig, logger.NewLogger())
	require.Error(t, err)
	require.Nil(t, b)

	metadataBackendConfig = &Config{Driver: "postgres", ConnectionString: "!fo{o}b@r"}
	b, err = NewBackend(metadataBackendConfig, logger.NewLogger())
	require.Error(t, err)
	require.Nil(t, b)
}

func TestConnectionPoolParams(t *testing.T) {
	metadataBackendConfig := *metadataBackendConfig
	metadataBackendConfig.MaxIdleConns = 10
	metadataBackendConfig.MaxOpenConns = 50
	b, err := NewBackend(&metadataBackendConfig, logger.NewLogger())
	require.NoError(t, err)
	require.NotNil(t, b)
	_ = b.Shutdown()
}

func TestGormConcurrent(t *testing.T) {
	type Object struct {
		gorm.Model
		Foo string
	}

	// https://github.com/jinzhu/gorm/issues/2875
	db, err := gorm.Open(sqlite.Open("/tmp/plik.db"))
	require.NoError(t, err, "DB open error")

	err = db.AutoMigrate(&Object{})
	require.NoError(t, err, "schema update error")

	count := 30
	var wg sync.WaitGroup
	errors := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			errors <- db.Create(&Object{Foo: fmt.Sprintf("%d", i)}).Error
		}(i)
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err, "unexpected error")
	}
}

func TestMetadataConcurrent(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	uploadID := "azertiop"
	upload := &common.Upload{ID: uploadID}

	err := b.db.Create(upload).Error
	require.NoError(t, err, "unable to create upload")

	count := 30
	var wg sync.WaitGroup
	errors := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			errors <- b.db.Create(&common.File{ID: fmt.Sprintf("file_%d", i), UploadID: uploadID}).Error
		}(i)
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err, "unexpected error")
	}

	upload = &common.Upload{}
	err = b.db.Preload("Files").Take(upload, "id = ?", uploadID).Error
	require.NoError(t, err, "unable to fetch upload")
}

func TestMetadataUpdateFileStatus(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	uploadID := "azertiop"
	upload := &common.Upload{ID: uploadID}

	err := b.db.Create(upload).Error
	require.NoError(t, err, "unable to create upload")

	file := &common.File{ID: "1234567890", UploadID: uploadID, Status: common.FileMissing}
	upload.Files = append(upload.Files, file)

	err = b.db.Save(&upload).Error
	require.NoError(t, err, "unable to update upload")

	file.Status = common.FileUploaded
	result := b.db.Where(&common.File{Status: common.FileUploading}).Save(&file)
	require.Error(t, result.Error, "able to update missing file")
	require.Equal(t, int64(0), result.RowsAffected, "unexpected update")

	//!\\ ON MYSQL SAVE MODIFIES THE FILE STATUS BACK TO MISSING ( wtf ? ) //!\\
	file.Status = common.FileUploaded

	result = b.db.Where(&common.File{Status: common.FileMissing}).Save(&file)
	require.NoError(t, result.Error, "unable to update missing file")
	require.Equal(t, int64(1), result.RowsAffected, "unexpected update")

	upload = &common.Upload{}
	err = b.db.Preload("Files").Take(upload, "id = ?", uploadID).Error
	require.NoError(t, err, "unable to fetch upload")
}

func TestMetadataNotFound(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	err := b.db.Where(&common.Upload{ID: "notfound"}).Take(upload).Error
	require.Error(t, err, "unable to fetch upload")
	require.Equal(t, gorm.ErrRecordNotFound, err, "unexpected error type")
}

func TestMetadataCursor(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	var expected = []string{"upload 1", "upload 2", "upload 3"}
	for _, id := range expected {
		err := b.db.Create(&common.Upload{ID: id}).Error
		require.NoError(t, err, "unable to create upload")
	}

	rows, err := b.db.Model(&common.Upload{}).Rows()
	require.NoError(t, err, "unable to fetch uploads")

	var ids []string
	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		require.NoError(t, err, "unable to read row")
		ids = append(ids, upload.ID)
	}

	require.Equal(t, expected, ids, "mismatch")
}

func TestMetadataExpiredCursor(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	err := b.db.Create(&common.Upload{ID: "upload 1"}).Error
	require.NoError(t, err, "unable to create upload")

	expire := time.Now().Add(time.Hour)
	err = b.db.Create(&common.Upload{ID: "upload 2", ExpireAt: &expire}).Error
	require.NoError(t, err, "unable to create upload")

	expire2 := time.Now().Add(-time.Hour)
	err = b.db.Create(&common.Upload{ID: "upload 3", ExpireAt: &expire2}).Error
	require.NoError(t, err, "unable to create upload")

	rows, err := b.db.Model(&common.Upload{}).Where("expire_at < ?", time.Now()).Rows()
	require.NoError(t, err, "unable to fetch uploads")

	var ids []string
	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		require.NoError(t, err, "unable to read row")
		ids = append(ids, upload.ID)
	}

	require.Equal(t, []string{"upload 3"}, ids, "mismatch")
}

// https://github.com/mattn/go-sqlite3/issues/569
func TestMetadataCursorLock(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	var expected = []string{"upload 1", "upload 2", "upload 3"}
	for _, id := range expected {
		err := b.db.Create(&common.Upload{ID: id}).Error
		require.NoError(t, err, "unable to create upload")
	}

	rows, err := b.db.Model(&common.Upload{}).Rows()
	require.NoError(t, err, "unable to select uploads")

	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		require.NoError(t, err, "unable to read row")

		upload.Comments = "lol"
		err = b.db.Save(upload).Error
		require.NoError(t, err, "unable to save upload")
	}
}

func TestUnscoped(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	uploadID := "azertiop"
	upload := &common.Upload{ID: uploadID}

	err := b.db.Create(upload).Error
	require.NoError(t, err, "unable to create upload")

	var count int64
	err = b.db.Model(&common.Upload{}).Unscoped().Where("deleted_at IS NOT NULL").Count(&count).Error
	require.NoError(t, err, "get deleted upload error")
	require.Equal(t, int64(0), count, "get deleted upload count error")

	err = b.db.Delete(&upload).Error
	require.NoError(t, err, "unable to delete upload")

	upload = &common.Upload{}
	err = b.db.Take(upload, &common.Upload{ID: uploadID}).Error
	require.Equal(t, gorm.ErrRecordNotFound, err, "get upload error")

	upload = &common.Upload{}
	err = b.db.Unscoped().Take(upload, &common.Upload{ID: uploadID}).Error
	require.NoError(t, err, "get upload error")
	require.NotNil(t, upload, "get upload nil")

	err = b.db.Model(&common.Upload{}).Unscoped().Where("deleted_at IS NOT NULL").Count(&count).Error
	require.NoError(t, err, "get deleted upload error")
	require.Equal(t, int64(1), count, "get deleted upload count error")
}

func TestDoubleDelete(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	uploadID := "azertiop"
	upload := &common.Upload{ID: uploadID}

	err := b.db.Create(upload).Error
	require.NoError(t, err, "unable to create upload")

	err = b.db.Delete(&upload).Error
	require.NoError(t, err, "unable to delete upload")

	err = b.db.Delete(&upload).Error
	require.NoError(t, err, "unable to delete upload")
}
