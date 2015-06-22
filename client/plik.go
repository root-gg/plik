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
	"errors"
	"fmt"
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

	"github.com/root-gg/plik/client/Godeps/_workspace/src/github.com/cheggaaa/pb"
	docopt "github.com/root-gg/plik/client/Godeps/_workspace/src/github.com/docopt/docopt-go"
	"github.com/root-gg/plik/client/Godeps/_workspace/src/github.com/kardianos/osext"
	"github.com/root-gg/plik/client/Godeps/_workspace/src/github.com/olekukonko/ts"
	"github.com/root-gg/plik/client/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/client/config"
	"github.com/root-gg/plik/server/common"
)

// Vars
var arguments map[string]interface{}
var transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
var client = http.Client{Transport: transport}
var basicAuth string
var err error

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
  -o, --oneshot             Enable OneShot ( Each file will be deleted on first download )
  -r, --removable           Enable Removable upload ( Each file can be deleted by anyone at anymoment )
  -S, --stream              Enable Streaming ( It will block until remote user starts downloading )
  -t, --ttl TTL             Time before expiration (Upload will be removed in m|h|d)
  -n, --name NAME           Set file name when piping from STDIN
  --server SERVER           Overrides plik url
  --comments COMMENT        Set comments of the upload ( MarkDown compatible )
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
  --update                  Update client
`
	// Parse command line arguments
	arguments, _ = docopt.Parse(usage, nil, true, "", false)

	// Unmarshal arguments in configuration
	err = config.UnmarshalArgs(arguments)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	// Check client version
	err = updateClient(arguments["--update"].(bool))
	if err != nil {
		printf("Unable to update Plik client : %s\n", err)
		os.Exit(1)
	}

	// Create upload
	config.Debug("Sending upload params : " + config.Sdump(config.Upload))
	uploadInfo, err := createUpload(config.Upload)
	if err != nil {
		printf("Unable to create upload : %s\n", err)
		os.Exit(1)
	}
	config.Debug("Got upload info : " + config.Sdump(uploadInfo))

	printf("Upload successfully created : \n\n")
	printf("    %s/#/?id=%s\n\n\n", config.Config.URL, uploadInfo.ID)

	// Match file id from server using client reference
	for _, clientFile := range config.Files {
		for _, serverFile := range uploadInfo.Files {
			if clientFile.Reference == serverFile.Reference {
				clientFile.ID = serverFile.ID
				break
			}
		}
	}

	if config.Config.Archive {
		pipeReader, pipeWriter := io.Pipe()
		err = config.GetArchiveBackend().Archive(arguments["FILE"].([]string), pipeWriter)
		if err != nil {
			printf("Unable to archive files : %s\n", err)
			os.Exit(1)
		}

		_, err = upload(uploadInfo, config.Files[0], pipeReader)
		if err != nil {
			printf("Unable to upload archive : %s\n", err)
			return
		}
		pipeReader.CloseWithError(err)

	} else {
		if len(config.Files) == 0 {
			file, err := upload(uploadInfo, config.Files[0], os.Stdin)
			if err != nil {
				printf("Unable to upload from STDIN : %s\n", err)
				return
			}

			uploadInfo.Files[file.ID] = file
		} else {
			// Upload individual files
			var wg sync.WaitGroup
			for _, fileToUpload := range config.Files {
				wg.Add(1)
				go func(fileToUpload *config.FileToUpload) {
					defer wg.Done()

					file, err := upload(uploadInfo, fileToUpload, fileToUpload.FileHandle)
					if err != nil {
						printf("Unable upload file : %s\n", err)
						return
					}

					uploadInfo.Files[file.ID] = file
				}(fileToUpload)
			}
			wg.Wait()
		}
	}

	// Comments
	if !uploadInfo.Stream {
		var totalSize int64
		printf("\n\nCommands\n\n")
		for _, file := range uploadInfo.Files {

			// Increment size
			totalSize += file.CurrentSize

			// Print file informations (only url if quiet mode enabled)
			if config.Config.Quiet {
				fmt.Println(getFileURL(uploadInfo, file))
			} else {
				fmt.Println(getFileCommand(uploadInfo, file))
			}
		}
		printf("\n")

		// Upload files
		printf("\nTotal\n\n")
		printf("    %s (%d file(s)) \n\n", utils.BytesToString(int(totalSize)), len(uploadInfo.Files))
	}
}

func createUpload(uploadParams *common.Upload) (upload *common.Upload, err error) {
	var URL *url.URL
	URL, err = url.Parse(config.Config.URL + "/upload")
	if err != nil {
		return
	}

	var j []byte
	j, err = json.Marshal(uploadParams)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = http.NewRequest("POST", URL.String(), bytes.NewBuffer(j))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ClientApp", "cli_client")
	req.Header.Set("Referer", config.Config.URL)

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

	// Parse Json error
	if resp.StatusCode != 200 {
		result := new(common.Result)
		err = json.Unmarshal(body, result)
		if err == nil && result.Message != "" {
			err = errors.New(result.Message)
		} else {
			err = fmt.Errorf("HTTP error %d %s", resp.StatusCode, resp.Status)
		}
		return
	}

	basicAuth = resp.Header.Get("Authorization")

	// Parse Json response
	upload = new(common.Upload)
	err = json.Unmarshal(body, upload)
	if err != nil {
		return
	}

	return
}

func upload(uploadInfo *common.Upload, fileToUpload *config.FileToUpload, reader io.Reader) (file *common.File, err error) {
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)

	if uploadInfo.Stream {
		fmt.Printf("%s\n", getFileCommand(uploadInfo, fileToUpload.File))
	}

	// TODO Handler error properly here
	go func() error {

		part, err := multipartWriter.CreateFormFile("file", fileToUpload.Name)
		if err != nil {
			fmt.Println(err)
			return pipeWriter.CloseWithError(err)
		}

		var multiWriter io.Writer

		if config.Config.Quiet {
			multiWriter = part
		} else {
			bar := pb.New64(fileToUpload.CurrentSize).SetUnits(pb.U_BYTES)
			bar.Prefix(fmt.Sprintf("%-"+strconv.Itoa(config.GetLongestFilename())+"s : ", fileToUpload.Name))
			bar.ShowSpeed = true
			bar.ShowFinalTime = false
			bar.SetWidth(100)
			bar.SetMaxWidth(100)
			multiWriter = io.MultiWriter(part, bar)
			bar.Start()
			defer bar.Finish()
		}

		if config.Config.Secure {
			err = config.GetCryptoBackend().Encrypt(reader, multiWriter)
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

	var URL *url.URL
	URL, err = url.Parse(config.Config.URL + "/upload/" + uploadInfo.ID + "/file/" + fileToUpload.ID)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = http.NewRequest("POST", URL.String(), pipeReader)
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Parse Json error
	if resp.StatusCode != 200 {
		result := new(common.Result)
		err = json.Unmarshal(body, result)
		if err == nil && result.Message != "" {
			err = errors.New(result.Message)
		} else {
			err = fmt.Errorf("HTTP error %d %s", resp.StatusCode, resp.Status)
		}
		return
	}

	// Parse Json response
	file = new(common.File)
	err = json.Unmarshal(body, file)
	if err != nil {
		return
	}

	config.Debug(fmt.Sprintf("Uploaded %s : %s", file.Name, config.Sdump(file)))
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

	command += fmt.Sprintf(` "%s/file/%s/%s/%s"`, config.Config.URL, upload.ID, file.ID, file.Name)

	// If Ssl
	if config.Config.Secure {
		command += fmt.Sprintf(" | %s", config.GetCryptoBackend().Comments())
	}

	// If archive
	if config.Config.Archive {
		if config.Config.ArchiveMethod == "zip" {
			command += fmt.Sprintf(" > %s", file.Name)
		} else {
			command += fmt.Sprintf(" | %s", config.GetArchiveBackend().Comments())
		}
	} else {
		command += " > " + file.Name
	}

	return
}

func getFileURL(upload *common.Upload, file *common.File) (fileURL string) {
	fileURL += fmt.Sprintf("%s/file/%s/%s/%s", config.Config.URL, upload.ID, file.ID, file.Name)
	return
}

func updateClient(force bool) (err error) {
	if !(config.Config.AutoUpdate || force) {
		return
	}

	// Get client MD5SUM
	path, err := osext.Executable()
	if err != nil {
		return
	}
	MD5Sum, err := utils.FileMd5sum(path)
	if err != nil {
		return
	}

	// Get client architechture
	arch := runtime.GOOS + "-" + runtime.GOARCH
	binary := "plik"
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	// Get last client MD5Sum
	baseURL := config.Config.URL + "/clients/" + arch
	var URL *url.URL
	URL, err = url.Parse(baseURL + "/MD5SUM")
	if err != nil {
		return
	}
	var req *http.Request
	req, err = http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		return
	}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = errors.New("Unable to get last MD5Sum : " + resp.Status)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	lastMD5Sum := utils.Chomp(string(body))

	// Check if the client is up to date
	if MD5Sum == lastMD5Sum {
		if force {
			println("Plik client is up to date")
			os.Exit(0)
		}
		return
	}
	fmt.Printf("Plik client is not up to date, do you want to update ? [Y/n] ")
	input := "y"
	fmt.Scanln(&input)
	strings.ToLower(input)
	if !strings.HasPrefix(input, "y") {
		if force {
			os.Exit(0)
		}
		return
	}

	// Download new client
	tmpPath := filepath.Dir(path) + "/" + "." + filepath.Base(path) + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return
	}
	defer tmpFile.Close()
	URL, err = url.Parse(baseURL + "/" + binary)
	if err != nil {
		return
	}
	req, err = http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		return
	}
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = errors.New("Unable to get last client : " + resp.Status)
		return
	}
	defer resp.Body.Close()
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return
	}

	// Check new MD5sum
	MD5Sum, err = utils.FileMd5sum(tmpPath)
	if err != nil {
		return
	}
	if MD5Sum != lastMD5Sum {
		err = fmt.Errorf("Invalid client MD5Sum %s does not match %s", MD5Sum, lastMD5Sum)
		return
	}

	// Replace old client
	err = os.Rename(tmpPath, path)
	if err != nil {
		return
	}

	println("Plik client sucessfully updated")
	os.Exit(0)
	return
}

func printf(format string, args ...interface{}) {
	if !config.Config.Quiet {
		fmt.Printf(format, args...)
	}
}
