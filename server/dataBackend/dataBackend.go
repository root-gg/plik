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

package dataBackend

import (
	"io"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/dataBackend/file"
	"github.com/root-gg/plik/server/dataBackend/stream"
	"github.com/root-gg/plik/server/dataBackend/swift"
	"github.com/root-gg/plik/server/dataBackend/weedfs"
)

var dataBackend DataBackend
var streamBackend DataBackend

// DataBackend interface describes methods that data backends
// must implements to be compatible with plik.
type DataBackend interface {
	GetFile(ctx *juliet.Context, u *common.Upload, id string) (rc io.ReadCloser, err error)
	AddFile(ctx *juliet.Context, u *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error)
	RemoveFile(ctx *juliet.Context, u *common.Upload, id string) (err error)
	RemoveUpload(ctx *juliet.Context, u *common.Upload) (err error)
}

// GetDataBackend return the primary data backend
func GetDataBackend() DataBackend {
	return dataBackend
}

// GetStreamBackend return the stream data backend
func GetStreamBackend() DataBackend {
	return streamBackend
}

// Initialize backend from type found in configuration
func Initialize() {
	if dataBackend == nil {
		switch common.Config.DataBackend {
		case "file":
			dataBackend = file.NewFileBackend(common.Config.DataBackendConfig)
		case "swift":
			dataBackend = swift.NewSwiftBackend(common.Config.DataBackendConfig)
		case "weedfs":
			dataBackend = weedfs.NewWeedFsBackend(common.Config.DataBackendConfig)
		default:
			common.Logger().Fatalf("Invalid data backend %s", common.Config.DataBackend)
		}
	}
	if common.Config.StreamMode {
		streamBackend = stream.NewStreamBackend(common.Config.StreamBackendConfig)
	}
}
