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
