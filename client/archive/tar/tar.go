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

package tar

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Backend object
type Backend struct {
	Config *BackendConfig
}

// NewTarBackend instantiate a new Tar Archive Backend
// and configure it from config map
func NewTarBackend(config map[string]interface{}) (tb *Backend, err error) {
	tb = new(Backend)
	tb.Config = NewTarBackendConfig(config)
	if _, err = os.Stat(tb.Config.Tar); os.IsNotExist(err) || os.IsPermission(err) {
		if tb.Config.Tar, err = exec.LookPath("tar"); err != nil {
			err = errors.New("tar binary not found in $PATH, please install or edit ~/.plickrc")
		}
	}
	return
}

// Configure implementation for TAR Archive Backend
func (tb *Backend) Configure(arguments map[string]interface{}) (err error) {
	if arguments["--compress"] != nil && arguments["--compress"].(string) != "" {
		tb.Config.Compress = arguments["--compress"].(string)
	}
	if arguments["--archive-options"] != nil && arguments["--archive-options"].(string) != "" {
		tb.Config.Options = arguments["--archive-options"].(string)
	}
	return
}

// Archive implementation for TAR Archive Backend
func (tb *Backend) Archive(files []string, writer io.WriteCloser) (err error) {
	if len(files) == 0 {
		fmt.Println("Unable to make a tar archive from STDIN")
		os.Exit(1)
		return
	}

	var args []string
	args = append(args, "--create")
	if tb.Config.Compress != "no" {
		args = append(args, "--"+tb.Config.Compress)
	}
	args = append(args, strings.Fields(tb.Config.Options)...)
	args = append(args, files...)

	cmd := exec.Command(tb.Config.Tar, args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	go func() {
		err := cmd.Start()
		if err != nil {
			fmt.Printf("Unable to run tar cmd : %s\n", err)
			os.Exit(1)
			return
		}
		err = cmd.Wait()
		if err != nil {
			fmt.Printf("Unable to run tar cmd : %s\n", err)
			os.Exit(1)
			return
		}
		err = writer.Close()
		if err != nil {
			fmt.Printf("Unable to run tar cmd : %s\n", err)
			return
		}
	}()
	return
}

// Comments implementation for TAR Archive Backend
func (tb *Backend) Comments() string {
	comment := "tar xvf -"
	if tb.Config.Compress != "no" {
		comment += " --" + tb.Config.Compress
	}

	return comment
}

// GetConfiguration implementation for TAR Archive Backend
func (tb *Backend) GetConfiguration() interface{} {
	return tb.Config
}

// GetFileName returns the final archive file name
func (tb *Backend) GetFileName(files []string) (name string) {
	name = "archive"
	if len(files) == 1 {
		name = filepath.Base(files[0])
	}
	name += ".tar" + getCompressExtention(tb.Config.Compress)
	return
}

func getCompressExtention(mode string) string {
	switch mode {
	case "gzip":
		return ".gz"
	case "bzip2":
		return ".bz2"
	case "xz":
		return ".xz"
	case "lzip":
		return ".lz"
	case "lzop":
		return ".lzo"
	case "lzma":
		return ".lzma"
	case "compres":
		return ".Z"
	default:
		return ""
	}
}
