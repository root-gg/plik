package s3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfigDefaults(t *testing.T) {
	config := NewConfig(make(map[string]any))
	require.NotNil(t, config, "invalid nil config")
	require.Equal(t, uint64(16*1024*1024), config.PartSize, "invalid default part size")
	require.Equal(t, uint(4), config.PartUploadConcurrency, "invalid default part upload concurrency")
	require.False(t, config.SendContentMd5, "SendContentMd5 should be disabled by default")
}

func TestNewConfigWithSendContentMd5(t *testing.T) {
	config := NewConfig(map[string]any{
		"SendContentMd5": true,
	})
	require.True(t, config.SendContentMd5, "invalid SendContentMd5 override")
}

func TestNewConfigWithPartUploadConcurrency(t *testing.T) {
	config := NewConfig(map[string]any{
		"PartUploadConcurrency": 4,
	})
	require.Equal(t, uint(4), config.PartUploadConcurrency, "invalid PartUploadConcurrency override")
}

func TestNewPutObjectOptions(t *testing.T) {
	backend := &Backend{
		config: &Config{
			SendContentMd5: true,
		},
	}

	opts := backend.newPutObjectOptions("application/octet-stream")
	require.Equal(t, "application/octet-stream", opts.ContentType, "invalid content type")
	require.True(t, opts.SendContentMd5, "invalid send content md5 option")
}
