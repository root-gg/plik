package zip

import (
	"github.com/root-gg/plik/server/utils"
	"io"
	"os/exec"
	//	"strings"
	"errors"
	"fmt"
	"github.com/root-gg/plik/client/config"
	"os"
	"path/filepath"
	"strings"
)

type ZipBackendConfig struct {
	Zip     string
	Options string
}

func NewZipBackendConfig(config map[string]interface{}) (this *ZipBackendConfig) {
	this = new(ZipBackendConfig)
	this.Zip = "/bin/zip"
	utils.Assign(this, config)
	return
}

type ZipBackend struct {
	Config *ZipBackendConfig
}

func NewZipBackend(config map[string]interface{}) (this *ZipBackend, err error) {
	this = new(ZipBackend)
	this.Config = NewZipBackendConfig(config)
	if _, err := os.Stat(this.Config.Zip); os.IsNotExist(err) || os.IsPermission(err) {
		if this.Config.Zip, err = exec.LookPath("zip"); err != nil {
			err = errors.New("zip binary not found in $PATH, please install or edit ~/.plickrc")
		}
	}
	return
}

func (this *ZipBackend) Configure(arguments map[string]interface{}) (err error) {
	if arguments["--archive-options"] != nil && arguments["--archive-options"].(string) != "" {
		this.Config.Options = arguments["--archive-options"].(string)
	}
	config.Debug("Zip configuration : " + config.Sdump(this.Config))
	return
}

func (this *ZipBackend) Archive(files []string, writer io.WriteCloser) (name string, err error) {
	if len(files) == 0 {
		fmt.Println("Unable to make a zip archive from STDIN")
		os.Exit(1)
		return
	}

	name = "archive"
	if len(files) == 1 {
		name = filepath.Base(files[0])
	}
	name += ".zip"

	args := make([]string, 0)
	args = append(args, strings.Fields(this.Config.Options)...)
	args = append(args, "-r", "-")
	args = append(args, files...)

	cmd := exec.Command(this.Config.Zip, args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	go func() {
		err = cmd.Start()
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
	return
}

func (this *ZipBackend) Comments() string {
	return ""
}
