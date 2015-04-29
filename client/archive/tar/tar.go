package tar

import (
	"errors"
	"fmt"
	"github.com/root-gg/plik/client/config"
	"github.com/root-gg/utils"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type TarBackendConfig struct {
	Tar      string
	Compress string
	Options  string
}

func NewTarBackendConfig(config map[string]interface{}) (this *TarBackendConfig) {
	this = new(TarBackendConfig)
	this.Tar = "/bin/tar"
	this.Compress = "gzip"
	utils.Assign(this, config)
	return
}

type TarBackend struct {
	Config *TarBackendConfig
}

func NewTarBackend(config map[string]interface{}) (this *TarBackend, err error) {
	this = new(TarBackend)
	this.Config = NewTarBackendConfig(config)
	if _, err = os.Stat(this.Config.Tar); os.IsNotExist(err) || os.IsPermission(err) {
		if this.Config.Tar, err = exec.LookPath("tar"); err != nil {
			err = errors.New("tar binary not found in $PATH, please install or edit ~/.plickrc")
		}
	}
	return
}

func (this *TarBackend) Configure(arguments map[string]interface{}) (err error) {
	if arguments["--compress"] != nil && arguments["--compress"].(string) != "" {
		this.Config.Compress = arguments["--compress"].(string)
	}
	if arguments["--archive-options"] != nil && arguments["--archive-options"].(string) != "" {
		this.Config.Options = arguments["--archive-options"].(string)
	}
	config.Debug("Tar configuration : " + config.Sdump(this.Config))
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
func (this *TarBackend) Archive(files []string, writer io.WriteCloser) (name string, err error) {
	if len(files) == 0 {
		fmt.Println("Unable to make a tar archive from STDIN")
		os.Exit(1)
		return
	}

	name = "archive"
	if len(files) == 1 {
		name = filepath.Base(files[0])
	}
	name += ".tar" + getCompressExtention(this.Config.Compress)

	args := make([]string, 0)
	args = append(args, "--create")
	if this.Config.Compress != "no" {
		args = append(args, "--"+this.Config.Compress)
	}
	args = append(args, strings.Fields(this.Config.Options)...)
	args = append(args, files...)

	cmd := exec.Command(this.Config.Tar, args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	go func() {
		err = cmd.Start()
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

func (this *TarBackend) Comments() string {
	if this.Config.Compress != "no" {
		return "tar zxvf -"
	} else {
		return "tar xvf -"
	}
}
