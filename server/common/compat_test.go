package common

import (
	"testing"

	"github.com/root-gg/utils"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalUpload(t *testing.T) {
	u := &Upload{}
	u.NewFile()

	bytes, _ := utils.ToJson(u)
	upload := &Upload{}
	version, err := UnmarshalUpload(bytes, upload)
	require.NoError(t, err, "unmarshal upload error")
	require.Equal(t, 0, version, "invalid version")
}

func TestUnmarshalUploadV1(t *testing.T) {
	v1 := &UploadV1{}
	v1.Files = make(map[string]*File)
	v1.Files["1"] = &File{ID: "1"}

	bytes, _ := utils.ToJson(v1)
	upload := &Upload{}
	version, err := UnmarshalUpload(bytes, upload)
	require.NoError(t, err, "unmarshal upload error")
	require.Equal(t, 1, version, "invalid version")
}

func TestUnmarshalUploadInvalid(t *testing.T) {
	upload := &Upload{}
	_, err := UnmarshalUpload([]byte("blah"), upload)
	require.Error(t, err, "unmarshal upload error expected")
}

func TestMarshalUpload(t *testing.T) {
	u := &Upload{}
	u.NewFile()

	bytes, err := MarshalUpload(u, 0)
	require.NoError(t, err, "marshal upload error")
	require.NotZero(t, len(bytes), "invalid json length")
}

func TestMarshalUploadV1(t *testing.T) {
	u := &Upload{}
	u.NewFile()

	bytes, err := MarshalUpload(u, 1)
	require.NoError(t, err, "marshal upload error")
	require.NotZero(t, len(bytes), "invalid json length")
}
