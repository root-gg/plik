/**

    Plik upload client

The MIT License (MIT)

Copyright (c) <2014> <mathieu.bodjikian@ovh.net>

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
	"github.com/root-gg/plik/server/utils"
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
var plikVersion string = "##VERSION##"

var baseUrl string
var transport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}
var client = http.Client{Transport: transport}
var cryptoBackend crypto.CryptoBackend
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
`
	arguments, _ = docopt.Parse(usage, nil, true, "Autoroot v"+plikVersion, false)

	if arguments["--debug"].(bool) {
		config.Config.Debug = true
	}
	config.Debug("Arguments : " + config.Sdump(arguments))
	config.Debug("Configuration : " + config.Sdump(config.Config))

	baseUrl = config.Config.Url
	if arguments["--server"] != nil && arguments["--server"].(string) != "" {
		baseUrl = arguments["--server"].(string)
	}

	uploadInfo := new(utils.Upload)

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
			secureMethod = arguments["--secure-method"].(string)
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
		fmt.Printf("Unable to create upload : %s\n", err)
		os.Exit(1)
	}
	config.Debug("Got upload info : " + config.Sdump(uploadInfo))

	fmt.Printf("Upload successfully created : \n\n")
	fmt.Printf("    %s/#/?id=%s\n\n\n", baseUrl, uploadInfo.Id)

	count := 0
	totalSize := 0
	if arguments["-a"].(bool) || arguments["--archive"] != nil {
		archiveMethod := config.Config.ArchiveMethod
		if arguments["--archive"] != nil && arguments["--archive"] != "" {
			archiveMethod = arguments["--archive"].(string)
		}
		archiveBackend, err := archive.NewArchiveBackend(archiveMethod, config.Config.ArchiveOptions)
		if err != nil {
			fmt.Printf("Invalid archive params : %s\n", err)
			os.Exit(1)
		}
		err = archiveBackend.Configure(arguments)
		if err != nil {
			fmt.Printf("Invalid archive params : %s\n", err)
			os.Exit(1)
		}

		pipeReader, pipeWriter := io.Pipe()
		name, err := archiveBackend.Archive(arguments["FILE"].([]string), pipeWriter)
		if err != nil {
			fmt.Printf("Unable to archive files : %s\n", err)
			os.Exit(1)
		}

		if arguments["--name"] != nil && arguments["--name"].(string) != "" {
			name = arguments["--name"].(string)
		}

		file, err := upload(uploadInfo, name, int64(0), pipeReader)
		if err != nil {
			fmt.Printf("Unable to upload from STDIN : %s\n", err)
			return
		}
		pipeReader.CloseWithError(err)

		//fmt.Println(utils.Dump(file))
		fmt.Printf("    %s/file/%s/%s/%s\n", baseUrl, uploadInfo.Id, file.Id, file.Name)
		count++
		totalSize += int(file.CurrentSize)
	} else {
		if len(arguments["FILE"].([]string)) == 0 {
			// Upload from STDIN
			name := "STDIN"
			if arguments["--name"] != nil && arguments["--name"].(string) != "" {
				name = arguments["--name"].(string)
			}

			file, err := upload(uploadInfo, name, int64(0), os.Stdin)
			if err != nil {
				fmt.Printf("Unable to upload from STDIN : %s\n", err)
				return
			}
			//fmt.Println(utils.Dump(file))
			fmt.Printf("    %s/file/%s/%s/%s\n", baseUrl, uploadInfo.Id, file.Id, file.Name)
			count++
			totalSize += int(file.CurrentSize)
		} else {
			// Upload individual files
			var wg sync.WaitGroup
			for _, path := range arguments["FILE"].([]string) {
				wg.Add(1)
				go func(path string) {
					defer wg.Done()

					fileHandle, err := os.Open(path)
					if err != nil {
						fmt.Printf("    %s : Unable to open file : %s\n", path, err)
						return
					}

					var fileInfo os.FileInfo
					fileInfo, err = fileHandle.Stat()
					if err != nil {
						fmt.Printf("    %s : Unable to stat file : %s\n", path, err)
						return
					}

					size := int64(0)
					if fileInfo.Mode().IsRegular() {
						size = fileInfo.Size()
					} else if fileInfo.Mode().IsDir() {
						fmt.Printf("    %s : Uploading directories is not yet implemented\n", path)
						return
					} else {
						fmt.Printf("    %s : Unknown file mode : %s\n", path, fileInfo.Mode())
						return
					}

					name := filepath.Base(path)
					file, err := upload(uploadInfo, name, size, fileHandle)
					if err != nil {
						fmt.Printf("Unable upload file : %s\n", err)
						return
					}

					fmt.Printf("    %s/file/%s/%s/%s\n", baseUrl, uploadInfo.Id, file.Id, file.Name)
					count++
					totalSize += int(file.CurrentSize)
				}(path)
			}
			wg.Wait()
		}
	}

	// Upload files
	fmt.Printf("\nTotal\n\n")
	fmt.Printf("    %s (%d file(s)) \n\n", bytesToString(totalSize), count)
}

func createUpload(uploadParams *utils.Upload) (upload *utils.Upload, err error) {
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
	upload = new(utils.Upload)
	err = json.Unmarshal(body, upload)
	if err != nil {
		return
	}

	return
}

func upload(uploadInfo *utils.Upload, name string, size int64, reader io.Reader) (file *utils.File, err error) {
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)

	// TODO Handler error properly here
	go func() error {
		part, err := multipartWriter.CreateFormFile("file", name)
		if err != nil {
			fmt.Println(err)
			return pipeWriter.CloseWithError(err)
		}

		bar := pb.New64(size).SetUnits(pb.U_BYTES)
		bar.ShowSpeed = true
		defer bar.Finish()

		multiWriter := io.MultiWriter(part, bar)
		bar.Start()

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
	file = new(utils.File)
	err = json.Unmarshal(responseBody, file)
	if err != nil {
		return
	}

	config.Debug(fmt.Sprintf("Uploaded %s : %s", name, config.Sdump(file)))
	return
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

//	// DocOpt Args
//	arguments, _ := docopt.Parse(usage, nil, true, "Autoroot v"+plikVersion, false)
//
//	for key, value := range arguments {
//		if _, ok := value.(bool); ok {
//			if key == "--oneshot" && value.(bool) {
//				config.OneShot = true
//			} else if key == "-s" {
//				config.EncryptMethod = value.(bool)
//
//				if encryption {
//					encryptionMethod = "symmetric"
//				}
//			} else if key == "--removable" && value.(bool) {
//				config.Removable = value.(bool)
//			} else if key == "--yubikey" {
//				yubikey = value.(bool)
//			}
//		} else if _, ok := value.(string); ok {
//			if key == "--key" {
//				encryption = true
//				encryptionPassphrase = value.(string)
//			} else if key == "--comments" {
//				comments = value.(string)
//			} else if key == "--cipher" {
//				encryptionCipher = value.(string)
//			} else if key == "--server" {
//				config.Url = value.(string)
////			} else if key == "--gpg" {
////				pgpSearchStr = value.(string)
////				pgpEnabled = true
////			} else if key == "--keyring" {
////				pgpKeyringFile = value.(string)
////			} else if key == "--search-keys" {
////				searchGpgKeys(value.(string))
////				return
////			} else if key == "--email" {
////				email = value.(string)
//			} else if key == "--name" {
//				fileNameParam = value.(string)
//			}
//		} else if _, ok := value.([]string); ok {
//			if key == "FILE" {
//				for _, value := range arguments[key].([]string) {
//					filesList = append(filesList, value)
//				}
//			}
//		}
//	}
//
//	// PGP Stuff
////	if pgpEnabled {
////
////		// Stat default keyring
////		_, err := os.Stat(pgpKeyringFile)
////		if err != nil {
////			fmt.Printf("GnuPG Keyring not found on your system ! (%s)\n", pgpKeyringFile)
////			os.Exit(1)
////		}
////
////		// Open it
////		pubringFile, err := os.Open(pgpKeyringFile)
////		if err != nil {
////			fmt.Printf("Fail to open your GnuPG keyring : %s\n", err)
////			os.Exit(1)
////		}
////
////		// Read it
////		pubring, err := openpgp.ReadKeyRing(pubringFile)
////		if err != nil {
////			fmt.Printf("Fail to read your GnuPG keyring : %s\n", err)
////			os.Exit(1)
////		}
////
////		// Search for key
////		entitiesFound := make(map[uint64]*openpgp.Entity)
////		intToEntity := make(map[int]uint64)
////		countEntitiesFound := 0
////
////		for _, entity := range pubring {
////			for _, ident := range entity.Identities {
////				if strings.Contains(ident.UserId.Email, pgpSearchStr) {
////					if _, ok := entitiesFound[entity.PrimaryKey.KeyId]; !ok {
////						entitiesFound[entity.PrimaryKey.KeyId] = entity
////						intToEntity[countEntitiesFound] = entity.PrimaryKey.KeyId
////						countEntitiesFound++
////					}
////				}
////			}
////		}
////
////		if countEntitiesFound == 0 {
////			fmt.Printf("No key found for input : %s in your keyring !\n", pgpSearchStr)
////			os.Exit(1)
////		} else if countEntitiesFound == 1 {
////			pgpEntity = entitiesFound[intToEntity[0]]
////		} else if countEntitiesFound > 1 {
////
////			fmt.Printf("Found %d identities corresponding your search : \n\n", countEntitiesFound)
////			for i, v := range intToEntity {
////
////				fmt.Printf("\t%d : %s\n", i, entitiesFound[v].PrimaryKey.CreationTime)
////
////				for _, ident := range entitiesFound[v].Identities {
////					fmt.Printf("\t\t%-25s -> %s - %s\n", ident.UserId.Email, ident.UserId.Id, ident.UserId.Comment)
////				}
////
////				fmt.Printf("\n")
////			}
////
////			choosenIdentityInteger := 0
////			fmt.Printf("Which one do you choose ? [default=0] : ")
////			fmt.Scanf("%d", &choosenIdentityInteger)
////			if _, ok := intToEntity[choosenIdentityInteger]; ok {
////				pgpEntity = entitiesFound[intToEntity[choosenIdentityInteger]]
////			} else {
////				fmt.Printf("No identity matching number %d ! \n", choosenIdentityInteger)
////				os.Exit(1)
////			}
////		}
////	}
//
//	// Create the upload
//	upload, err := createUpload()
//	if err != nil {
//		fmt.Printf("Error : %s\n", err)
//		return
//	}
//
//	fmt.Printf("Upload successfully created : \n\n")
//	fmt.Printf("    %s/#/?id=%s\n\n\n", config.Url, upload.Id)
//
//	// Upload files...
//	if len(filesList) == 0 {
//		fmt.Printf("Read STDIN \n\n")
//		url, err := addFileToUpload(uploadId, "")
//		if err != nil {
//			fmt.Printf("Error add stdin to upload : %s\n", err)
//			return
//		} else {
//			fmt.Printf("    %s\n", url)
//		}
//	} else {
//		fmt.Printf("Files : \n\n")
//		for index := range filesList {
//			url, err := addFileToUpload(uploadId, filesList[index])
//
//			if err != nil {
//				fmt.Printf("Error adding file to upload : %s\n", err)
//				return
//			} else {
//				fmt.Printf("    %s\n", url)
//			}
//		}
//	}
//	fmt.Printf("\n\nTotal\n\n")
//	fmt.Printf("    %s (%d file(s)) \n\n", bytesToString(totalFilesSize), countUploadedFiles)
//}
//
////
////// Functions
////
//
//func createUpload() (upload *utils.Upload, err error) {
//
//	// Enable insecure upload
//	tr := &http.Transport{
//		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
//	}
//
//	client := &http.Client{Transport: tr}
//
//	// Url
//	var Url *url.URL
//	Url, err := url.Parse(config.Url)
//	if err != nil {
//		return "", err
//	}
//
//	// Params
//	Url.Path += "/upload"
//	upload = utils.NewUpload()
//	upload.OneShot =
//	upload.Removable = config.Removable
//	upload.Comment = comments
//	parameters.Add("oneShot", boolToString(config.OneShot))
//	parameters.Add("removable", boolToString(config.Removable))
//	parameters.Add("comments", comments)
//	parameters.Add("email", email)
//
//	// Ask yubikey if enabled
//	if yubikey {
//		token := ""
//		fmt.Printf("Yubikey token : ")
//		_, err := fmt.Scanf("%s\n", &token)
//		if err != nil {
//		}
//		parameters.Add("yubikeyToken", token)
//		parameters.Add("yubikey", "1")
//	}
//
//	// Specify GPG Key
//	if pgpKeyId != "" {
//		parameters.Add("pgpKeyId", pgpKeyId)
//	}
//
//	// First, we create a http requestuest
//	resp, err := client.PostForm(Url.String(), parameters)
//
//	if err != nil {
//		return "", err
//	}
//
//	defer resp.Body.Close()
//	body, err := ioutil.ReadAll(resp.Body)
//
//	// Parse Json
//	js, err := simplejson.NewJson(body)
//	if err != nil {
//		return "", err
//	} else {
//		status, _ := js.Get("status").Int()
//
//		if status == 100 {
//			uploadId, _ := js.Get("value").Get("id").String()
//			return uploadId, nil
//		} else {
//			message, _ := js.Get("message").String()
//
//			return "", errors.New(message)
//		}
//	}
//
//	return "", nil
//}
//
//func addFileToUpload(uploadId string, fileName string) (string, error) {
//	var file *os.File
//	var fsize int64 = 0
//
//	if fileName == "" {
//		fileName = fileNameParam
//		file = os.Stdin
//	} else {
//		//Open file
//		var err error
//		file, err = os.Open(fileName)
//		if err != nil {
//			return "", err
//		}
//		stat, err := file.Stat()
//		if err != nil {
//			return "", err
//		}
//		fsize = stat.Size()
//	}
//
//	pipeReader, pipeWriter := io.Pipe()
//
//	// Create http requestuest
//	response := multipart.NewWriter(pipeWriter)
//
//	go func() error {
//		part, err := response.CreateFormFile("file", filepath.Base(fileName))
//
//		if err != nil {
//			return pipeWriter.CloseWithError(err)
//		}
//
//		bar := pb.New64(fsize).SetUnits(pb.U_BYTES)
//		bar.ShowSpeed = true
//		defer bar.Finish()
//
//		multiWriter := io.MultiWriter(part, bar)
//		bar.Start()
//
//		if pgpEntity != nil {
//			w, _ := armor.Encode(multiWriter, "PGP MESSAGE", nil)
//			plaintext, _ := openpgp.Encrypt(w, []*openpgp.Entity{pgpEntity}, nil, nil, nil)
//
//			_, err = io.Copy(plaintext, file)
//			if err != nil {
//				return pipeWriter.CloseWithError(err)
//			}
//
//			plaintext.Close()
//			w.Close()
//
//		} else {
//			_, err = io.Copy(multiWriter, file)
//			if err != nil {
//				return pipeWriter.CloseWithError(err)
//			}
//		}
//
//		// POST variables
//		_ = response.WriteField("action", "addFile")
//		_ = response.WriteField("uploadId", uploadId)
//		_ = response.WriteField("oneShot", boolToString(config.OneShot))
//		_ = response.WriteField("removable", boolToString(config.Removable))
//		_ = response.WriteField("encryptMethod", encryptionMethod)
//		_ = response.WriteField("encryptPassphrase", encryptionPassphrase)
//		_ = response.WriteField("encryptCipher", encryptionCipher)
//
//		// Close Writer
//		err = response.Close()
//		return pipeWriter.CloseWithError(err)
//	}()
//	// Create and execute requestuest
//	requestuest, err := http.NewRequest("POST", config.Url+"/auto", pipeReader)
//	if err != nil {
//		return "", err
//	}
//
//	// Add right header
//	requestuest.Header.Add("Content-Type", response.FormDataContentType())
//
//	// Enable insecure upload
//	tr := &http.Transport{
//		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
//	}
//
//	// Go go go
//	client := &http.Client{Transport: tr}
//	resp, err := client.Do(requestuest)
//	if err != nil {
//		return "", err
//	} else {
//		body, err := ioutil.ReadAll(resp.Body)
//
//		resp.Body.Close()
//
//		// Parse Json
//		js, err := simplejson.NewJson(body)
//		if err != nil {
//			return "", err
//		} else {
//
//			// Get status
//			status, _ := js.Get("status").Int()
//
//			if status == 100 {
//				url, _ := js.Get("value").Get("fileUrl").String()
//				size, _ := js.Get("value").Get("fileSize").Int()
//
//				countUploadedFiles += 1
//				totalFilesSize += size
//
//				return url, nil
//			} else {
//				message, _ := js.Get("message").String()
//				return message, nil
//			}
//		}
//	}
//
//	return fileName, nil
//}
//
//func searchGpgKeys(input string) (string, error) {
//
//	// Enable insecure upload
//	tr := &http.Transport{
//		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
//	}
//
//	client := &http.Client{Transport: tr}
//
//	// Url
//	var Url *url.URL
//	Url, err := url.Parse(config.Url)
//	if err != nil {
//		return "", err
//	}
//
//	// Params
//	Url.Path += "/auto"
//	parameters := url.Values{}
//	parameters.Add("action", "getPgpPublicKeys")
//	parameters.Add("input", input)
//	Url.RawQuery = parameters.Encode()
//
//	// First, we create a http requestuest
//	resp, err := client.Get(Url.String())
//
//	if err != nil {
//		return "", err
//	}
//
//	defer resp.Body.Close()
//	body, err := ioutil.ReadAll(resp.Body)
//
//	// Parse Json
//	js, err := simplejson.NewJson(body)
//	if err != nil {
//		return "", err
//	} else {
//		status, _ := js.Get("status").Int()
//
//		if status == 100 {
//			for _, value := range js.Get("value").MustArray() {
//				keyDetail := value.(map[string]interface{})
//
//				keyId := keyDetail["id"].(string)
//				keyEmail := keyDetail["email"].(string)
//				keyName := keyDetail["name"].(string)
//				keyDate := keyDetail["dateCreated"].(string)
//
//				fmt.Printf("%40.40s\t%35.35s\t%30.30s\t%32.32s\n", keyId, keyEmail, keyName, keyDate)
//			}
//		} else {
//			message, _ := js.Get("message").String()
//
//			return "", errors.New(message)
//		}
//	}
//
//	return "", nil
//}
//
//// Misc
//
//func boolToString(b bool) string {
//	if b {
//		return "1"
//	}
//	return "0"
//}
//
//func bytesToString(size int) string {
//	if size <= 1024 {
//		return fmt.Sprintf("%.2f B", float64(size))
//	} else if size <= 1024*1024 {
//		return fmt.Sprintf("%.2f kB", float64(size)/float64(1024))
//	} else {
//		return fmt.Sprintf("%.2f MB", float64(size)/float64((1024*1024)))
//	}
//
//	return fmt.Sprintf("%.2f GB", float64(size)/float64((1024*1024*1024)))
//}
//
//func UserHomeDir() string {
//	if runtime.GOOS == "windows" {
//		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
//		if home == "" {
//			home = os.Getenv("USERPROFILE")
//		}
//		return home
//	}
//	return os.Getenv("HOME")
//}
//
//func GeneratePassphrase() string {
//	rb := make([]byte, 32)
//	rand.Read(rb)
//	return base64.URLEncoding.EncodeToString(rb)
//}
