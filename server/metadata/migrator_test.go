package metadata

import (
	"testing"

	"github.com/root-gg/logger"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

// newSecondaryTestMetadataBackend creates a fresh (erased) SQLite backend
// at a different path, usable as a migration target alongside the primary test backend.
func newSecondaryTestMetadataBackend() *Backend {
	cfg := &Config{Driver: "sqlite3", ConnectionString: "/tmp/plik.migrate.dst.db", EraseFirst: true, Debug: false}
	b, err := NewBackend(cfg, logger.NewLogger())
	if err != nil {
		panic("unable to create secondary metadata backend: " + err.Error())
	}
	return b
}

func TestMigrate_Basic(t *testing.T) {
	src := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(src)

	// Populate source
	user := common.NewUser(common.ProviderLocal, "miguser")
	user.NewToken()
	createUser(t, src, user)

	upload := &common.Upload{}
	upload.NewFile()
	upload.User = user.ID
	createUpload(t, src, upload)

	setting := &common.Setting{Key: "migrate-key", Value: "migrate-value"}
	err := src.CreateSetting(setting)
	require.NoError(t, err)

	// Migrate to dst
	dst := newSecondaryTestMetadataBackend()
	defer shutdownTestMetadataBackend(dst)

	stats, err := Migrate(src, dst, nil)
	require.NoError(t, err)

	require.Equal(t, 1, stats.Users, "expected 1 user migrated")
	require.Equal(t, 0, stats.UserErrors)
	require.Equal(t, 1, stats.Tokens, "expected 1 token migrated")
	require.Equal(t, 0, stats.TokenErrors)
	require.Equal(t, 1, stats.Uploads, "expected 1 upload migrated")
	require.Equal(t, 0, stats.UploadErrors)
	require.Equal(t, 1, stats.Files, "expected 1 file migrated")
	require.Equal(t, 0, stats.FileErrors)
	require.Equal(t, 1, stats.Settings, "expected 1 setting migrated")
	require.Equal(t, 0, stats.SettingErrors)

	// Verify dst has the data
	gotUser, err := dst.GetUser(user.ID)
	require.NoError(t, err)
	require.NotNil(t, gotUser)
	require.Equal(t, user.Login, gotUser.Login)

	gotUpload, err := dst.GetUpload(upload.ID)
	require.NoError(t, err)
	require.NotNil(t, gotUpload)
	require.Equal(t, upload.ID, gotUpload.ID)

	gotSetting, err := dst.GetSetting(setting.Key)
	require.NoError(t, err)
	require.NotNil(t, gotSetting)
	require.Equal(t, setting.Value, gotSetting.Value)
}

func TestMigrate_SoftDeletedUpload(t *testing.T) {
	src := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(src)

	upload := &common.Upload{}
	upload.NewFile()
	createUpload(t, src, upload)

	// Soft-delete the upload
	err := src.RemoveUpload(upload.ID)
	require.NoError(t, err)

	dst := newSecondaryTestMetadataBackend()
	defer shutdownTestMetadataBackend(dst)

	stats, err := Migrate(src, dst, nil)
	require.NoError(t, err)

	// Even soft-deleted uploads and their files should be migrated (FK integrity)
	require.Equal(t, 1, stats.Uploads)
	require.Equal(t, 1, stats.Files)
}

func TestMigrate_IgnoreErrors(t *testing.T) {
	src := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(src)

	// Create two users with the same login (will cause duplicate key on dst)
	user1 := common.NewUser(common.ProviderLocal, "dupuser")
	createUser(t, src, user1)

	dst := newSecondaryTestMetadataBackend()
	defer shutdownTestMetadataBackend(dst)

	// Pre-populate dst with the same user to force a conflict
	createUser(t, dst, user1)

	// Without ignore-errors: should fail
	_, err := Migrate(src, dst, &MigrateOptions{IgnoreErrors: false})
	require.Error(t, err, "expected error on duplicate user")

	// Reset dst
	shutdownTestMetadataBackend(dst)
	dst = newSecondaryTestMetadataBackend()
	defer shutdownTestMetadataBackend(dst)
	createUser(t, dst, user1)

	// With ignore-errors: should succeed and count the error
	stats, err := Migrate(src, dst, &MigrateOptions{IgnoreErrors: true})
	require.NoError(t, err)
	require.Equal(t, 0, stats.Users, "user should not have been counted as success")
	require.Equal(t, 1, stats.UserErrors, "expected 1 user error")
}
