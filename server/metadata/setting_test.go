package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestCreateSetting(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	err := b.CreateSetting(&common.Setting{Key: "foo", Value: "bar"})
	require.NoError(t, err, "create setting error")

	err = b.CreateSetting(&common.Setting{Key: "foo", Value: "bar"})
	require.Error(t, err, "create setting error expected")
}

func TestGetSetting(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	setting, err := b.GetSetting("foo")
	require.NoError(t, err, "get setting error")
	require.Nil(t, setting, "non nil setting")

	err = b.CreateSetting(&common.Setting{Key: "foo", Value: "bar"})
	require.NoError(t, err, "create setting error")

	setting, err = b.GetSetting("foo")
	require.NoError(t, err, "get setting error")
	require.NotNil(t, setting, "nil setting")
	require.Equal(t, "foo", setting.Key, "invalid setting key")
	require.Equal(t, "bar", setting.Value, "invalid setting value")
}

func TestUpdateSetting(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	err := b.UpdateSetting("foo", "bar", "baz")
	require.Error(t, err, "update setting error expected")

	err = b.CreateSetting(&common.Setting{Key: "foo", Value: "bar"})
	require.NoError(t, err, "create setting error")

	err = b.UpdateSetting("foo", "bar", "baz")
	require.NoError(t, err, "update setting error")

	err = b.UpdateSetting("foo", "bar", "baz")
	require.Error(t, err, "update setting error expected")

	setting, err := b.GetSetting("foo")
	require.NoError(t, err, "get setting error")
	require.NotNil(t, setting, "nil setting")
	require.Equal(t, "foo", setting.Key, "invalid setting key")
	require.Equal(t, "baz", setting.Value, "invalid setting value")
}

func TestDeleteSetting(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	err := b.CreateSetting(&common.Setting{Key: "foo", Value: "bar"})
	require.NoError(t, err, "create setting error")

	setting, err := b.GetSetting("foo")
	require.NoError(t, err, "get setting error : %s", err)
	require.NotNil(t, setting, "nil setting")

	err = b.DeleteSetting("foo")
	require.NoError(t, err, "delete setting error : %s", err)

	setting, err = b.GetSetting("foo")
	require.NoError(t, err, "get setting error : %s", err)
	require.Nil(t, setting, "non nil setting")
}

func TestBackend_ForEachSetting(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	err := b.CreateSetting(&common.Setting{Key: "foo", Value: "bar"})
	require.NoError(t, err, "create setting error")

	count := 0
	f := func(setting *common.Setting) error {
		count++
		require.Equal(t, "foo", setting.Key, "invalid setting key")
		require.Equal(t, "bar", setting.Value, "invalid setting value")
		return nil
	}
	err = b.ForEachSetting(f)
	require.NoError(t, err, "for each setting error : %s", err)
	require.Equal(t, 1, count, "invalid setting count")

	f = func(setting *common.Setting) error {
		return fmt.Errorf("expected")
	}
	err = b.ForEachSetting(f)
	require.Errorf(t, err, "expected")
}
