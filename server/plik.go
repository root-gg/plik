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
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/facebookgo/httpdown"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/logger"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/dataBackend"
	"github.com/root-gg/plik/server/metadataBackend"
	"github.com/root-gg/plik/server/shortenBackend"
)

var log *logger.Logger

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	log = common.Log()

	var configFile = flag.String("config", "plikd.cfg", "Configuration file (default: plikd.cfg")
	var version = flag.Bool("version", false, "Show version of plikd")
	var port = flag.Int("port", 0, "Overrides plik listen port")
	flag.Parse()
	if *version {
		fmt.Printf("Plikd v%s\n", common.GetVersion())
		os.Exit(0)
	}

	common.LoadConfiguration(*configFile)
	log.Infof("Starting plikd server v" + common.GetVersion())

	// Overrides port if provided in command line
	if *port != 0 {
		common.Config.ListenPort = *port
	}

	// Initialize all backends
	metadataBackend.Initialize()
	dataBackend.Initialize()
	shortenBackend.Initialize()

	// Initialize the httpdown wrapper
	hd := &httpdown.HTTP{
		StopTimeout: 5 * time.Minute,
		KillTimeout: 1 * time.Second,
	}

	// HTTP Api routes configuration
	r := mux.NewRouter()
	r.HandleFunc("/config", getConfigurationHandler).Methods("GET")
	r.HandleFunc("/upload", createUploadHandler).Methods("POST")
	r.HandleFunc("/upload/{uploadID}", getUploadHandler).Methods("GET")
	r.HandleFunc("/file/{uploadID}", addFileHandler).Methods("POST")
	r.HandleFunc("/file/{uploadID}/{fileID}/{filename}", addFileHandler).Methods("POST")
	r.HandleFunc("/file/{uploadID}/{fileID}/{filename}", removeFileHandler).Methods("DELETE")
	r.HandleFunc("/file/{uploadID}/{fileID}/{filename}", getFileHandler).Methods("HEAD", "GET")
	r.HandleFunc("/file/{uploadID}/{fileID}/{filename}/yubikey/{yubikey}", getFileHandler).Methods("GET")
	r.HandleFunc("/stream/{uploadID}/{fileID}/{filename}", addFileHandler).Methods("POST")
	r.HandleFunc("/stream/{uploadID}/{fileID}/{filename}", removeFileHandler).Methods("DELETE")
	r.HandleFunc("/stream/{uploadID}/{fileID}/{filename}", getFileHandler).Methods("HEAD", "GET")
	r.HandleFunc("/stream/{uploadID}/{fileID}/{filename}/yubikey/{yubikey}", getFileHandler).Methods("GET")
	r.PathPrefix("/clients/").Handler(http.StripPrefix("/clients/", http.FileServer(http.Dir("../clients"))))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	http.Handle("/", r)

	go UploadsCleaningRoutine()

	// Start HTTP server
	var err error
	var server *http.Server

	address := common.Config.ListenAddress + ":" + strconv.Itoa(common.Config.ListenPort)

	if common.Config.SslEnabled {

		// Load cert
		cert, err := tls.LoadX509KeyPair(common.Config.SslCert, common.Config.SslKey)
		if err != nil {
			log.Fatalf("Unable to load ssl certificate : %s", err)
		}

		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS10, Certificates: []tls.Certificate{cert}}
		server = &http.Server{Addr: address, Handler: r, TLSConfig: tlsConfig}
	} else {
		server = &http.Server{Addr: address, Handler: r}
	}

	err = httpdown.ListenAndServe(server, hd)
	if err != nil {
		log.Fatalf("Unable to start HTTP server : %s", err)
	}

}

/*
 * HTTP HANDLERS
 */

func getConfigurationHandler(resp http.ResponseWriter, req *http.Request) {

	var err error
	var json []byte

	ctx := common.NewPlikContext("get configuration handler", req)
	defer ctx.Finalize(err)

	if json, err = utils.ToJson(common.Config); err != nil {
		ctx.Warningf("Unable to serialize response body : %s", err)
		http.Error(resp, common.NewResult("Unable to serialize response body", nil).ToJSONString(), 500)
	}

	resp.Write(json)
}

func createUploadHandler(resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx := common.NewPlikContext("create upload handler", req)
	defer ctx.Finalize(err)

	// Check that source IP address is valid and whitelisted
	code, err := checkSourceIP(ctx, true)
	if err != nil {
		http.Error(resp, common.NewResult(err.Error(), nil).ToJSONString(), code)
		return
	}

	upload := common.NewUpload()
	ctx.SetUpload(upload.ID)

	// Read request body
	defer req.Body.Close()
	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ctx.Warningf("Unable to read request body : %s", err)
		http.Error(resp, common.NewResult("Unable to read request body", nil).ToJSONString(), 500)
		return
	}

	// Deserialize json body
	if len(body) > 0 {
		err = json.Unmarshal(body, upload)
		if err != nil {
			ctx.Warningf("Unable to deserialize request body : %s", err)
			http.Error(resp, common.NewResult("Unable to deserialize json request body", nil).ToJSONString(), 500)
			return
		}
	}

	// Set upload id, creation date, upload token, ...
	upload.Create()
	ctx.SetUpload(upload.ID)
	upload.RemoteIP = req.RemoteAddr
	uploadToken := upload.UploadToken

	if upload.Stream {
		if !common.Config.StreamMode {
			ctx.Warning("Stream mode is not enabled")
			http.Error(resp, common.NewResult("Stream mode is not enabled", nil).ToJSONString(), 400)
			return
		}
		upload.OneShot = true
	}

	// TTL = Time in second before the upload expiration
	// 0 	-> No ttl specified : default value from configuration
	// -1	-> No expiration : checking with configuration if that's ok
	switch upload.TTL {
	case 0:
		upload.TTL = common.Config.DefaultTTL
	case -1:
		if common.Config.MaxTTL != 0 {
			ctx.Warningf("Cannot set infinite ttl (maximum allowed is : %d)", common.Config.MaxTTL)
			http.Error(resp, common.NewResult(fmt.Sprintf("Cannot set infinite ttl (maximum allowed is : %d)", common.Config.MaxTTL), nil).ToJSONString(), 400)
			return
		}
	default:
		if upload.TTL < 0 {
			ctx.Warningf("Invalid value for ttl : %d", upload.TTL)
			http.Error(resp, common.NewResult(fmt.Sprintf("Invalid value for ttl : %d", upload.TTL), nil).ToJSONString(), 400)
			return
		}
		if common.Config.MaxTTL != 0 && upload.TTL > common.Config.MaxTTL {
			ctx.Warningf("Cannot set ttl to %d (maximum allowed is : %d)", upload.TTL, common.Config.MaxTTL)
			http.Error(resp, common.NewResult(fmt.Sprintf("Cannot set ttl to %d (maximum allowed is : %d)", upload.TTL, common.Config.MaxTTL), nil).ToJSONString(), 400)
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
			http.Error(resp, common.NewResult("Unable to generate password hash", nil).ToJSONString(), 500)
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
			ctx.Warningf("Got a Yubikey upload but Yubikey backend is disabled")
			http.Error(resp, common.NewResult("Yubikey are disabled on this server", nil).ToJSONString(), 500)
			return
		}

		_, ok, err := common.Config.YubiAuth.Verify(upload.Yubikey)
		if err != nil {
			ctx.Warningf("Unable to validate yubikey token : %s", err)
			http.Error(resp, common.NewResult("Unable to validate yubikey token", nil).ToJSONString(), 500)
			return
		}

		if !ok {
			ctx.Warningf("Invalid yubikey token")
			http.Error(resp, common.NewResult("Invalid yubikey token", nil).ToJSONString(), 401)
			return
		}

		upload.Yubikey = upload.Yubikey[:12]
	}

	// A short url is created for each upload if a shorten backend is specified in the configuration.
	// Referer header is used to get the url of incoming request, clients have to set it in order
	// to get this feature working
	if shortenBackend.GetShortenBackend() != nil {
		if req.Header.Get("Referer") != "" {
			u, err := url.Parse(req.Header.Get("Referer"))
			if err != nil {
				ctx.Warningf("Unable to parse referer url : %s", err)
			}
			longURL := u.Scheme + "://" + u.Host + "#/?id=" + upload.ID
			shortURL, err := shortenBackend.GetShortenBackend().Shorten(ctx.Fork("shorten url"), longURL)
			if err == nil {
				upload.ShortURL = shortURL
			} else {
				ctx.Warningf("Unable to shorten url %s : %s", longURL, err)
			}
		}
	}

	// Create files
	for i, file := range upload.Files {
		file.GenerateID()
		file.Status = "missing"
		delete(upload.Files, i)
		upload.Files[file.ID] = file
	}

	// Save the metadata
	err = metadataBackend.GetMetaDataBackend().Create(ctx.Fork("create metadata"), upload)
	if err != nil {
		ctx.Warningf("Create new upload error : %s", err)
		http.Error(resp, common.NewResult("Unable to create new upload", nil).ToJSONString(), 500)
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
		http.Error(resp, common.NewResult("Unable to serialize response body", nil).ToJSONString(), 500)
	}

	resp.Write(json)
}

func getUploadHandler(resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx := common.NewPlikContext("get upload handler", req)
	defer ctx.Finalize(err)

	// Check that source IP address is valid
	code, err := checkSourceIP(ctx, false)
	if err != nil {
		http.Error(resp, common.NewResult(err.Error(), nil).ToJSONString(), code)
		return
	}

	// Get the upload id and file id from the url params
	vars := mux.Vars(req)
	uploadID := vars["uploadID"]
	ctx.SetUpload(uploadID)

	// Retrieve upload metadata
	upload, err := metadataBackend.GetMetaDataBackend().Get(ctx.Fork("get metadata"), uploadID)
	if err != nil {
		ctx.Warningf("Upload %s not found : %s", uploadID, err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Upload %s not found", uploadID), nil).ToJSONString(), 404)
		return
	}

	ctx.Infof("Got upload from metadata backend")

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		ctx.Warningf("Unauthorized %s : %s", upload.ID, err)
		return
	}

	// Remove all private informations (ip, data backend details, ...) before
	// sending metadata back to the client
	upload.Sanitize()

	// Print upload metadata in the json response.
	var json []byte
	if json, err = utils.ToJson(upload); err != nil {
		ctx.Warningf("Unable to serialize response body : %s", err)
		http.Error(resp, common.NewResult("Unable to serialize response body", nil).ToJSONString(), 500)
	}
	resp.Write(json)
}

func getFileHandler(resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx := common.NewPlikContext("get file handler", req)
	defer ctx.Finalize(err)

	// Check that source IP address is valid
	code, err := checkSourceIP(ctx, false)
	if err != nil {
		redirect(req, resp, err, code)
		return
	}

	// Get the upload id and file id from the url params
	vars := mux.Vars(req)
	uploadID := vars["uploadID"]
	fileID := vars["fileID"]
	fileName := vars["filename"]
	if uploadID == "" {
		ctx.Warning("Missing upload id")
		redirect(req, resp, errors.New("Missing upload id"), 404)
		return
	}
	if fileID == "" {
		ctx.Warning("Missing file id")
		redirect(req, resp, errors.New("Missing file id"), 404)
		return
	}
	ctx.SetUpload(uploadID)

	// Get the upload informations from the metadata backend
	upload, err := metadataBackend.GetMetaDataBackend().Get(ctx.Fork("get metadata"), uploadID)
	if err != nil {
		ctx.Warningf("Upload %s not found : %s", uploadID, err)
		redirect(req, resp, fmt.Errorf("Upload %s not found", uploadID), 404)
		return
	}

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		ctx.Warningf("Unauthorized : %s", err)
		return
	}

	// Test if upload is not expired
	if upload.TTL != 0 {
		if time.Now().Unix() >= (upload.Creation + int64(upload.TTL)) {
			ctx.Warningf("Upload is expired since %s", time.Since(time.Unix(upload.Creation, int64(0)).Add(time.Duration(upload.TTL)*time.Second)).String())
			redirect(req, resp, fmt.Errorf("Upload %s is expired", upload.ID), 404)
			return
		}
	}

	// Retrieve file using data backend
	if _, ok := upload.Files[fileID]; !ok {
		ctx.Warningf("File %s not found", fileID)
		redirect(req, resp, fmt.Errorf("File %s not found", fileID), 404)
		return
	}

	file := upload.Files[fileID]
	ctx.SetFile(file.Name)

	// Compare url filename with upload filename
	if file.Name != fileName {
		ctx.Warningf("Invalid filename %s mismatch %s", fileName, file.Name)
		redirect(req, resp, fmt.Errorf("File %s not found", fileName), 404)
		return
	}

	// If upload has OneShot option, test if file has not been already downloaded once
	if upload.OneShot && file.Status == "downloaded" {
		ctx.Warningf("File %s has already been downloaded in upload %s", file.Name, upload.ID)
		redirect(req, resp, fmt.Errorf("File %s has already been downloaded", file.Name), 404)
		return
	}

	// If the file is marked as deleted by a previous call, we abort request
	if upload.Removable && file.Status == "removed" {
		ctx.Warningf("File %s has been removed", file.Name)
		redirect(req, resp, fmt.Errorf("File %s has been removed", file.Name), 404)
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
			ctx.Warningf("Got a Yubikey upload but Yubikey backend is disabled")
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
	if file.CurrentSize > 0 {
		resp.Header().Set("Content-Length", strconv.Itoa(int(file.CurrentSize)))
	}

	// If "dl" GET params is set
	// -> Set Content-Disposition header
	// -> The client should download file instead of displaying it
	dl := req.URL.Query().Get("dl")
	if dl != "" {
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`attachement; filename="%s"`, file.Name))
	} else {
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, file.Name))
	}

	// HEAD Request => Do not print file, user just wants http headers
	// GET  Request => Print file content
	ctx.Infof("Got a %s request", req.Method)

	if req.Method == "GET" {
		// Get file in data backend
		var backend dataBackend.DataBackend
		if upload.Stream {
			backend = dataBackend.GetStreamBackend()
		} else {
			backend = dataBackend.GetDataBackend()
		}
		fileReader, err := backend.GetFile(ctx.Fork("get file"), upload, file.ID)
		if err != nil {
			ctx.Warningf("Failed to get file %s in upload %s : %s", file.Name, upload.ID, err)
			redirect(req, resp, fmt.Errorf("Failed to read file %s", file.Name), 404)
			return
		}
		defer fileReader.Close()

		// Update metadata if oneShot option is set
		if upload.OneShot {
			file.Status = "downloaded"
			err = metadataBackend.GetMetaDataBackend().AddOrUpdateFile(ctx.Fork("update metadata"), upload, file)
			if err != nil {
				ctx.Warningf("Error while deleting file %s from upload %s metadata : %s", file.Name, upload.ID, err)
			}
		}

		// File is piped directly to http response body without buffering
		_, err = io.Copy(resp, fileReader)
		if err != nil {
			ctx.Warningf("Error while copying file to response : %s", err)
		}

		// Remove file from data backend if oneShot option is set
		if upload.OneShot {
			err = backend.RemoveFile(ctx.Fork("remove file"), upload, file.ID)
			if err != nil {
				ctx.Warningf("Error while deleting file %s from upload %s : %s", file.Name, upload.ID, err)
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

	// Check that source IP address is valid
	code, err := checkSourceIP(ctx, false)
	if err != nil {
		http.Error(resp, common.NewResult(err.Error(), nil).ToJSONString(), code)
		return
	}

	// Get the upload id from the url params
	vars := mux.Vars(req)
	uploadID := vars["uploadID"]
	fileID := vars["fileID"]
	ctx.SetUpload(uploadID)

	// Get upload metadata
	upload, err := metadataBackend.GetMetaDataBackend().Get(ctx.Fork("get metadata"), uploadID)
	if err != nil {
		ctx.Warningf("Upload metadata not found")
		http.Error(resp, common.NewResult(fmt.Sprintf("Upload %s not found", uploadID), nil).ToJSONString(), 404)
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
		http.Error(resp, common.NewResult("Invalid upload token in X-UploadToken header", nil).ToJSONString(), 404)
		return
	}

	// Create a new file object
	var newFile *common.File
	if fileID != "" {
		if _, ok := upload.Files[fileID]; ok {
			newFile = upload.Files[fileID]
		} else {
			ctx.Warningf("Invalid file id %s", fileID)
			http.Error(resp, common.NewResult("Invalid file id", nil).ToJSONString(), 404)
			return
		}
	} else {
		newFile = common.NewFile()
		newFile.Type = "application/octet-stream"
	}
	ctx.SetFile(newFile.ID)

	// Get file handle from multipart request
	var file io.Reader
	multiPartReader, err := req.MultipartReader()
	if err != nil {
		ctx.Warningf("Failed to get file from multipart request : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Failed to get file from multipart request"), nil).ToJSONString(), 500)
		return
	}

	// Read multipart body until the "file" part
	for {
		part, errPart := multiPartReader.NextPart()
		if errPart == io.EOF {
			break
		}
		if part.FormName() == "file" {
			file = part
			newFile.Name = part.FileName()
			break
		}
	}
	if file == nil {
		ctx.Warning("Missing file from multipart request")
		http.Error(resp, common.NewResult("Missing file from multipart request", nil).ToJSONString(), 400)
		return
	}
	if newFile.Name == "" {
		ctx.Warning("Missing file name from multipart request")
		http.Error(resp, common.NewResult("Missing file name from multipart request", nil).ToJSONString(), 400)
	}
	ctx.SetFile(newFile.Name)

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
	var backend dataBackend.DataBackend
	if upload.Stream {
		backend = dataBackend.GetStreamBackend()
	} else {
		backend = dataBackend.GetDataBackend()
	}
	backendDetails, err := backend.AddFile(ctx.Fork("save file"), upload, newFile, preprocessReader)
	if err != nil {
		ctx.Warningf("Unable to save file : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Error saving file %s in upload %s : %s", newFile.Name, upload.ID, err), nil).ToJSONString(), 500)
		return
	}

	// Fill-in file informations
	newFile.CurrentSize = int64(totalBytes)
	if upload.Stream {
		newFile.Status = "downloaded"
	} else {
		newFile.Status = "uploaded"
	}
	newFile.Md5 = fmt.Sprintf("%x", md5Hash.Sum(nil))
	newFile.UploadDate = time.Now().Unix()
	newFile.BackendDetails = backendDetails

	// Update upload metadata
	upload.Files[newFile.ID] = newFile
	err = metadataBackend.GetMetaDataBackend().AddOrUpdateFile(ctx.Fork("update metadata"), upload, newFile)
	if err != nil {
		ctx.Warningf("Unable to update metadata : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Error adding file %s to upload %s metadata : %s", newFile.Name, upload.ID, err), nil).ToJSONString(), 500)
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
		http.Error(resp, common.NewResult("Unable to serialize response body", nil).ToJSONString(), 500)
	}
}

func removeFileHandler(resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx := common.NewPlikContext("remove file handler", req)
	defer ctx.Finalize(err)

	// Check that source IP address is valid
	code, err := checkSourceIP(ctx, false)
	if err != nil {
		http.Error(resp, common.NewResult(err.Error(), nil).ToJSONString(), code)
		return
	}

	// Get the upload id and file id from the url params
	vars := mux.Vars(req)
	uploadID := vars["uploadID"]
	fileID := vars["fileID"]
	if uploadID == "" {
		ctx.Warning("Missing upload id")
		http.Error(resp, common.NewResult(fmt.Sprintf("Upload %s not found", uploadID), nil).ToJSONString(), 404)
		return
	}
	if fileID == "" {
		ctx.Warning("Missing file id")
		http.Error(resp, common.NewResult(fmt.Sprintf("File %s not found", fileID), nil).ToJSONString(), 404)
		return
	}
	ctx.SetUpload(uploadID)

	// Retrieve Upload
	upload, err := metadataBackend.GetMetaDataBackend().Get(ctx.Fork("get metadata"), uploadID)
	if err != nil {
		ctx.Warning("Upload not found")
		http.Error(resp, common.NewResult(fmt.Sprintf("Upload not %s found", uploadID), nil).ToJSONString(), 404)
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
		ctx.Warningf("User tried to remove file %s of an non removeable upload", fileID)
		http.Error(resp, common.NewResult("Can't remove files on this upload", nil).ToJSONString(), 401)
		return
	}

	// Check upload token
	if req.Header.Get("X-UploadToken") != upload.UploadToken {
		ctx.Warningf("Invalid upload token %s", req.Header.Get("X-UploadToken"))
		http.Error(resp, common.NewResult("Invalid upload token in X-UploadToken header", nil).ToJSONString(), 403)
		return
	}

	// Retrieve file informations in upload
	file, ok := upload.Files[fileID]
	if !ok {
		ctx.Warningf("File not found")
		http.Error(resp, common.NewResult(fmt.Sprintf("File %s not found in upload %s", fileID, upload.ID), nil).ToJSONString(), 404)
		return
	}

	// Set status to removed, and save metadatas
	file.Status = "removed"
	if err := metadataBackend.GetMetaDataBackend().AddOrUpdateFile(ctx.Fork("update metadata"), upload, file); err != nil {
		ctx.Warningf("Error while updating file metadata : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Error while updating file %s metadata in upload %s", file.Name, upload.ID), nil).ToJSONString(), 500)
		return
	}

	// Remove file from data backend
	// Get file in data backend
	var backend dataBackend.DataBackend
	if upload.Stream {
		backend = dataBackend.GetStreamBackend()
	} else {
		backend = dataBackend.GetDataBackend()
	}
	if err := backend.RemoveFile(ctx.Fork("remove file"), upload, file.ID); err != nil {

		ctx.Warningf("Error while deleting file : %s", err)
		http.Error(resp, common.NewResult(fmt.Sprintf("Error while deleting file %s in upload %s", file.Name, upload.ID), nil).ToJSONString(), 500)
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
		http.Error(resp, common.NewResult("Unable to serialize response body", nil).ToJSONString(), 500)
	}
	resp.Write(json)
}

//
//// Misc functions
//

// Check if source IP address is valid and whitelisted
func checkSourceIP(ctx *common.PlikContext, whitelist bool) (code int, err error) {
	// Get source IP address from context
	sourceIPstr, ok := ctx.Get("RemoteIP")
	if !ok || sourceIPstr.(string) == "" {
		ctx.Warning("Unable to get source IP address from context")
		err = errors.New("Unable to get source IP address")
		code = 401
		return
	}

	// Parse source IP address
	sourceIP := net.ParseIP(sourceIPstr.(string))
	if sourceIP == nil {
		ctx.Warningf("Unable to parse source IP address %s", sourceIPstr)
		err = errors.New("Unable to parse source IP address")
		code = 401
		return
	}

	// If needed check that source IP address is in whitelist
	if whitelist && len(common.UploadWhitelist) > 0 {
		for _, net := range common.UploadWhitelist {
			if net.Contains(sourceIP) {
				return
			}
		}
		ctx.Warningf("Unauthorized source IP address %s", sourceIPstr)
		err = errors.New("Unauthorized source IP address")
		code = 403
	}
	return
}

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
				err = fmt.Errorf("Inavlid Authorization header %s", req.Header.Get("Authorization"))
			}
			if auth[0] != "Basic" {
				err = fmt.Errorf("Inavlid http authorization scheme : %s", auth[0])
			}
			var md5sum string
			md5sum, err = utils.Md5sum(auth[1])
			if err != nil {
				err = fmt.Errorf("Unable to hash credentials : %s", err)
			}
			if md5sum != upload.Password {
				err = errors.New("Invalid credentials")
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

var userAgents = []string{"wget", "curl", "python-urllib", "libwwww-perl", "php", "pycurl"}

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

// UploadsCleaningRoutine periodicaly remove expired uploads
func UploadsCleaningRoutine() {
	ctx := common.RootContext().Fork("clean expired uploads")

	for {

		// Sleep between 2 hours and 3 hours
		// This is a dirty trick to avoid frontends doing this at the same time
		r, _ := rand.Int(rand.Reader, big.NewInt(3600))
		randomSleep := r.Int64() + 7200

		log.Infof("Will clean old uploads in %d seconds.", randomSleep)
		time.Sleep(time.Duration(randomSleep) * time.Second)

		// Get uploads that needs to be removed
		log.Infof("Cleaning expired uploads...")

		uploadIds, err := metadataBackend.GetMetaDataBackend().GetUploadsToRemove(ctx)
		if err != nil {
			log.Warningf("Failed to get expired uploads : %s")
		} else {

			// Remove them
			for _, uploadID := range uploadIds {
				ctx.SetUpload(uploadID)
				log.Infof("Removing expired upload %s", uploadID)
				// Get upload metadata
				childCtx := ctx.Fork("get metadata")
				childCtx.AutoDetach()
				upload, err := metadataBackend.GetMetaDataBackend().Get(childCtx, uploadID)
				if err != nil {
					log.Warningf("Unable to get infos for upload: %s", err)
					continue
				}

				// Remove from data backend
				childCtx = ctx.Fork("remove upload data")
				childCtx.AutoDetach()
				err = dataBackend.GetDataBackend().RemoveUpload(childCtx, upload)
				if err != nil {
					log.Warningf("Unable to remove upload data : %s", err)
					continue
				}

				// Remove from metadata backend
				childCtx = ctx.Fork("remove upload metadata")
				childCtx.AutoDetach()
				err = metadataBackend.GetMetaDataBackend().Remove(childCtx, upload)
				if err != nil {
					log.Warningf("Unable to remove upload metadata : %s", err)
				}
			}
		}
	}
}

// RemoveUploadIfNoFileAvailable iterates on upload files and remove upload files
// and metadata if all the files have been downloaded (usefull for OneShot uploads)
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

		if !upload.Stream {
			err = dataBackend.GetDataBackend().RemoveUpload(ctx, upload)
			if err != nil {
				return
			}
		}
		err = metadataBackend.GetMetaDataBackend().Remove(ctx, upload)
		if err != nil {
			return
		}
	}

	return
}
