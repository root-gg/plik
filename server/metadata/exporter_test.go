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
	createMetadata(t, b)

	path := "/tmp/plik.metadata.test.snappy.gob"
	err := b.Export(path)
	require.NoError(t, err, "export error %s", err)

	b = newTestMetadataBackend()
	err = b.Import(path)
	require.NoError(t, err, "import error %s", err)
}
