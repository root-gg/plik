package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func createMetadata(t *testing.T, b *Backend) {
	user := common.NewUser(common.ProviderLocal, "user")
	user.NewToken()
	createUser(t, b, user)

	upload := &common.Upload{}
	upload.NewFile()
	upload.User = user.ID
	upload.Token = user.Tokens[0].Token
	createUpload(t, b, upload)

	setting := &common.Setting{Key: "foo", Value: "bar"}
	err := b.CreateSetting(setting)
	require.NoError(t, err)
}

func TestBackend_Export(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	createMetadata(t, b)

	path := "/tmp/plik.metadata.test.snappy.gob"
	err := b.Export(path)
	require.NoError(t, err, "export error %s", err)

	b = newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	err = b.Import(path, &ImportOptions{})
	require.NoError(t, err, "import error %s", err)
}

func TestBackend_ExportRemovedFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	upload.NewFile()
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
}
