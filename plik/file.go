/**

    Plik upload client

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

package plik

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/root-gg/plik/server/common"
)

// File contains all relevant info needed to upload data to a Plik server
type File struct {
	*common.File           // File metadata
	Reader       io.Reader // Byte stream to upload
	Error        error     // Status of the upload
	Done         func()    // Upload callback
}

// NewFileFromReader creates a File from a filename and an io.Reader
func NewFileFromReader(name string, reader io.Reader) *File {
	file := &File{}
	file.File = &common.File{}
	file.Name = name
	file.Reader = reader
	return file
}

// NewFileFromPath creates a File from a filesystem path
func NewFileFromPath(path string) (file *File, err error) {
	file = &File{}
	file.File = &common.File{}
	file.Name = filepath.Base(path)

	// Test if file exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("File %s not found", path)
	}

	// Check mode
	if fileInfo.Mode().IsRegular() {
		file.CurrentSize = fileInfo.Size()
	} else {
		return nil, fmt.Errorf("Unhandled file mode %s for file %s", fileInfo.Mode().String(), path)
	}

	// Open file
	file.Reader, err = os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Unable to open %s : %s", path, err)
	}

	return file, err
}
