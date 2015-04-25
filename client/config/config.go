package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	homedir "github.com/mitchellh/go-homedir"
	"os"
)

var Config *UploadConfig

type UploadConfig struct {
	Debug          bool
	Quiet          bool
	HomeDir        string
	Url            string
	OneShot        bool
	Removable      bool
	Secure         bool
	SecureMethod   string
	SecureOptions  map[string]interface{}
	Archive        bool
	ArchiveMethod  string
	ArchiveOptions map[string]interface{}
	DownloadBinary string
	Comments       string
	Yubikey        bool
	Password       string
	Ttl            int
}

func NewUploadConfig() (config *UploadConfig) {
	config = new(UploadConfig)
	config.Debug = false
	config.Quiet = false
	config.Url = "http://127.0.0.1:8080"
	config.OneShot = false
	config.Removable = false
	config.Secure = false
	config.Archive = false
	config.ArchiveMethod = "tar"
	config.ArchiveOptions = make(map[string]interface{})
	config.ArchiveOptions["Tar"] = "/bin/tar"
	config.ArchiveOptions["Compress"] = "gzip"
	config.ArchiveOptions["Options"] = ""
	config.SecureMethod = "openssl"
	config.SecureOptions = make(map[string]interface{})
	config.SecureOptions["Openssl"] = "/usr/bin/openssl"
	config.SecureOptions["Cipher"] = "aes256"
	config.DownloadBinary = "curl"
	config.Comments = ""
	config.Yubikey = false
	config.Password = ""
	config.Ttl = 86400 * 30
	return
}

func Load() (err error) {
	Config = NewUploadConfig()

	// Detect home dir
	home, err := homedir.Dir()
	if err != nil {
		Config.HomeDir = os.Getenv("HOME")
	} else {
		Config.HomeDir = home
	}

	// Stat file
	configFile := home + "/.plikrc"
	_, err = os.Stat(configFile)
	if err != nil {
		// File not present
		// Encode in toml
		buf := new(bytes.Buffer)
		if err = toml.NewEncoder(buf).Encode(Config); err != nil {
			return errors.New(fmt.Sprint("Failed to serialize ~/.plickrc : %s", err))
		}

		// Write file
		f, err := os.OpenFile(configFile, os.O_CREATE|os.O_RDWR, 0700)
		if err != nil {
			return errors.New(fmt.Sprint("Failed to save ~/.plickrc : %s", err))
		}

		f.Write(buf.Bytes())
		f.Close()
	} else {
		// Load toml
		if _, err := toml.DecodeFile(configFile, &Config); err != nil {
			return errors.New(fmt.Sprint("Failed to deserialize ~/.plickrc : %s", err))
		}
	}
	return
}

func Debug(message string) {
	if Config.Debug {
		fmt.Println(message)
	}
}

func Dump(data interface{}) {
	fmt.Println(Sdump(data))
}

func Sdump(data interface{}) string {
	buf := new(bytes.Buffer)
	if json, err := json.MarshalIndent(data, "", "    "); err != nil {
		fmt.Println("Unable to dump data %v : %s", data, err)
	} else {
		buf.Write(json)
		buf.WriteString("\n")
	}
	return string(buf.Bytes())
}
