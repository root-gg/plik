/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package stream

import (
	"bytes"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/root-gg/juliet"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/stretchr/testify/require"
)

func newTestingContext(config *common.Configuration) (ctx *juliet.Context) {
	ctx = juliet.NewContext()
	ctx.Set("config", config)
	ctx.Set("logger", logger.NewLogger())
	return ctx
}

func TestAddGetFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	config := NewConfig(make(map[string]interface{}))
	backend := NewBackend(config)

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		details, err := backend.AddFile(ctx, upload, file, bytes.NewBufferString("data"))
		require.NoError(t, err, "unable to add file")
		require.NotNil(t, details, "invalid nil details")
		wg.Done()
	}()

	f := func() {
		for {
			reader, err := backend.GetFile(ctx, upload, file.ID)
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
	ctx := newTestingContext(common.NewConfiguration())

	config := NewConfig(make(map[string]interface{}))
	backend := NewBackend(config)

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	err := backend.RemoveFile(ctx, upload, file.ID)
	require.Error(t, err, "able to remove file")
}

func TestRemoveUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	config := NewConfig(make(map[string]interface{}))
	backend := NewBackend(config)

	upload := common.NewUpload()
	upload.Create()

	err := backend.RemoveUpload(ctx, upload)
	require.Error(t, err, "able to remove upload")
}
