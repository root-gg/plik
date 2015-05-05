/* The MIT License (MIT)

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
THE SOFTWARE. */

package main

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data_backend"
	"github.com/root-gg/plik/server/metadata_backend"
	"github.com/root-gg/plik/server/shorten_backend"
	"github.com/root-gg/utils"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var log *logger.Logger

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())

	log = common.Log()

	var configFile = flag.String("config", "plikd.cfg", "Configuration file (default: plikd.cfg")
	var version = flag.Bool("version", false, "Show version of plikd")
	flag.Parse()
	if *version {
		fmt.Printf("Plikd v%s\n", common.PlikVersion)
		os.Exit(0)
	}

	common.LoadConfiguration(*configFile)
	log.Infof("Starting plikd server v" + common.PlikVersion)

	// Initialize all backends
	metadata_backend.Initialize()
	data_backend.Initialize()
	shorten_backend.Initialize()

	// HTTP Api routes configuration
	r := mux.NewRouter()
	r.HandleFunc("/upload", createUploadHandler).Methods("POST")
	r.HandleFunc("/upload/{uploadid}", getUploadHandler).Methods("GET")
	r.HandleFunc("/upload/{uploadid}/file", addFileHandler).Methods("POST")
	r.HandleFunc("/upload/{uploadid}/file/{fileid}", getFileHandler).Methods("GET")
	r.HandleFunc("/upload/{uploadid}/file/{fileid}", removeFileHandler).Methods("DELETE")
	r.HandleFunc("/file/{uploadid}/{fileid}/{filename}", getFileHandler).Methods("GET", "HEAD")
	r.HandleFunc("/file/{uploadid}/{fileid}/{filename}/yubikey/{yubikey}", getFileHandler).Methods("GET")
	r.PathPrefix("/clients/").Handler(http.StripPrefix("/clients/", http.FileServer(http.Dir("../clients"))))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	http.Handle("/", r)

	go UploadsCleaningRoutine()

	// Start HTTP server
	go func() {
		var err error
		if common.Config.SslEnabled {
			address := common.Config.ListenAddress + ":" + strconv.Itoa(common.Config.ListenPort)
			tlsConfig := &tls.Config{MinVersion: tls.VersionTLS10}
			server := &http.Server{Addr: address, Handler: r, TLSConfig: tlsConfig}
			err = server.ListenAndServeTLS(common.Config.SslCert, common.Config.SslKey)
		} else {
			err = http.ListenAndServe(common.Config.ListenAddress+":"+strconv.Itoa(common.Config.ListenPort), nil)
		}

		if err != nil {
			log.Fatalf("Unable to start HTTP server : %s", err)
		}
	}()

	// Handle signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	for {
		select {
		case s := <-c:
			log.Infof("Got signal : %s", s)
			os.Exit(0)
		}
	}
}

/*
 * HTTP HANDLERS
 */

func createUploadHandler(resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx := common.NewPlikContext("create upload handler", req)
	defer ctx.Finalize(err)

	upload := common.NewUpload()
	ctx.SetUpload(upload.Id)

	// Read request body
	defer req.Body.Close()
	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ctx.Warningf("Unable to read request body : %s", err)
		http.Error(resp, common.NewResult("Unable to read request body", nil).ToJsonString(), 500)
		return
	}

	// Deserialize json body
	if len(body) > 0 {
		err = json.Unmarshal(body, upload)
		if err != nil {
			ctx.Warningf("Unable to deserialize request body : %s", err)
			http.Error(resp, common.NewResult("Unable to deserialize json request body", nil).ToJsonString(), 500)
			return
		}
	}

	// Set upload id, creation date, upload token, ...
	upload.Create()
	ctx.SetUpload(upload.Id)
	upload.RemoteIp = req.RemoteAddr
	uploadToken := upload.UploadToken

	// TTL = Time in second before the upload expiration
	// 0 	-> No ttl specified : default value from configuration
	// -1	-> No expiration : checking with configuration if that's ok
	switch upload.Ttl {
	case 0:
		upload.Ttl = common.Config.DefaultTtl
	case -1:
		if common.Config.MaxTtl != 0 {
			ctx.Warningf("Cannot set infinite ttl (maximum allowed is : %d)", common.Config.MaxTtl)
			http.Error(resp, common.NewResult(fmt.Sprintf("Cannot set infinite ttl (maximum allowed is : %d)", common.Config.MaxTtl), nil).ToJsonString(), 500)
			return
		}
	default:
		if upload.Ttl < 0 {
			ctx.Warningf("Invalid value for ttl : %d", upload.Ttl)
			http.Error(resp, common.NewResult(fmt.Sprintf("Invalid value for ttl : %d", upload.Ttl), nil).ToJsonString(), 500)
			return
		}
		if common.Config.MaxTtl != 0 && upload.Ttl > common.Config.MaxTtl {
			ctx.Warningf("Cannot set ttl to %d (maximum allowed is : %d)", upload.Ttl, common.Config.MaxTtl)
			http.Error(resp, common.NewResult(fmt.Sprintf("Cannot set ttl to %d (maximum allowed is : %d)", upload.Ttl, common.Config.MaxTtl), nil).ToJsonString(), 500)
			return
		}
	}

	// Protect upload with HTTP basic auth
	// Add Authorization header to the response for convenience
	// So clients can just copy this header into the next request
	if upload.Password != "" {
		upload.ProtectedByPassword = true
		if upload.Login == "" {
			upload.Login = "plik"
		}

		// The Authorization header will contain the base64 version of "login:password"
		// Save only the md5sum of this string to authenticate further requests
		b64str := base64.StdEncoding.EncodeToString([]byte(upload.Login + ":" + upload.Password))
		upload.Password, err = utils.Md5sum(b64str)
		if err != nil {
			ctx.Warningf("Unable to generate password hash : %s", err)
			http.Error(resp, common.NewResult("Unable to generate password hash", nil).ToJsonString(), 500)
			return
		}
		resp.Header().Add("Authorization", "Basic "+b64str)
	}

	// Check the token validity with api.yubico.com
	// Only the Yubikey id part of the token is stored
	// The yubikey id is the 12 first characters of the token
	// The 32 lasts characters are the actual OTP
	if upload.Yubikey != "" {
		upload.ProtectedByYubikey = true

		if !common.Config.YubikeyEnabled {
			ctx.Warningf("Got a yubikey upload but Yubikey backend is disabled")
			http.Error(resp, common.NewResult("Yubikey are disabled on this server", nil).ToJsonString(), 500)
			return
		}

		_, ok, err := common.Config.YubiAuth.Verify(upload.Yubikey)
		if err != nil {
			ctx.Warningf("Unable to validate yubikey token : %s", err)
			http.Error(resp, common.NewResult("Unable to validate yubikey token", nil).ToJsonString(), 500)
			return
		}

		if !ok {
			ctx.Warningf("Invalid yubikey token")
			http.Error(resp, common.NewResult("Invalid yubikey token", nil).ToJsonString(), 500)
			return
		}

		upload.Yubikey = upload.Yubikey[:12]
	}

	// A short url is created for each upload if a shorten backend is specified in the configuration.
	// Referer header is used to get the url of incoming request, clients have to set it in order
	// to get this feature working
	if shorten_backend.GetShortenBackend() != nil {
		if req.Header.Get("Referer") != "" {
			u, err := url.Parse(req.Header.Get("Referer"))
			if err != nil {
				ctx.Warningf("Unable to parse referer url : %s", err)
			}
			longUrl := u.Scheme + "://" + u.Host + "#/?id=" + upload.Id
			shortUrl, err := shorten_backend.GetShortenBackend().Shorten(ctx.Fork("shorten url"), longUrl)
			if err == nil {
				upload.ShortUrl = shortUrl
			} else {
				ctx.Warningf("Unable to shorten url %s : %s", longUrl, err)
			}
		}
	}

	// Save the metadata
	err = metadata_backend.GetMetaDataBackend().Create(ctx.Fork("create metadata"), upload)
	if err != nil {
		ctx.Warningf("Create new upload error : %s", err)
		http.Error(resp, common.NewResult("Invalid yubikey token", nil).ToJsonString(), 500)
		return
	}

	// Remove all private informations (ip, data backend details, ...) before
	// sending metadata back to the client
	upload.Sanitize()

	// Show upload token since its an upload creation
	upload.UploadToken = uploadToken

	// Print upload metadata in the json response.
	var json []byte
	if json, err = utils.ToJson(upload); err != nil {
		ctx.Warningf("Unable to serialize response body : %s", err)
		http.Error(resp, common.NewResult("Unable to serialize response body", nil).ToJsonString(), 500)
	}

	resp.Write(json)
}

func getUploadHandler(resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx := common.NewPlikContext("get upload handler", req)
	defer ctx.Finalize(err)

	// Get the upload id and file id from the url params
	vars := mux.Vars(req)
	uploadId := vars["uploadid"]
	ctx.SetUpload(uploadId)

	// Retrieve upload metadata
	upload, err := metadata_backend.GetMetaDataBackend().Get(ctx.Fork("get metadata"), uploadId)
	if err != nil {
		ctx.Warningf("Upload %s not found : %s", uploadId, err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Upload %s not found", uploadId), nil).ToJsonString(), 404)
		return
	}

	ctx.Infof("Got upload from metadata backend")

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		ctx.Warningf("Unauthorized %s : %s", upload.Id, err)
		return
	}

	// Remove all private informations (ip, data backend details, ...) before
	// sending metadata back to the client
	upload.Sanitize()

	// Print upload metadata in the json response.
	var json []byte
	if json, err = utils.ToJson(upload); err != nil {
		ctx.Warningf("Unable to serialize response body : %s", err)
		http.Error(resp, common.NewResult("Unable to serialize response body", nil).ToJsonString(), 500)
	}
	resp.Write(json)
}

func getFileHandler(resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx := common.NewPlikContext("get file handler", req)
	defer ctx.Finalize(err)

	// Get the upload id and file id from the url params
	vars := mux.Vars(req)
	uploadId := vars["uploadid"]
	fileId := vars["fileid"]
	fileName := vars["filename"]
	if uploadId == "" {
		ctx.Warning("Missing upload id")
		redirect(req, resp, errors.New("Missing upload id"), 404)
		return
	}
	if fileId == "" {
		ctx.Warning("Missing file id")
		redirect(req, resp, errors.New("Missing file id"), 404)
		return
	}
	ctx.SetUpload(uploadId)

	// Get the upload informations from the metadata backend
	upload, err := metadata_backend.GetMetaDataBackend().Get(ctx.Fork("get metadata"), uploadId)
	if err != nil {
		ctx.Warningf("Upload %s not found : %s", uploadId, err)
		redirect(req, resp, errors.New(fmt.Sprintf("Upload %s not found", uploadId)), 404)
		return
	}

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		ctx.Warningf("Unauthorized : %s", err)
		return
	}

	// Test if upload is not expired
	if upload.Ttl != 0 {
		if time.Now().Unix() >= (upload.Creation + int64(upload.Ttl)) {
			ctx.Warningf("Upload is expired since %s", time.Since(time.Unix(upload.Creation, int64(0)).Add(time.Duration(upload.Ttl)*time.Second)).String())
			redirect(req, resp, errors.New(fmt.Sprintf("Upload %s is expired", upload.Id)), 404)
			return
		}
	}

	// Retrieve file using data backend
	if _, ok := upload.Files[fileId]; !ok {
		ctx.Warningf("File %s not found", fileId)
		redirect(req, resp, errors.New(fmt.Sprintf("File %s not found", fileId)), 404)
		return
	}

	file := upload.Files[fileId]
	ctx.SetFile(file.Name)

	// Compare url filename with upload filename
	if file.Name != fileName {
		ctx.Warningf("Invalid filename %s mismatch %s", fileName, file.Name)
		redirect(req, resp, errors.New(fmt.Sprintf("File %s not found", fileName)), 404)
		return
	}

	// If upload has OneShot option, test if file has not been already downloaded once
	if upload.OneShot && file.Status == "downloaded" {
		ctx.Warningf("File %s has already been downloaded in upload %s", file.Name, upload.Id)
		redirect(req, resp, errors.New(fmt.Sprintf("File %s has already been downloaded", file.Name)), 401)
		return
	}

	// If the file is marked as deleted by a previous call, we abort request
	if upload.Removable && file.Status == "removed" {
		ctx.Warningf("File %s has been removed", file.Name)
		redirect(req, resp, errors.New(fmt.Sprintf("File %s has been removed", file.Name)), 404)
		return
	}

	// Check yubikey
	// If upload is yubikey protected, user must send an OTP when he wants to get a file.
	if upload.Yubikey != "" {
		token := vars["yubikey"]
		if token == "" {
			ctx.Warningf("Missing yubikey token")
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}
		if len(token) != 44 {
			ctx.Warningf("Invalid yubikey token : %s", token)
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}
		if token[:12] != upload.Yubikey {
			ctx.Warningf("Invalid yubikey device : %s", token)
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}

		// Error if yubikey is disabled on server, and enabled on upload
		if !common.Config.YubikeyEnabled {
			ctx.Warningf("Got a yubikey upload but Yubikey backend is disabled")
			redirect(req, resp, errors.New("Yubikey are disabled on this server"), 500)
			return
		}

		_, isValid, err := common.Config.YubiAuth.Verify(token)
		if err != nil {
			ctx.Warningf("Failed to validate yubikey token : %s", err)
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}
		if !isValid {
			ctx.Warningf("Invalid yubikey token : %s", token)
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}
	}

	// Set content type and print file
	resp.Header().Set("Content-Type", file.Type)
	resp.Header().Set("Content-Length", strconv.Itoa(int(file.CurrentSize)))

	// If "dl" GET params is set
	// -> Set Content-Disposition header
	// -> The client should download file instead of displaying it
	dl := req.URL.Query().Get("dl")
	if dl != "" {
		resp.Header().Set("Content-Disposition", "attachement; filename="+file.Name)
	} else {
		resp.Header().Set("Content-Disposition", "filename="+file.Name)
	}

	// HEAD Request => Do not print file, user just wants http headers
	// GET  Request => Print file content
	ctx.Infof("Got a %s request", req.Method)

	if req.Method == "GET" {
		// Get file in data backend
		fileReader, err := data_backend.GetDataBackend().GetFile(ctx.Fork("get file"), upload, file.Id)
		if err != nil {
			ctx.Warningf("Failed to get file %s in upload %s : %s", file.Name, upload.Id, err)
			redirect(req, resp, errors.New(fmt.Sprintf("Failed to read file %s", file.Name)), 404)
			return
		}
		defer fileReader.Close()

		// Update metadata if oneShot option is set
		if upload.OneShot {
			file.Status = "downloaded"
			err = metadata_backend.GetMetaDataBackend().AddOrUpdateFile(ctx.Fork("update metadata"), upload, file)
			if err != nil {
				ctx.Warningf("Error while deleting file %s from upload %s metadata : %s", file.Name, upload.Id, err)
			}
		}

		// File is piped directly to http response body without buffering
		_, err = io.Copy(resp, fileReader)
		if err != nil {
			ctx.Warningf("Error while copying file to response : %s", err)
		}

		// Remove file from data backend if oneShot option is set
		if upload.OneShot {
			err = data_backend.GetDataBackend().RemoveFile(ctx.Fork("remove file"), upload, file.Id)
			if err != nil {
				ctx.Warningf("Error while deleting file %s from upload %s : %s", file.Name, upload.Id, err)
				return
			}
		}

		// Remove upload if no files are available
		err = RemoveUploadIfNoFileAvailable(ctx, upload)
		if err != nil {
			ctx.Warningf("Error while checking if upload can be removed : %s", err)
		}
	}
}

func addFileHandler(resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx := common.NewPlikContext("add file handler", req)
	defer ctx.Finalize(err)

	// Get the upload id from the url params
	vars := mux.Vars(req)
	uploadId := vars["uploadid"]
	ctx.SetUpload(uploadId)

	// Get upload metadata
	upload, err := metadata_backend.GetMetaDataBackend().Get(ctx.Fork("get metadata"), uploadId)
	if err != nil {
		ctx.Warningf("Upload metadata not found")
		http.Error(resp, common.NewResult(fmt.Sprintf("Upload %s not found", uploadId), nil).ToJsonString(), 404)
		return
	}

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		ctx.Warningf("Unauthorized : %s", err)
		return
	}

	// Check upload token
	if req.Header.Get("X-UploadToken") != upload.UploadToken {
		ctx.Warningf("Invalid upload token %s", req.Header.Get("X-UploadToken"))
		http.Error(resp, common.NewResult("Invalid upload token in X-UploadToken header", nil).ToJsonString(), 404)
		return
	}

	// Get file handle from multipart request
	var file io.Reader
	var fileName string = ""
	multiPartReader, err := req.MultipartReader()
	if err != nil {
		ctx.Warningf("Failed to get file from multipart request : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Failed to get file from multipart request"), nil).ToJsonString(), 500)
		return
	}

	// Read multipart body until the "file" part
	for {
		part, err_part := multiPartReader.NextPart()
		if err_part == io.EOF {
			break
		}

		if part.FormName() == "file" {
			file = part
			fileName = part.FileName()
			break
		}
	}
	if file == nil {
		ctx.Warning("Missing file from multipart request")
		http.Error(resp, common.NewResult("Missing file from multipart request", nil).ToJsonString(), 400)
	}
	if fileName == "" {
		ctx.Warning("Missing file name from multipart request")
		http.Error(resp, common.NewResult("Missing file name from multipart request", nil).ToJsonString(), 400)
	}

	// Create a new file object
	newFile := common.NewFile()
	newFile.Name = fileName
	newFile.Type = "application/octet-stream"
	ctx.SetFile(fileName)

	// Pipe file data from the request body to a preprocessing goroutine
	//  - Guess content type
	//  - Compute md5sum
	//  - Limit upload size
	preprocessReader, preprocessWriter := io.Pipe()
	md5Hash := md5.New()
	totalBytes := 0
	go func() {
		for {
			buf := make([]byte, 1024)
			bytesRead, err := file.Read(buf)
			if err != nil {
				if err != io.EOF {
					ctx.Warningf("Unable to read data from request body : %s", err)
				}

				preprocessWriter.Close()
				return
			}

			// Detect the content-type using the 512 first bytes
			if totalBytes == 0 {
				newFile.Type = http.DetectContentType(buf)
				ctx.Infof("Got Content-Type : %s", newFile.Type)
			}

			// Increment size
			totalBytes += bytesRead

			// Compute md5sum
			md5Hash.Write(buf[:bytesRead])

			// Check upload max size limit
			if totalBytes > common.Config.MaxFileSize {
				err = ctx.EWarningf("File too big (limit is set to %d bytes)", common.Config.MaxFileSize)
				preprocessWriter.CloseWithError(err)
				return
			}

			// Pass file data to data backend
			preprocessWriter.Write(buf[:bytesRead])
		}
	}()

	// Save file in the data backend
	backendDetails, err := data_backend.GetDataBackend().AddFile(ctx.Fork("save file"), upload, newFile, preprocessReader)
	if err != nil {
		ctx.Warningf("Unable to save file : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Error saving file %s in upload %s : %s", newFile.Name, upload.Id, err), nil).ToJsonString(), 500)
		return
	}

	// Fill-in file informations
	newFile.CurrentSize = int64(totalBytes)
	newFile.Status = "uploaded"
	newFile.Md5 = fmt.Sprintf("%x", md5Hash.Sum(nil))
	newFile.UploadDate = time.Now().Unix()
	newFile.BackendDetails = backendDetails

	// Update upload metadata
	upload.Files[newFile.Id] = newFile
	err = metadata_backend.GetMetaDataBackend().AddOrUpdateFile(ctx.Fork("update metadata"), upload, newFile)
	if err != nil {
		ctx.Warningf("Unable to update metadata : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Error adding file %s to upload %s metadata", newFile.Name, upload.Id, err), nil).ToJsonString(), 500)
		return
	}

	// Remove all private informations (ip, data backend details, ...) before
	// sending metadata back to the client
	newFile.Sanitize()

	// Print file metadata in the json response.
	var json []byte
	if json, err = utils.ToJson(newFile); err == nil {
		resp.Write(json)
	} else {
		http.Error(resp, common.NewResult("Unable to serialize response body", nil).ToJsonString(), 500)
	}
}

func removeFileHandler(resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx := common.NewPlikContext("remove file handler", req)
	defer ctx.Finalize(err)

	// Get the upload id and file id from the url params
	vars := mux.Vars(req)
	uploadId := vars["uploadid"]
	fileId := vars["fileid"]
	if uploadId == "" {
		ctx.Warning("Missing upload id")
		redirect(req, resp, errors.New("Missing upload id"), 404)
		return
	}
	if fileId == "" {
		ctx.Warning("Missing file id")
		redirect(req, resp, errors.New("Missing file id"), 404)
		return
	}
	ctx.SetUpload(uploadId)

	// Retrieve Upload
	upload, err := metadata_backend.GetMetaDataBackend().Get(ctx.Fork("get metadata"), uploadId)
	if err != nil {
		ctx.Warning("Upload not found")
		http.Error(resp, common.NewResult(fmt.Sprintf("Upload not %s found", uploadId), nil).ToJsonString(), 404)
		return
	}

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		ctx.Warningf("Unauthorized : %s", err)
		return
	}

	// Test if upload is removable
	if !upload.Removable {
		ctx.Warningf("User tried to remove file %s of an non removeable upload", fileId)
		redirect(req, resp, errors.New(fmt.Sprintf("Can't remove files on this upload", uploadId)), 401)
		return
	}

	// Retrieve file informations in upload
	file, ok := upload.Files[fileId]
	if !ok {
		ctx.Warningf("File not found")
		http.Error(resp, common.NewResult(fmt.Sprintf("File %s not found in upload %s", fileId, upload.Id), nil).ToJsonString(), 404)
		return
	}

	// Set status to removed, and save metadatas
	file.Status = "removed"
	if err := metadata_backend.GetMetaDataBackend().AddOrUpdateFile(ctx.Fork("update metadata"), upload, file); err != nil {
		ctx.Warningf("Error while updating file metadata : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Error while updating file %s metadata in upload %s", file.Name, upload.Id), nil).ToJsonString(), 500)
		return
	}

	// Remove file from data backend
	if err := data_backend.GetDataBackend().RemoveFile(ctx.Fork("remove file"), upload, file.Id); err != nil {
		ctx.Warningf("Error while deleting file : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Error while deleting file %s in upload %s", file.Name, upload.Id), nil).ToJsonString(), 500)
		return
	}

	// Remove upload if no files anymore
	err = RemoveUploadIfNoFileAvailable(ctx, upload)
	if err != nil {
		ctx.Warningf("Error occured when checking if upload can be removed : %s", err)
	}

	// Print upload metadata in the json response.
	var json []byte
	if json, err = utils.ToJson(upload); err != nil {
		ctx.Warningf("Unable to serialize response body : %s", err)
		http.Error(resp, common.NewResult("Unable to serialize response body", nil).ToJsonString(), 500)
	}
	resp.Write(json)
}

//
//// Misc functions
//
func httpBasicAuth(req *http.Request, resp http.ResponseWriter, upload *common.Upload) (err error) {
	if upload.ProtectedByPassword {
		if req.Header.Get("Authorization") == "" {
			err = errors.New("Missing Authorization header")
		} else {
			// Basic auth Authorization header must be set to
			// "Basic base64("login:password")". Only the md5sum
			// of the base64 string is saved in the upload metadata
			auth := strings.Split(req.Header.Get("Authorization"), " ")
			if len(auth) != 2 {
				err = errors.New(fmt.Sprintf("Inavlid Authorization header %s", req.Header.Get("Authorization")))
			}
			if auth[0] != "Basic" {
				err = errors.New(fmt.Sprintf("Inavlid http authorization scheme : %s", auth[0]))
			}
			var md5sum string
			md5sum, err = utils.Md5sum(auth[1])
			if err != nil {
				err = errors.New(fmt.Sprintf("Unable to hash credentials : %s", err))
			}
			if md5sum != upload.Password {
				err = errors.New(fmt.Sprintf("Invalid credentials"))
			}
		}
		if err != nil {
			// WWW-Authenticate header tells the client to retry the request
			// with valid http basic credentials set in the Authorization headers.
			resp.Header().Set("WWW-Authenticate", "Basic realm=\"plik\"")
			http.Error(resp, "Please provide valid credentials to download this file", 401)
		}
	}
	return
}

var userAgents []string = []string{"wget", "curl", "python-urllib", "libwwww-perl", "php", "pycurl"}

func redirect(req *http.Request, resp http.ResponseWriter, err error, status int) {
	// The web client uses http redirect to get errors
	// from http redirect and display a nice HTML error message
	// But cli clients needs a clean string response
	userAgent := strings.ToLower(req.UserAgent())
	for _, ua := range userAgents {
		if strings.HasPrefix(userAgent, ua) {
			http.Error(resp, err.Error(), status)
			return
		}
	}
	http.Redirect(resp, req, fmt.Sprintf("/#/?err=%s&errcode=%d&uri=%s", err.Error(), status, req.RequestURI), 301)
	return
}

// Periodicaly remove expired uploads
func UploadsCleaningRoutine() {
	ctx := common.RootContext().Fork("clean expired uploads")

	for {

		// Sleep between 2 hours and 3 hours
		// This is a dirty trick to avoid frontends doing this at the same time
		randSleep := rand.Intn(3600) + 7200
		log.Infof("Will clean old uploads in %d seconds.", randSleep)
		time.Sleep(time.Duration(randSleep) * time.Second)

		// Get uploads that needs to be removed
		log.Infof("Cleaning expired uploads...")

		uploadsId, err := metadata_backend.GetMetaDataBackend().GetUploadsToRemove(ctx)
		if err != nil {
			log.Warningf("Failed to get expired uploads : %s")
		} else {

			// Remove them
			for _, uploadId := range uploadsId {
				ctx.SetUpload(uploadId)
				log.Infof("Removing expired upload %s", uploadId)
				// Get upload metadata
				childCtx := ctx.Fork("get metadata")
				childCtx.AutoDetach()
				upload, err := metadata_backend.GetMetaDataBackend().Get(childCtx, uploadId)
				if err != nil {
					log.Warningf("Unable to get infos for upload: %s", err)
					continue
				}

				// Remove from data backend
				childCtx = ctx.Fork("remove upload data")
				childCtx.AutoDetach()
				err = data_backend.GetDataBackend().RemoveUpload(childCtx, upload)
				if err != nil {
					log.Warningf("Unable to remove upload data : %s", err)
					continue
				}

				// Remove from metadata backend
				childCtx = ctx.Fork("remove upload metadata")
				childCtx.AutoDetach()
				err = metadata_backend.GetMetaDataBackend().Remove(childCtx, upload)
				if err != nil {
					log.Warningf("Unable to remove upload metadata : %s", err)
				}
			}
		}
	}
}

// Removing upload if there is no available files
func RemoveUploadIfNoFileAvailable(ctx *common.PlikContext, upload *common.Upload) (err error) {

	// Test if there are remaining files
	filesInUpload := len(upload.Files)
	for _, f := range upload.Files {
		if f.Status == "downloaded" {
			filesInUpload--
		}
	}

	if filesInUpload == 0 {

		ctx.Debugf("No more files in upload. Removing all informations.")

		err = data_backend.GetDataBackend().RemoveUpload(ctx, upload)
		if err != nil {
			return
		}
		err = metadata_backend.GetMetaDataBackend().Remove(ctx, upload)
		if err != nil {
			return
		}
	}

	return
}
