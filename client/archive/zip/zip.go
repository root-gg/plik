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

package zip

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Backend config
type Backend struct {
	Config *BackendConfig
}

// NewZipBackend instantiate a new ZIP Archive Backend
// and configure it from config map
func NewZipBackend(config map[string]interface{}) (zb *Backend, err error) {
	zb = new(Backend)
	zb.Config = NewZipBackendConfig(config)
	if _, err = os.Stat(zb.Config.Zip); os.IsNotExist(err) || os.IsPermission(err) {
		if zb.Config.Zip, err = exec.LookPath("zip"); err != nil {
			err = errors.New("zip binary not found in $PATH, please install or edit ~/.plickrc")
		}
	}
	return
}

// Configure implementation for ZIP Archive Backend
func (zb *Backend) Configure(arguments map[string]interface{}) (err error) {
	if arguments["--archive-options"] != nil && arguments["--archive-options"].(string) != "" {
		zb.Config.Options = arguments["--archive-options"].(string)
	}
	return
}

// Archive implementation for ZIP Archive Backend
func (zb *Backend) Archive(files []string) (reader io.Reader, err error) {
	if len(files) == 0 {
		fmt.Println("Unable to make a zip archive from STDIN")
		os.Exit(1)
		return
	}

	var args []string
	args = append(args, strings.Fields(zb.Config.Options)...)
	args = append(args, "-r", "-")
	args = append(args, files...)

	reader, writer := io.Pipe()

	cmd := exec.Command(zb.Config.Zip, args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	go func() {
		err := cmd.Start()
		if err != nil {
			fmt.Printf("Unable to run zip cmd : %s\n", err)
			os.Exit(1)
			return
		}
		err = cmd.Wait()
		if err != nil {
			fmt.Printf("Unable to run zip cmd : %s\n", err)
			os.Exit(1)
			return
		}
		err = writer.Close()
		if err != nil {
			fmt.Printf("Unable to run zip cmd : %s\n", err)
			return
		}
	}()

	return reader, nil
}

// Comments implementation for ZIP Archive Backend
// Left empty because ZIP can't accept piping to it's STDIN
func (zb *Backend) Comments() string {
	return ""
}

// GetFileName returns the final archive file name
func (zb *Backend) GetFileName(files []string) (name string) {
	name = "archive"
	if len(files) == 1 {
		name = filepath.Base(files[0])
	}
	name += ".zip"
	return
}

// GetConfiguration implementation for ZIP Archive Backend
func (zb *Backend) GetConfiguration() interface{} {
	return zb.Config
}
