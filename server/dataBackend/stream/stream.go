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
	"io"

	"time"

	"github.com/root-gg/plik/server/common"
)

// Backend object
type Backend struct {
	Config *BackendConfig
	Store  map[string]io.ReadCloser
}

// NewStreamBackend instantiate a new Stream Data Backend
// from configuration passed as argument
func NewStreamBackend(config map[string]interface{}) (sb *Backend) {
	sb = new(Backend)
	sb.Config = NewStreamBackendConfig(config)
	sb.Store = make(map[string]io.ReadCloser)
	return
}

// GetFile implementation for steam data backend will search
// on filesystem the requested steam and return its reading filehandle
func (sb *Backend) GetFile(ctx *common.PlikContext, upload *common.Upload, id string) (stream io.ReadCloser, err error) {
	defer ctx.Finalize(err)
	storeID := upload.ID + "/" + id
	stream, ok := sb.Store[storeID]
	if !ok {
		err = ctx.EWarningf("Missing reader")
	}
	delete(sb.Store, id)
	return
}

// AddFile implementation for steam data backend will creates a new steam for the given upload
// and save it on filesystem with the given steam reader
func (sb *Backend) AddFile(ctx *common.PlikContext, upload *common.Upload, file *common.File, stream io.Reader) (backendDetails map[string]interface{}, err error) {
	defer ctx.Finalize(err)
	backendDetails = make(map[string]interface{})
	id := upload.ID + "/" + file.ID
	pipeReader, pipeWriter := io.Pipe()
	sb.Store[id] = pipeReader
	ctx.Infof("Store in %s", id)
	buf := make([]byte, 1024)
	for {
		done := make(chan struct{})
		go func() {
			var size int
			size, err = stream.Read(buf)
			pipeWriter.Write(buf[:size])
			done <- struct{}{}
		}()
		timer := time.NewTimer(time.Duration(sb.Config.Timeout) * time.Second)
		select {
		case <-done:
			timer.Stop()
		case <-timer.C:
			err = ctx.EWarning("timeout")
		}
		if err != nil {
			ctx.Info(err.Error())
			break
		}
	}
	pipeReader.Close()
	delete(sb.Store, id)
	if err == io.EOF {
		err = nil
	}
	return
}

// RemoveFile implementation for steam data backend will delete the given
// steam from filesystem
func (sb *Backend) RemoveFile(ctx *common.PlikContext, upload *common.Upload, id string) (err error) {
	defer ctx.Finalize(err)
	return
}

// RemoveUpload implementation for steam data backend will
// delete the whole upload. Given that an upload is a directory,
// we remove the whole directory at once.
func (sb *Backend) RemoveUpload(ctx *common.PlikContext, upload *common.Upload) (err error) {
	defer ctx.Finalize(err)
	return
}
