package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFile(t *testing.T) {
	file := NewFile()
	require.NotNil(t, file, "invalid file")
	require.NotZero(t, file.ID, "invalid file id")
}

func TestFileGenerateID(t *testing.T) {
	file := &File{}
	file.GenerateID()
	require.NotEqual(t, "", file.ID, "missing file id")
}

func TestFileSanitize(t *testing.T) {
	file := &File{}
	file.BackendDetails = "value"
	file.Sanitize()
	require.Zero(t, file.BackendDetails, "invalid backend details")
}

func TestFilePrepareInsert(t *testing.T) {
	upload := &Upload{}
	file := &File{}
	file.BackendDetails = "value"

	err := file.PrepareInsert(nil)
	require.Errorf(t, err, "missing upload")

	err = file.PrepareInsert(upload)
	require.Errorf(t, err, "upload not initialized")

	upload.PrepareInsertForTests()

	err = file.PrepareInsert(upload)
	require.Errorf(t, err, "missing file name")

	for i := 0; i < 2048; i++ {
		file.Name += "x"
	}

	err = file.PrepareInsert(upload)
	require.Errorf(t, err, "too long")

	file.Name = "file name"
	err = file.PrepareInsert(upload)
	require.NoError(t, err, "too long")

	require.NotNil(t, file.ID, "missing file id")
	require.Equal(t, FileMissing, file.Status, "missing file id")
}
