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

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/cheggaaa/pb"
	docopt "github.com/docopt/docopt-go"
	"github.com/olekukonko/ts"
	"github.com/root-gg/plik/client/archive"
	"github.com/root-gg/plik/client/config"
	"github.com/root-gg/plik/client/crypto"
	"github.com/root-gg/plik/server/common"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Vars
var arguments map[string]interface{}

var baseUrl string
var transport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}
var client = http.Client{Transport: transport}
var cryptoBackend crypto.CryptoBackend
var archiveBackend archive.ArchiveBackend
var basicAuth string

// Main
func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	ts.GetSize()

	// Load config
	config.Load()

	// Usage /!\ INDENT THIS WITH SPACES NOT TABS /!\
	usage := `plik

Usage:
  plik [options] [FILE] ...

Options:
  -h --help                 Show this help
  -d --debug                Enable debug mode
  -q --quiet                Enable quiet mode
  -v --version              Show plik version
  -o, --oneshot             Enable OneShot (Each file will be deleted on first download)
  -r, --removable           Enable Removable upload (Each file can be deleted by anyone at anymoment)
  -t, --ttl TTL             Time before expiration (Upload will be removed in m|h|d)
  -n, --name NAME           Set file name when piping from STDIN
  --comments COMMENT        Set comments of the upload (MarkDown compatible)
  --archive-options OPTIONS [tar|zip] Additional command line options
  -p                        Protect the upload with login and password
  --password PASSWD         Protect the upload with login:password ( if omitted default login is "plik" )
  -y, --yubikey             Protect the upload with a Yubikey OTP
  -a                        Archive upload using default archive params ( see ~/.plikrc )
  --archive MODE            Archive upload using specified archive backend : tar|zip
  --compress MODE           [tar] Compression codec : gzip|bzip2|xz|lzip|lzma|lzop|compress|no
  -s                        Encrypt upload usnig default encrypt params ( see ~/.plikrc )
  --secure MODE             Archive upload using specified archive backend : openssl|pgp
  --cipher CIPHER           [openssl] Openssl cipher to use ( see openssl help )
  --passphrase PASSPHRASE   [openssl] Passphrase or '-' to be prompted for a passphrase
  --secure-options OPTIONS  [openssl|pgp] Additional command line options
  --recipient RECIPIENT     [pgp] Set recipient for pgp backend ( example : --recipient Bob )
`
	arguments, _ = docopt.Parse(usage, nil, true, "", false)

	if arguments["--debug"].(bool) {
		config.Config.Debug = true
	}
	if arguments["--quiet"].(bool) {
		config.Config.Quiet = true
	}

	config.Debug("Arguments : " + config.Sdump(arguments))
	config.Debug("Configuration : " + config.Sdump(config.Config))

	baseUrl = config.Config.Url
	if arguments["--server"] != nil && arguments["--server"].(string) != "" {
		baseUrl = arguments["--server"].(string)
	}

	uploadInfo := new(common.Upload)

	uploadInfo.OneShot = config.Config.OneShot
	if arguments["--oneshot"].(bool) {
		uploadInfo.OneShot = true
	}

	uploadInfo.Removable = config.Config.Removable
	if arguments["--removable"].(bool) {
		uploadInfo.OneShot = true
	}

	// TODO Generate comments if encrypted
	uploadInfo.Comments = config.Config.Comments
	if arguments["--comments"] != nil && arguments["--comments"].(string) != "" {
		uploadInfo.Comments = arguments["--comments"].(string)
	}

	uploadInfo.Ttl = config.Config.Ttl
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
			fmt.Println("Invalid TTL %s", arguments["--ttl"].(string))
			os.Exit(1)
		}
		uploadInfo.Ttl = ttl * mul
	}

	if arguments["-s"].(bool) || arguments["--secure"] != nil {
		config.Config.Secure = true
		secureMethod := config.Config.SecureMethod
		if arguments["--secure"] != nil && arguments["--secure"].(string) != "" {
			secureMethod = arguments["--secure"].(string)
		}
		var err error
		cryptoBackend, err = crypto.NewCryptoBackend(secureMethod, config.Config.SecureOptions)
		if err != nil {
			fmt.Printf("Invalid secure params : %s\n", err)
			os.Exit(1)
		}
		err = cryptoBackend.Configure(arguments)
		if err != nil {
			fmt.Printf("Invalid secure params : %s\n", err)
			os.Exit(1)
		}
	}

	if arguments["-p"].(bool) {
		fmt.Printf("Login [plik]: ")
		var err error
		_, err = fmt.Scanln(&uploadInfo.Login)
		if err != nil && err.Error() != "unexpected newline" {
			fmt.Printf("Unable to get login : %s", err)
			os.Exit(1)
		}
		if uploadInfo.Login == "" {
			uploadInfo.Login = "plik"
		}
		fmt.Printf("Password: ")
		_, err = fmt.Scanln(&uploadInfo.Password)
		if err != nil {
			fmt.Printf("Unable to get password : %s", err)
			os.Exit(1)
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
		uploadInfo.Login = login
		uploadInfo.Password = password
	}

	if config.Config.Yubikey || arguments["--yubikey"].(bool) {
		fmt.Printf("Yubikey token : ")
		_, err := fmt.Scanln(&uploadInfo.Yubikey)
		if err != nil {
			fmt.Println("Unable to get yubikey token : %s", err)
			os.Exit(1)
		}
	}

	config.Debug("Sending upload params : " + config.Sdump(uploadInfo))
	var err error
	uploadInfo, err = createUpload(uploadInfo)
	if err != nil {
		printf("Unable to create upload : %s\n", err)
		os.Exit(1)
	}
	config.Debug("Got upload info : " + config.Sdump(uploadInfo))

	printf("Upload successfully created : \n\n")
	printf("    %s/#/?id=%s\n\n\n", baseUrl, uploadInfo.Id)

	count := 0
	totalSize := 0
	if arguments["-a"].(bool) || arguments["--archive"] != nil {
		config.Config.Archive = true

		if arguments["--archive"] != nil && arguments["--archive"] != "" {
			config.Config.ArchiveMethod = arguments["--archive"].(string)
		}
		archiveBackend, err = archive.NewArchiveBackend(config.Config.ArchiveMethod, config.Config.ArchiveOptions)
		if err != nil {
			printf("Invalid archive params : %s\n", err)
			os.Exit(1)
		}
		err = archiveBackend.Configure(arguments)
		if err != nil {
			printf("Invalid archive params : %s\n", err)
			os.Exit(1)
		}

		pipeReader, pipeWriter := io.Pipe()
		name, err := archiveBackend.Archive(arguments["FILE"].([]string), pipeWriter)
		if err != nil {
			printf("Unable to archive files : %s\n", err)
			os.Exit(1)
		}

		if arguments["--name"] != nil && arguments["--name"].(string) != "" {
			name = arguments["--name"].(string)
		}

		file, err := upload(uploadInfo, name, int64(0), pipeReader)
		if err != nil {
			printf("Unable to upload from STDIN : %s\n", err)
			return
		}
		pipeReader.CloseWithError(err)

		//fmt.Println(utils.Dump(file))
		printFileInformations(uploadInfo, file)
		count++
		totalSize += int(file.CurrentSize)
		uploadInfo.Files[file.Id] = file
	} else {
		if len(arguments["FILE"].([]string)) == 0 {
			// Upload from STDIN
			name := "STDIN"
			if arguments["--name"] != nil && arguments["--name"].(string) != "" {
				name = arguments["--name"].(string)
			}

			file, err := upload(uploadInfo, name, int64(0), os.Stdin)
			if err != nil {
				printf("Unable to upload from STDIN : %s\n", err)
				return
			}

			printFileInformations(uploadInfo, file)
			count++
			totalSize += int(file.CurrentSize)
			uploadInfo.Files[file.Id] = file
		} else {
			// Upload individual files
			var wg sync.WaitGroup
			for _, path := range arguments["FILE"].([]string) {
				wg.Add(1)
				go func(path string) {
					defer wg.Done()

					fileHandle, err := os.Open(path)
					if err != nil {
						printf("    %s : Unable to open file : %s\n", path, err)
						return
					}

					var fileInfo os.FileInfo
					fileInfo, err = fileHandle.Stat()
					if err != nil {
						printf("    %s : Unable to stat file : %s\n", path, err)
						return
					}

					size := int64(0)
					if fileInfo.Mode().IsRegular() {
						size = fileInfo.Size()
					} else if fileInfo.Mode().IsDir() {
						printf("    %s : Uploading directories is only supported in archive mode\n", path)
						return
					} else {
						printf("    %s : Unknown file mode : %s\n", path, fileInfo.Mode())
						return
					}

					name := filepath.Base(path)
					file, err := upload(uploadInfo, name, size, fileHandle)
					if err != nil {
						printf("Unable upload file : %s\n", err)
						return
					}

					printFileInformations(uploadInfo, file)
					count++
					totalSize += int(file.CurrentSize)
					uploadInfo.Files[file.Id] = file
				}(path)
			}
			wg.Wait()
		}
	}

	// Comments
	printf("\n\nCommands\n\n")
	for _, file := range uploadInfo.Files {
		printf("%s\n", getFileCommand(uploadInfo, file))
	}
	printf("\n")

	// Upload files
	printf("\nTotal\n\n")
	printf("    %s (%d file(s)) \n\n", bytesToString(totalSize), count)
}

func createUpload(uploadParams *common.Upload) (upload *common.Upload, err error) {
	var Url *url.URL
	Url, err = url.Parse(baseUrl + "/upload")
	if err != nil {
		return
	}

	var j []byte
	j, err = json.Marshal(uploadParams)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = http.NewRequest("POST", Url.String(), bytes.NewBuffer(j))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ClientApp", "cli_client")
	req.Header.Set("Referer", config.Config.Url)

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	basicAuth = resp.Header.Get("Authorization")

	// Parse Json
	upload = new(common.Upload)
	err = json.Unmarshal(body, upload)
	if err != nil {
		return
	}

	return
}

func upload(uploadInfo *common.Upload, name string, size int64, reader io.Reader) (file *common.File, err error) {
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)

	// TODO Handler error properly here
	go func() error {
		part, err := multipartWriter.CreateFormFile("file", name)
		if err != nil {
			fmt.Println(err)
			return pipeWriter.CloseWithError(err)
		}

		var multiWriter io.Writer

		if config.Config.Quiet {
			multiWriter = part
		} else {
			bar := pb.New64(size).SetUnits(pb.U_BYTES)
			bar.ShowSpeed = true

			multiWriter = io.MultiWriter(part, bar)
			bar.Start()
			defer bar.Finish()
		}

		if config.Config.Secure {
			err = cryptoBackend.Encrypt(reader, multiWriter)
			if err != nil {
				fmt.Println(err)
				return pipeWriter.CloseWithError(err)
			}
		} else {
			_, err = io.Copy(multiWriter, reader)
			if err != nil {
				fmt.Println(err)
				return pipeWriter.CloseWithError(err)
			}
		}

		err = multipartWriter.Close()
		return pipeWriter.CloseWithError(err)
	}()

	var Url *url.URL
	Url, err = url.Parse(baseUrl + "/upload/" + uploadInfo.Id + "/file")
	if err != nil {
		return
	}

	var req *http.Request
	req, err = http.NewRequest("POST", Url.String(), pipeReader)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Set("X-ClientApp", "cli_client")
	req.Header.Set("X-UploadToken", uploadInfo.UploadToken)

	if uploadInfo.ProtectedByPassword {
		req.Header.Set("Authorization", basicAuth)
	}

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Parse Json
	file = new(common.File)
	err = json.Unmarshal(responseBody, file)
	if err != nil {
		return
	}

	config.Debug(fmt.Sprintf("Uploaded %s : %s", name, config.Sdump(file)))
	return
}

func getFileCommand(upload *common.Upload, file *common.File) (command string) {

	// Step one - Downloading file
	switch config.Config.DownloadBinary {
	case "wget":
		command += "wget -q -O-"
	case "curl":
		command += "curl -s"
	default:
		command += config.Config.DownloadBinary
	}

	command += fmt.Sprintf(" %s/file/%s/%s/%s", baseUrl, upload.Id, file.Id, file.Name)

	// If Ssl
	if config.Config.Secure {
		command += fmt.Sprintf(" | %s", cryptoBackend.Comments())
	}

	// If archive
	if config.Config.Archive {
		if config.Config.ArchiveMethod == "zip" {
			command += fmt.Sprintf(" > %s", file.Name)
		} else {
			command += fmt.Sprintf(" | %s", archiveBackend.Comments())
		}
	} else {
		command += " > " + file.Name
	}

	return
}

func printf(format string, args ...interface{}) {
	if !config.Config.Quiet {
		fmt.Printf(format, args...)
	}
}

func printFileInformations(upload *common.Upload, file *common.File) {
	var line string
	if !config.Config.Quiet {
		line += "    "
	}

	line += fmt.Sprintf("%s/file/%s/%s/%s", baseUrl, upload.Id, file.Id, file.Name)
	fmt.Println(line)
}

func bytesToString(size int) string {
	if size <= 1024 {
		return fmt.Sprintf("%.2f B", float64(size))
	} else if size <= 1024*1024 {
		return fmt.Sprintf("%.2f kB", float64(size)/float64(1024))
	} else {
		return fmt.Sprintf("%.2f MB", float64(size)/float64((1024*1024)))
	}

	return fmt.Sprintf("%.2f GB", float64(size)/float64((1024*1024*1024)))
}
