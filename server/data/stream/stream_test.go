package stream

import (
	"bytes"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestAddGetFile(t *testing.T) {
	backend := NewBackend()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		err := backend.AddFile(file, bytes.NewBufferString("data"))
		require.NoError(t, err, "unable to add file")
		require.NotNil(t, file.BackendDetails, "invalid nil details")
		wg.Done()
	}()

	f := func() {
		for {
			reader, err := backend.GetFile(file)
			if err != nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			data, err := ioutil.ReadAll(reader)
			require.NoError(t, err, "unable to read reader")

			err = reader.Close()
			require.NoError(t, err, "unable to close reader")

			require.Equal(t, "data", string(data), "invalid reader content")
			break
		}
		wg.Wait()
	}

	err := common.TestTimeout(f, 1*time.Second)
	require.NoError(t, err, "timeout")
}

func TestRemoveFile(t *testing.T) {
	backend := NewBackend()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	err := backend.RemoveFile(file)
	require.NoError(t, err)
}
