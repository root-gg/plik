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

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/root-gg/plik/client/archive"
	"github.com/root-gg/plik/client/crypto"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var Config *UploadConfig
var Upload *common.Upload
var Files []*FileToUpload

var CryptoBackend crypto.CryptoBackend
var ArchiveBackend archive.ArchiveBackend

var LongestFilenameSize int

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

type FileToUpload struct {
	Path       string
	Base       string
	Size       int64
	FileHandle *os.File
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
	config.SecureOptions["Cipher"] = "aes-256-cbc"
	config.DownloadBinary = "curl"
	config.Comments = ""
	config.Yubikey = false
	config.Password = ""
	config.Ttl = 86400 * 30
	return
}

func Load() (err error) {
	Config = NewUploadConfig()
	Upload = common.NewUpload()
	Files = make([]*FileToUpload, 0)

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
		// File not present. Ask for domain
		var domain string
		fmt.Printf("Please enter your plik domain [default:http://127.0.0.1:8080] : ")
		_, err := fmt.Scanf("%s", &domain)
		if err == nil {
			Config.Url = domain
			if !strings.HasPrefix(domain, "http") {
				Config.Url = "http://" + domain
			}
		}

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

// Here, we are unmarshalling args into upload informations
// Argument takes priority over config file param
func UnmarshalArgs(arguments map[string]interface{}) (err error) {

	// Handle flags
	if arguments["--debug"].(bool) {
		Config.Debug = true
	}
	if arguments["--quiet"].(bool) {
		Config.Quiet = true
	}

	Debug("Arguments : " + Sdump(arguments))
	Debug("Configuration : " + Sdump(Config))

	// Plik url
	if arguments["--server"] != nil && arguments["--server"].(string) != "" {
		Config.Url = arguments["--server"].(string)
	}

	// Check files
	if _, ok := arguments["FILE"].([]string); ok {

		// Test if they exist
		for _, filePath := range arguments["FILE"].([]string) {

			fileToUpload := new(FileToUpload)
			fileToUpload.Path = filePath
			fileToUpload.Base = filepath.Base(filePath)

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				return errors.New(fmt.Sprintf("File %s not found", filePath))
			}

			// Check file size (for displaying purpose later)
			if len(fileToUpload.Base) > LongestFilenameSize {
				LongestFilenameSize = len(fileToUpload.Base)
			}

			// Check mode
			// Enable archive if one of them is a directory
			if fileInfo.Mode().IsDir() {
				Config.Archive = true
			} else if fileInfo.Mode().IsRegular() {
				fileToUpload.Size = fileInfo.Size()
			} else {
				return errors.New(fmt.Sprintf("Unhandled file mode %s for file %s", fileInfo.Mode().String(), filePath))
			}

			Files = append(Files, fileToUpload)
		}
	} else {
		return errors.New(fmt.Sprint("No files specified"))
	}

	// Upload options
	Upload.OneShot = Config.OneShot
	if arguments["--oneshot"].(bool) {
		Upload.OneShot = true
	}
	Upload.Removable = Config.Removable
	if arguments["--removable"].(bool) {
		Upload.OneShot = true
	}
	Upload.Comments = Config.Comments
	if arguments["--comments"] != nil && arguments["--comments"].(string) != "" {
		Upload.Comments = arguments["--comments"].(string)
	}

	// Upload time to live
	Upload.Ttl = Config.Ttl
	if arguments["--ttl"] != nil && arguments["--ttl"].(string) != "" {
		ttlStr := arguments["--ttl"].(string)
		mul := 1
		if string(ttlStr[len(ttlStr)-1]) == "m" {
			mul = 60
		} else if string(ttlStr[len(ttlStr)-1]) == "h" {
			mul = 3600
		} else if string(ttlStr[len(ttlStr)-1]) == "d" {
			mul = 86400
		}
		if mul != 1 {
			ttlStr = ttlStr[:len(ttlStr)-1]
		}
		ttl, err := strconv.Atoi(ttlStr)
		if err != nil {
			return errors.New(fmt.Sprintf("Invalid TTL %s", arguments["--ttl"].(string)))
		}
		Upload.Ttl = ttl * mul
	}

	// Do we need a crypto backend ?
	if arguments["-s"].(bool) || arguments["--secure"] != nil || Config.Secure {
		Config.Secure = true
		secureMethod := Config.SecureMethod
		if arguments["--secure"] != nil && arguments["--secure"].(string) != "" {
			secureMethod = arguments["--secure"].(string)
		}
		var err error
		CryptoBackend, err = crypto.NewCryptoBackend(secureMethod, Config.SecureOptions)
		if err != nil {
			return errors.New(fmt.Sprintf("Invalid secure params : %s\n", err))
		}
		err = CryptoBackend.Configure(arguments)
		if err != nil {
			return errors.New(fmt.Sprintf("Invalid secure params : %s\n", err))
		}

		Debug("Crypto backend configuration : " + utils.Sdump(CryptoBackend.GetConfiguration()))
	}

	// Do we need an archive backend
	if arguments["-a"].(bool) || arguments["--archive"] != nil || Config.Archive {
		Config.Archive = true

		if arguments["--archive"] != nil && arguments["--archive"] != "" {
			Config.ArchiveMethod = arguments["--archive"].(string)
		}
		ArchiveBackend, err = archive.NewArchiveBackend(Config.ArchiveMethod, Config.ArchiveOptions)
		if err != nil {
			return errors.New(fmt.Sprintf("Invalid archive params : %s\n", err))
		}
		err = ArchiveBackend.Configure(arguments)
		if err != nil {
			return errors.New(fmt.Sprintf("Invalid archive params : %s\n", err))
		}

		Debug("Archive backend configuration : " + utils.Sdump(ArchiveBackend.GetConfiguration()))
	} else {
		for _, fileToUpload := range Files {
			fh, err := os.Open(fileToUpload.Path)
			if err != nil {
				return errors.New(fmt.Sprintf("Unable to open %s : %s", fileToUpload.Path, err))
			}

			fileToUpload.FileHandle = fh
		}
	}

	// Do user wants a password protected upload ?
	if arguments["-p"].(bool) {
		fmt.Printf("Login [plik]: ")
		var err error
		_, err = fmt.Scanln(&Upload.Login)
		if err != nil && err.Error() != "unexpected newline" {
			return errors.New(fmt.Sprintf("Unable to get login : %s", err))
		}
		if Upload.Login == "" {
			Upload.Login = "plik"
		}
		fmt.Printf("Password: ")
		_, err = fmt.Scanln(&Upload.Password)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to get password : %s", err))
		}
	} else if arguments["--password"] != nil && arguments["--password"].(string) != "" {
		credentials := arguments["--password"].(string)
		sepIndex := strings.Index(credentials, ":")
		var login, password string
		if sepIndex > 0 {
			login = credentials[:sepIndex]
			password = credentials[sepIndex+1:]
		} else {
			login = "plik"
			password = credentials
		}
		Upload.Login = login
		Upload.Password = password
	}

	// User wants Yubikey protected upload ?
	if Config.Yubikey || arguments["--yubikey"].(bool) {
		fmt.Printf("Yubikey token : ")
		_, err := fmt.Scanln(&Upload.Yubikey)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to get yubikey token : %s", err))
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
