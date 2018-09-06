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
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb"
	docopt "github.com/docopt/docopt-go"
	"github.com/kardianos/osext"
	"github.com/olekukonko/ts"
	"github.com/root-gg/plik/client/config"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
)

// Vars
var arguments map[string]interface{}
var transport = &http.Transport{
	Proxy:           http.ProxyFromEnvironment,
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
var client = http.Client{Transport: transport}
var basicAuth string
var err error

// Main
func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	ts.GetSize()

	// Load config
	err = config.Load()
	if err != nil {
		fmt.Printf("Unable to load configuration : %s\n", err)
		os.Exit(1)
	}

	// Usage /!\ INDENT THIS WITH SPACES NOT TABS /!\
	usage := `plik

Usage:
  plik [options] [FILE] ...

Options:
  -h --help                 Show this help
  -d --debug                Enable debug mode
  -q --quiet                Enable quiet mode
  -o, --oneshot             Enable OneShot ( Each file will be deleted on first download )
  -r, --removable           Enable Removable upload ( Each file can be deleted by anyone at anymoment )
  -S, --stream              Enable Streaming ( It will block until remote user starts downloading )
  -t, --ttl TTL             Time before expiration (Upload will be removed in m|h|d)
  -n, --name NAME           Set file name when piping from STDIN
  --server SERVER           Overrides plik url
  --token TOKEN             Specify an upload token
  --comments COMMENT        Set comments of the upload ( MarkDown compatible )
  -p                        Protect the upload with login and password ( be prompted )
  --password PASSWD         Protect the upload with "login:password" ( if omitted default login is "plik" )
  -y, --yubikey             Protect the upload with a Yubikey OTP
  -a                        Archive upload using default archive params ( see ~/.plikrc )
  --archive MODE            Archive upload using the specified archive backend : tar|zip
  --compress MODE           [tar] Compression codec : gzip|bzip2|xz|lzip|lzma|lzop|compress|no
  --archive-options OPTIONS [tar|zip] Additional command line options
  -s                        Encrypt upload using the default encryption parameters ( see ~/.plikrc )
  --not-secure              Do not encrypt upload files regardless of the ~/.plikrc configurations
  --secure MODE             Encrypt upload files using the specified crypto backend : openssl|pgp
  --cipher CIPHER           [openssl] Openssl cipher to use ( see openssl help )
  --passphrase PASSPHRASE   [openssl] Passphrase or '-' to be prompted for a passphrase
  --recipient RECIPIENT     [pgp] Set recipient for pgp backend ( example : --recipient Bob )
  --secure-options OPTIONS  [openssl|pgp] Additional command line options
  --update                  Update client
  -v --version              Show client version
`
	// Parse command line arguments
	arguments, _ = docopt.Parse(usage, nil, true, "", false)

	// Unmarshal arguments in configuration
	err = config.UnmarshalArgs(arguments)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// Check client version
	updateFlag := arguments["--update"].(bool)
	err = updateClient(updateFlag)
	if err == nil {
		if updateFlag {
			os.Exit(0)
		}
	} else {
		printf("Unable to update Plik client : \n")
		printf("%s\n", err)
		if updateFlag {
			os.Exit(1)
		}
	}

	// Detect STDIN type
	// --> If from pipe : ok, doing nothing
	// --> If not from pipe, and no files in arguments : printing help
	fi, _ := os.Stdin.Stat()

	if runtime.GOOS != "windows" {
		if (fi.Mode()&os.ModeCharDevice) != 0 && len(arguments["FILE"].([]string)) == 0 {
			fmt.Println(usage)
			os.Exit(0)
		}
	} else {
		if len(arguments["FILE"].([]string)) == 0 {
			fmt.Println(usage)
			os.Exit(0)
		}
	}

	// Create upload
	config.Debug("Sending upload params : " + config.Sdump(config.Upload))
	uploadInfo, err := createUpload(config.Upload)
	if err != nil {
		printf("Unable to create upload\n")
		printf("%s\n", err)
		os.Exit(1)
	}
	config.Debug("Got upload info : " + config.Sdump(uploadInfo))

	// Mon, 02 Jan 2006 15:04:05 MST
	creationDate := time.Unix(uploadInfo.Creation, 0).Format(time.RFC1123)

	// Display upload url
	printf("Upload successfully created at %s : \n", creationDate)
	printf("    %s/#/?id=%s\n\n", config.Config.URL, uploadInfo.ID)

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

		file, err := upload(uploadInfo, config.Files[0], pipeReader)
		if err != nil {
			printf("Unable to upload archive : %s\n", err)
			return
		}
		uploadInfo.Files[file.ID] = file
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
						printf("Unable to upload file : \n")
						printf("%s\n", err)
						return
					}

					uploadInfo.Files[file.ID] = file
				}(fileToUpload)
			}
			wg.Wait()
		}
	}

	// Display commands
	if !uploadInfo.Stream {
		printf("\nCommands : \n")
		for _, file := range uploadInfo.Files {
			// Print file information (only url if quiet mode is enabled)
			if config.Config.Quiet {
				fmt.Println(getFileURL(uploadInfo, file))
			} else {
				fmt.Println(getFileCommand(uploadInfo, file))
			}
		}
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

	resp, err := makeRequest(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
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

	errCh := make(chan error)
	go func(errCh chan error) {
		part, err := multipartWriter.CreateFormFile("file", fileToUpload.Name)
		if err != nil {
			err = fmt.Errorf("Unable to create multipartWriter : %s", err)
			pipeWriter.CloseWithError(err)
			errCh <- err
			return
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
				pipeWriter.CloseWithError(err)
				errCh <- err
				return
			}
		} else {
			_, err = io.Copy(multiWriter, reader)
			if err != nil {
				pipeWriter.CloseWithError(err)
				errCh <- err
				return
			}
		}

		err = multipartWriter.Close()
		if err != nil {
			err = fmt.Errorf("Unable to close multipartWriter : %s", err)
		}

		pipeWriter.CloseWithError(err)
		errCh <- err
	}(errCh)

	mode := "file"
	if uploadInfo.Stream {
		mode = "stream"
	}

	var URL *url.URL
	URL, err = url.Parse(config.Config.URL + "/" + mode + "/" + uploadInfo.ID + "/" + fileToUpload.ID + "/" + fileToUpload.Name)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", URL.String(), pipeReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Set("X-UploadToken", uploadInfo.UploadToken)

	if uploadInfo.ProtectedByPassword {
		req.Header.Set("Authorization", basicAuth)
	}

	resp, err := makeRequest(req)
	if err != nil {
		return nil, err
	}

	err = <-errCh
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse Json response
	file = new(common.File)
	err = json.Unmarshal(body, file)
	if err != nil {
		return nil, err
	}

	config.Debug(fmt.Sprintf("Uploaded %s : %s", file.Name, config.Sdump(file)))

	return file, nil
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

	command += fmt.Sprintf(` "%s"`, getFileURL(upload, file))

	// If Ssl
	if config.Config.Secure {
		command += fmt.Sprintf(" | %s", config.GetCryptoBackend().Comments())
	}

	// If archive
	if config.Config.Archive {
		if config.Config.ArchiveMethod == "zip" {
			command += fmt.Sprintf(` > '%s'`, file.Name)
		} else {
			command += fmt.Sprintf(" | %s", config.GetArchiveBackend().Comments())
		}
	} else {
		command += fmt.Sprintf(` > '%s'`, file.Name)
	}

	return
}

func getFileURL(upload *common.Upload, file *common.File) (fileURL string) {
	mode := "file"
	if upload.Stream {
		mode = "stream"
	}

	var domain string
	if upload.DownloadDomain != "" {
		domain = upload.DownloadDomain
	} else {
		domain = config.Config.URL
	}

	fileURL += fmt.Sprintf("%s/%s/%s/%s/%s", domain, mode, upload.ID, file.ID, file.Name)

	// Parse to get a nice escaped url
	u, err := url.Parse(fileURL)
	if err != nil {
		return ""
	}

	return u.String()
}

func updateClient(updateFlag bool) (err error) {
	// Do not check for update if AutoUpdate is not enabled
	if !updateFlag && !config.Config.AutoUpdate {
		return
	}

	// Do not update when quiet mode is enabled
	if !updateFlag && config.Config.Quiet {
		return
	}

	// Get client MD5SUM
	path, err := osext.Executable()
	if err != nil {
		return
	}
	currentMD5, err := utils.FileMd5sum(path)
	if err != nil {
		return
	}

	// Check server version
	currentVersion := common.GetBuildInfo().Version

	var newVersion string
	var downloadURL string
	var newMD5 string
	var buildInfo *common.BuildInfo

	var URL *url.URL
	URL, err = url.Parse(config.Config.URL + "/version")
	if err != nil {
		err = fmt.Errorf("Unable to get server version : %s", err)
		return
	}
	var req *http.Request
	req, err = http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		err = fmt.Errorf("Unable to get server version : %s", err)
		return
	}

	resp, err := makeRequest(req)
	if resp == nil {
		err = fmt.Errorf("Unable to get server version : %s", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// >=1.1 use BuildInfo from /version

		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}

		// Parse json BuildInfo object
		buildInfo = new(common.BuildInfo)
		err = json.Unmarshal(body, buildInfo)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}

		newVersion = buildInfo.Version
		for _, client := range buildInfo.Clients {
			if client.OS == runtime.GOOS && client.ARCH == runtime.GOARCH {
				newMD5 = client.Md5
				downloadURL = config.Config.URL + "/" + client.Path
				break
			}
		}

		if newMD5 == "" || downloadURL == "" {
			err = fmt.Errorf("Server does not offer a %s-%s client", runtime.GOOS, runtime.GOARCH)
			return
		}
	} else if resp.StatusCode == 404 {
		// <1.1 fallback on MD5SUM file

		baseURL := config.Config.URL + "/clients/" + runtime.GOOS + "-" + runtime.GOARCH
		var URL *url.URL
		URL, err = url.Parse(baseURL + "/MD5SUM")
		if err != nil {
			return
		}
		var req *http.Request
		req, err = http.NewRequest("GET", URL.String(), nil)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}

		resp, err = makeRequest(req)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			err = fmt.Errorf("Unable to get server version : %s", resp.Status)
			return
		}

		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}
		newMD5 = utils.Chomp(string(body))

		binary := "plik"
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		downloadURL = baseURL + "/" + binary
	} else {
		err = fmt.Errorf("Unable to get server version : %s", err)
		return
	}

	// Check if the client is up to date
	if currentMD5 == newMD5 {
		if updateFlag {
			if newVersion != "" {
				printf("Plik client %s is up to date\n", newVersion)
			} else {
				printf("Plik client is up to date\n")
			}
			os.Exit(0)
		}
		return
	}

	// Ask for permission
	if newVersion != "" {
		fmt.Printf("Update Plik client from %s to %s ? [Y/n] ", currentVersion, newVersion)
	} else {
		fmt.Printf("Update Plik client to match server version ? [Y/n] ")
	}
	input := "y"
	fmt.Scanln(&input)
	if !strings.HasPrefix(strings.ToLower(input), "y") {
		if updateFlag {
			os.Exit(0)
		}
		return
	}

	// Display release notes
	if buildInfo != nil && buildInfo.Releases != nil {

		// Find current release
		currentReleaseIndex := -1
		for i, release := range buildInfo.Releases {
			if release.Name == currentVersion {
				currentReleaseIndex = i
			}
		}

		// Find new release
		newReleaseIndex := -1
		for i, release := range buildInfo.Releases {
			if release.Name == newVersion {
				newReleaseIndex = i
			}
		}

		// Find releases between current and new version
		var releases []*common.Release
		if currentReleaseIndex > 0 && newReleaseIndex > 0 && currentReleaseIndex < newReleaseIndex {
			releases = buildInfo.Releases[currentReleaseIndex+1 : newReleaseIndex+1]
		}

		for _, release := range releases {
			// Get release notes from server
			var URL *url.URL
			URL, err = url.Parse(config.Config.URL + "/changelog/" + release.Name)
			if err != nil {
				continue
			}
			var req *http.Request
			req, err = http.NewRequest("GET", URL.String(), nil)
			if err != nil {
				err = fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
				continue
			}

			resp, err = makeRequest(req)
			if err != nil {
				err = fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				err = fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
				continue
			}

			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				err = fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
				continue
			}

			// Ask to display the release notes
			fmt.Printf("Do you want to browse the release notes of version %s ? [Y/n] ", release.Name)
			input := "y"
			fmt.Scanln(&input)
			if !strings.HasPrefix(strings.ToLower(input), "y") {
				continue
			}

			// Display the release notes
			releaseDate := time.Unix(release.Date, 0).Format("Mon Jan 2 2006 15:04")
			fmt.Printf("Plik %s has been released %s\n\n", release.Name, releaseDate)
			fmt.Println(string(body))

			// Let user review the last release notes and ask to confirm update
			if release.Name == newVersion {
				fmt.Printf("\nUpdate Plik client from %s to %s ? [Y/n] ", currentVersion, newVersion)
				input = "y"
				fmt.Scanln(&input)
				if !strings.HasPrefix(strings.ToLower(input), "y") {
					if updateFlag {
						os.Exit(0)
					}
					return
				}
				break
			}
		}
	}

	// Download new client
	tmpPath := filepath.Dir(path) + "/" + "." + filepath.Base(path) + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	URL, err = url.Parse(downloadURL)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	req, err = http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	resp, err = makeRequest(req)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Unable to download client : %s", resp.Status)
		return
	}
	defer resp.Body.Close()
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	err = tmpFile.Close()
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}

	// Check download integrity
	downloadMD5, err := utils.FileMd5sum(tmpPath)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	if downloadMD5 != newMD5 {
		err = fmt.Errorf("Unable to download client : md5sum %s does not match %s", downloadMD5, newMD5)
		return
	}

	// Replace old client
	err = os.Rename(tmpPath, path)
	if err != nil {
		err = fmt.Errorf("Unable to replace client : %s", err)
		return
	}

	if newVersion != "" {
		printf("Plik client successfully updated to %s\n", newVersion)
	} else {
		printf("Plik client successfully updated\n")
	}

	return
}

func makeRequest(req *http.Request) (resp *http.Response, err error) {

	// Set client version headers
	req.Header.Set("X-ClientApp", "cli_client")
	bi := common.GetBuildInfo()
	if bi != nil {
		version := runtime.GOOS + "-" + runtime.GOARCH + "-" + bi.Version
		req.Header.Set("X-ClientVersion", version)
	}

	// Set authentication header
	if config.Config.Token != "" {
		req.Header.Set("X-PlikToken", config.Config.Token)
	}

	// Log request
	if config.Config.Debug {
		dump, err := httputil.DumpRequest(req, true)
		if err == nil {
			config.Debug(string(dump))
		} else {
			printf("Unable to dump HTTP request : %s", err)
		}
	}

	// Make request
	resp, err = client.Do(req)
	if err != nil {
		return
	}

	// Log response
	if config.Config.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			config.Debug(string(dump))
		} else {
			printf("Unable to dump HTTP response : %s", err)
		}
	}

	// Parse Json error
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}

		result := new(common.Result)
		err = json.Unmarshal(body, result)
		if err == nil && result.Message != "" {
			err = fmt.Errorf("%s : %s", resp.Status, result.Message)
		} else if len(body) > 0 {
			err = fmt.Errorf("%s : %s", resp.Status, string(body))
		} else {
			err = fmt.Errorf("%s", resp.Status)
		}
		return
	}

	return
}

func printf(format string, args ...interface{}) {
	if !config.Config.Quiet {
		fmt.Printf(format, args...)
	}
}
