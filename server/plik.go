/* The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr> : core, file (data/metadata) backend, shorten backends
	- Charles-Antoine Mathieu <skatkatt@root.gg> : core, frontend, mongo metadata backend, cli client
	- Loïc Porte <bewiwi@bibabox.fr> : swift databackend, tests

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
	"github.com/root-gg/plik/server/data_backend"
	"github.com/root-gg/plik/server/metadata_backend"
	"github.com/root-gg/plik/server/shorten_backend"
	"github.com/root-gg/plik/server/utils"
	"io"
	"io/ioutil"
	"log"
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

func main() {
	// Misc
	rand.Seed(time.Now().UTC().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Read args
	var configFile = flag.String("config", "plikd.cfg", "Configuration file (default: plikd.cfg")
	var version = flag.Bool("version", false, "Show version of plikd")
	flag.Parse()

	// Show version if asked
	if *version {
		fmt.Printf("Plikd v%s\n", utils.PlikVersion)
		os.Exit(0)
	}

	// Load configuration
	log.Printf("Starting plikd server v" + utils.PlikVersion)
	utils.LoadConfiguration(*configFile)

	// Here are all the plikd REST api calls
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

	// Remove expired uploads routine
	//  -> Will remove periodicaly expired uploads
	go UploadsCleaningRoutine()

	// Http router spawning
	go func() {

		var err error

		if utils.Config.SslCert != "" && utils.Config.SslKey != "" {
			address := utils.Config.ListenAddress + ":" + strconv.Itoa(utils.Config.ListenPort)
			tlsConfig := &tls.Config{MinVersion: tls.VersionTLS10}
			server := &http.Server{Addr: address, Handler: r, TLSConfig: tlsConfig}
			err = server.ListenAndServeTLS(utils.Config.SslCert, utils.Config.SslKey)

		} else {
			err = http.ListenAndServe(utils.Config.ListenAddress+":"+strconv.Itoa(utils.Config.ListenPort), nil)

		}

		if err != nil {
			log.Fatalln(err)
		}
	}()

	// Handle signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	for {
		select {
		case s := <-c:
			fmt.Println("Got signal:", s)
			os.Exit(0)
		}
	}
}

/*
 * HTTP HANDLERS
 */

func createUploadHandler(resp http.ResponseWriter, req *http.Request) {
	log.Println("New upload")
	upload := utils.NewUpload()
	upload.RemoteIp = req.RemoteAddr

	// Read body request
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("Unable to read request body : %s", err)
		http.Error(resp, utils.NewResult("Unable to read request body", nil).ToJsonString(), 500)
		return
	}

	// Deserialize body into newly created upload object
	if len(body) > 0 {
		// Parse Json
		err = json.Unmarshal(body, upload)
		if err != nil {
			log.Printf("Unable to deserialize request body : %s", err)
			http.Error(resp, utils.NewResult("Unable to deserialize json request body", nil).ToJsonString(), 500)
			return
		}
	}

	// After deserialization, we can init upload
	// This will set upload id, upload token, ...
	upload.Create()

	// Handle TTL
	// 0 	-> No ttl specified : we set default value from configuration
	// -1	-> User wants no expiration : checking with configuration if that's ok
	// >0	-> User wants specific ttl  : checking with configuration if that's ok

	switch upload.Ttl {
	case 0:
		upload.Ttl = utils.Config.DefaultTtl
	case -1:
		if utils.Config.MaxTtl != 0 {
			http.Error(resp, utils.NewResult(fmt.Sprintf("Cannot set infinite ttl (maximum allowed is : %d)", utils.Config.MaxTtl), nil).ToJsonString(), 500)
			return
		}
	default:
		if upload.Ttl < 0 {
			http.Error(resp, utils.NewResult(fmt.Sprintf("Invalid value for ttl : %d", upload.Ttl), nil).ToJsonString(), 500)
			return
		}
		if utils.Config.MaxTtl != 0 && upload.Ttl > utils.Config.MaxTtl {
			http.Error(resp, utils.NewResult(fmt.Sprintf("Cannot set ttl to %d (maximum allowed is : %d)", upload.Ttl, utils.Config.MaxTtl), nil).ToJsonString(), 500)
			return
		}
	}

	// If user want a password, we set this setting in upload informations
	// Default login is "plik"
	// We add Authorization header to the response for convenience

	if upload.Password != "" {
		upload.ProtectedByPassword = true
		if upload.Login == "" {
			upload.Login = "plik"
		}
		b64str := base64.StdEncoding.EncodeToString([]byte(upload.Login + ":" + upload.Password))
		upload.Password, err = utils.Md5sum(b64str)
		if err != nil {
			log.Printf("Unable to generate password hash : %s", err)
			http.Error(resp, utils.NewResult("Unable to generate password hash", nil).ToJsonString(), 500)
			return
		}
		resp.Header().Add("Authorization", "Basic "+b64str)
	}

	// Handle Yubikiey
	// If user specified a yubikey token :
	// 	-> We check the token validity in api.yubico.com
	// 	-> If the token is ok, we store the yubikey id in upload (12 first characters in token)

	if upload.Yubikey != "" {
		upload.ProtectedByYubikey = true
		ok, err := utils.YubikeyCheckToken(upload.Yubikey)
		if err != nil {
			log.Printf("Unable to validate yubikey token : %s", err)
			http.Error(resp, utils.NewResult("Unable to validate yubikey token", nil).ToJsonString(), 500)
			return
		}

		if !ok {
			log.Printf("Invalid yubikey token")
			http.Error(resp, utils.NewResult("Invalid yubikey token", nil).ToJsonString(), 500)
			return
		}

		upload.Yubikey = upload.Yubikey[:12]
	}

	// We create a short url for upload with the backend specified in configuration.
	//	  -> We use Referer to get the fqdn of incoming request.
	//	  -> The Shorten() method will return the short link
	//	  -> We store it in upload informations

	shortenBackend := shorten_backend.GetShortenBackend()
	if shortenBackend != nil {
		if req.Header.Get("Referer") != "" {
			u, err := url.Parse(req.Header.Get("Referer"))
			if err != nil {
				log.Printf("Unable to parse referer url : %s", err)
			}
			longUrl := u.Scheme + "://" + u.Host + "#/?id=" + upload.Id
			shortUrl, err := shortenBackend.Shorten(longUrl)
			if err == nil {
				upload.ShortUrl = shortUrl
			} else {
				log.Println(fmt.Printf("Unable to shorten url %s : %s", longUrl, err))
			}
		}
	}

	// We must now save the upload informations in the metadata backend
	err = metadata_backend.GetMetadataBackend().Create(upload)
	if err != nil {
		log.Printf("Create new upload error : %s", err)
		http.Error(resp, utils.NewResult("Invalid yubikey token", nil).ToJsonString(), 500)
		return
	}

	// Before sending back upload informations to client, we remove
	// some sensible informations (ip, data backend details, ...)
	upload.Sanitize()

	// We can print upload to client using json library.
	var json []byte
	if json, err = utils.ToJson(upload); err == nil {
		resp.Write(json)
	} else {
		http.Error(resp, utils.NewResult("Unable to serialize response body", nil).ToJsonString(), 500)
	}
}

func getUploadHandler(resp http.ResponseWriter, req *http.Request) {

	// Get the upload id in url from mux router
	vars := mux.Vars(req)
	uploadId := vars["uploadid"]

	// Get the upload informations from the metadata backend
	upload, err := metadata_backend.GetMetadataBackend().Get(uploadId)
	if err != nil {
		log.Printf("Upload %s not found : %s", uploadId, err)
		http.Error(resp, utils.NewResult(fmt.Sprintf("Upload %s not found", uploadId), nil).ToJsonString(), 404)
		return
	}

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		log.Printf("Unauthorized %s : %s", upload.Id, err)
		return
	}

	// Do not show some sensible informations to client
	upload.Sanitize()

	// Show upload using json
	var json []byte
	if json, err = utils.ToJson(upload); err == nil {
		resp.Write(json)
	} else {
		http.Error(resp, utils.NewResult("Unable to serialize response body", nil).ToJsonString(), 500)
	}
}

func getFileHandler(resp http.ResponseWriter, req *http.Request) {

	// Get the upload id and file id in the url from mux variables
	// If we don't have both, request is aborting now
	vars := mux.Vars(req)
	uploadId := vars["uploadid"]
	fileId := vars["fileid"]

	if uploadId == "" || fileId == "" {
		http.Redirect(resp, req, "/", 301)
		redirect(req, resp, errors.New("Missing upload or file id"), 404)
		return
	}

	// Get the upload informations from the metadata backend
	upload, err := metadata_backend.GetMetadataBackend().Get(uploadId)
	if err != nil {
		log.Printf("Upload %s not found : %s", uploadId, err)
		redirect(req, resp, errors.New(fmt.Sprintf("Upload %s not found", uploadId)), 404)
		return
	}

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		log.Printf("Unauthorized %s : %s", upload.Id, err)
		return
	}

	// Test if upload is not expired
	if upload.Ttl != 0 {
		if time.Now().Unix() > (upload.Creation + int64(upload.Ttl)) {
			log.Printf("Upload %s is expired", uploadId)
			redirect(req, resp, errors.New(fmt.Sprintf("Upload %s is expired", upload.Id)), 404)
			return
		}
	}

	// Retrieve file using data backend
	if _, ok := upload.Files[fileId]; !ok {
		log.Printf("File %s not found in upload %s", fileId, upload.Id)
		redirect(req, resp, errors.New(fmt.Sprintf("File %s not found", fileId)), 404)
		return
	}

	file := upload.Files[fileId]

	// If upload has OneShot option, testing if file has not been already downloaded once
	if upload.OneShot && file.Status == "downloaded" {
		log.Printf("File %s has already been downloaded in upload %s", file.Name, upload.Id)
		redirect(req, resp, errors.New(fmt.Sprintf("File %s has already been downloaded", file.Name)), 401)
		return
	}

	// If the file is marked as deleted by a previous call, we abort request
	if upload.Removable && file.Status == "removed" {
		log.Printf("File %s has been removed", file.Name)
		redirect(req, resp, errors.New(fmt.Sprintf("File %s has been removed", file.Name)), 401)
		return
	}

	// Check yubikey
	// If upload is yubikey protected, user must send an OTP when he wants to get a file.
	if upload.Yubikey != "" {
		token := vars["yubikey"]

		if token == "" {
			log.Println("Missing yubikey token")
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}
		if len(token) != 44 {
			log.Printf("Invalid yubikey token : %s", token)
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}
		if token[:12] != upload.Yubikey {
			log.Printf("Invalid yubikey device : %s", token)
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}

		isValid, err := utils.YubikeyCheckToken(token)
		if err != nil {
			log.Printf("Failed to validate yubikey token : %s", err)
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}
		if !isValid {
			log.Println("Invalid yubikey token : %s", token)
			redirect(req, resp, errors.New("Invalid yubikey token"), 401)
			return
		}
	}

	// Set content type and print file
	resp.Header().Set("Content-Type", file.Type)
	resp.Header().Set("Content-Length", strconv.Itoa(int(file.CurrentSize)))

	// If we have a dl variable in GET params
	// -> We set attachement param in Content-Disposition header
	// -> The navigator should download file instead of showing it in the view

	dl := req.URL.Query().Get("dl")
	if dl != "" {
		resp.Header().Set("Content-Disposition", "attachement; filename="+file.Name)
	} else {
		resp.Header().Set("Content-Disposition", "filename="+file.Name)
	}

	// HEAD Request => Do not print file, user just wants http headers
	// GET  Request => Print file content

	if req.Method == "GET" {

		// Get file in data backend
		fileReader, err := data_backend.GetDataBackend().GetFile(upload, file.Id)
		if err != nil {
			log.Printf("Failed to get file %s in upload %s : %s", file.Name, upload.Id, err)
			redirect(req, resp, errors.New(fmt.Sprintf("Failed to read file %s", file.Name)), 404)
			return
		}
		defer fileReader.Close()

		// Copy content to response
		resultChan := make(chan error)
		go func() {
			_, err = io.Copy(resp, fileReader)
			if err != nil {
				log.Printf("Error while copying file to response : %s", err)
			}
			resultChan <- err
		}()

		// Remove if oneShot
		if upload.OneShot {
			file.Status = "downloaded"
			err = metadata_backend.GetMetadataBackend().AddOrUpdateFile(upload, file)
			if err != nil {
				log.Printf("Error while deleting file %s from upload %s metadata : %s", file.Name, upload.Id, err)
			}
			// Remove file from data backend
			if err := data_backend.GetDataBackend().RemoveFile(upload, file.Id); err != nil {
				log.Printf("Error while deleting file %s from upload %s : %s", file.Name, upload.Id, err)
				return
			}
		}

		// Waiting for the write of the file
		// to be finished before ending handler
		<-resultChan
	}
}

func addFileHandler(resp http.ResponseWriter, req *http.Request) {

	// Get the upload id in the url from mux variables
	vars := mux.Vars(req)
	uploadId := vars["uploadid"]

	// Get Upload
	upload, err := metadata_backend.GetMetadataBackend().Get(uploadId)
	if err != nil {
		log.Printf("Upload not %s found : %s", uploadId, err)
		http.Error(resp, utils.NewResult(fmt.Sprintf("Upload %s not found", uploadId), nil).ToJsonString(), 404)
		return
	}
	log.Printf(" - [META] Got metadatas from upload %s on backend %s", uploadId, utils.Config.MetadataBackend)

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		log.Printf("Unauthorized %s : %s", upload.Id, err)
		return
	}

	// Check if user has specify the upload token in http header
	// Test if it's the right one from upload infos
	if req.Header.Get("X-UploadToken") != upload.UploadToken {
		http.Error(resp, utils.NewResult("Invalid upload token in X-UploadToken header", nil).ToJsonString(), 404)
		return
	}

	// Get file handle from multipart request
	var file io.Reader
	var fileName string = ""

	multiPartReader, err := req.MultipartReader()
	if err != nil {
		log.Printf("Failed to get file in multipart request : %s", err)
		http.Error(resp, utils.NewResult(fmt.Sprintf("Failed to get file in multipart request"), nil).ToJsonString(), 500)
		return
	}

	// Read multipart until find the "file" part, which contain uploaded data
	// -> We have also the filename !
	for {
		part, err_part := multiPartReader.NextPart()
		if err_part == io.EOF {
			break
		}

		if part.FormName() == "file" {
			log.Printf(" - [MAIN] Got filehandle for file %s on upload %s", part.FileName(), uploadId)
			file = part
			fileName = part.FileName()
			break
		}
	}

	// Create a new file object
	newFile := utils.NewFile()
	newFile.Name = fileName
	newFile.Type = "application/octet-stream"

	// Here, we create a pipe
	// It will allow us to send file data throught it, and be able to :
	//    -> Compute md5 of file
	//    -> Determine size of file
	//    -> Determine content type of file (by reading 512 first bytes)

	checkLenghtReader, checkLenghtWriter := io.Pipe()
	md5Hash := md5.New()
	totalBytes := 0

	go func() {
		for {
			buf := make([]byte, 1024)
			bytesRead, err := file.Read(buf)

			if err != nil {
				checkLenghtWriter.Close()
				return
			}

			// If first loop detect content type
			if totalBytes == 0 {
				newFile.Type = http.DetectContentType(buf)
				log.Printf(" - [MAIN] Got Content-Type : %s", newFile.Type)
			}

			// Increment size
			totalBytes += bytesRead

			// Md5 stuff
			md5Hash.Write(buf[:bytesRead])

			// Check max size with config
			if totalBytes > utils.Config.MaxFileSize {
				maxSizeReachedError := errors.New(fmt.Sprintf("File too big (limit is set to %d bytes)", utils.Config.MaxFileSize))
				checkLenghtWriter.CloseWithError(maxSizeReachedError)
				return
			}

			// Writing buf to writer
			checkLenghtWriter.Write(buf[:bytesRead])
		}
	}()

	// Save file in the data backend
	backendDetails, err := data_backend.GetDataBackend().AddFile(upload, newFile, checkLenghtReader)
	if err != nil {
		log.Printf("Error saving file %s in upload %s : %s", newFile.Name, upload.Id, err)
		http.Error(resp, utils.NewResult(fmt.Sprintf("Error saving file %s in upload %s : %s", newFile.Name, upload.Id, err), nil).ToJsonString(), 500)
		return
	}
	log.Printf(" - [MAIN] File saved to data backend %s", utils.Config.DataBackend)

	// Fill-in file informations
	newFile.CurrentSize = int64(totalBytes)
	newFile.Status = "uploaded"
	newFile.Md5 = fmt.Sprintf("%x", md5Hash.Sum(nil))
	newFile.UploadDate = time.Now().Unix()
	newFile.BackendDetails = backendDetails

	// Save file to metadata backend
	upload.Files[newFile.Id] = newFile
	err = metadata_backend.GetMetadataBackend().AddOrUpdateFile(upload, newFile)
	if err != nil {
		log.Printf("Error adding file %s to upload %s metadata : %s", newFile.Name, upload.Id, err)
		http.Error(resp, utils.NewResult(fmt.Sprintf("Error adding file %s to upload %s metadata", newFile.Name, upload.Id, err), nil).ToJsonString(), 500)
		return
	}
	log.Printf(" - [MAIN] File saved to metadata backend %s", utils.Config.MetadataBackend)

	// Remove private data
	newFile.Sanitize()

	// Write response to client
	var json []byte
	if json, err = utils.ToJson(newFile); err == nil {
		resp.Write(json)
	} else {
		http.Error(resp, utils.NewResult("Unable to serialize response body", nil).ToJsonString(), 500)
	}
}

func removeFileHandler(resp http.ResponseWriter, req *http.Request) {

	// Get the upload id and file id in the url from mux variables
	// If we don't have both, request is aborting now
	log.Println("Remove file")
	vars := mux.Vars(req)
	uploadId := vars["uploadid"]
	fileId := vars["fileid"]

	// Retrieve Upload
	upload, err := metadata_backend.GetMetadataBackend().Get(uploadId)
	if err != nil {
		log.Printf("Upload not %s found : %s", uploadId, err)
		http.Error(resp, utils.NewResult(fmt.Sprintf("Upload not %s found", uploadId), nil).ToJsonString(), 404)
		return
	}

	// Handle basic auth if upload is password protected
	err = httpBasicAuth(req, resp, upload)
	if err != nil {
		log.Printf("Unauthorized %s : %s", upload.Id, err)
		return
	}

	// Retrieve file informations in upload
	file, ok := upload.Files[fileId]
	if !ok {
		log.Printf("File %s not found in upload %s", fileId, upload.Id)
		http.Error(resp, utils.NewResult(fmt.Sprintf("File %s not found in upload %s", fileId, upload.Id), nil).ToJsonString(), 404)
		return
	}

	// Set status to removed, and save metadatas
	file.Status = "removed"
	if err := metadata_backend.GetMetadataBackend().AddOrUpdateFile(upload, file); err != nil {
		log.Printf("Error while updating file %s metadata in upload %s : %s", file.Name, upload.Id, err)
		http.Error(resp, utils.NewResult(fmt.Sprintf("Error while updating file %s metadata in upload %s", file.Name, upload.Id), nil).ToJsonString(), 500)
		return
	}

	// Remove file from data backend
	if err := data_backend.GetDataBackend().RemoveFile(upload, file.Id); err != nil {
		log.Printf("Error while deleting file %s in upload %s : %s", file.Name, err)
		http.Error(resp, utils.NewResult(fmt.Sprintf("Error while deleting file %s in upload %s", file.Name, upload.Id), nil).ToJsonString(), 500)
		return
	}

	// TODO Remove upload if there is no more files availables
	var json []byte
	if json, err = utils.ToJson(upload); err == nil {
		resp.Write(json)
	} else {
		http.Error(resp, utils.NewResult("Unable to serialize response body", nil).ToJsonString(), 500)
	}
}

//
//// Misc functions
//

/*
	Function that handle basic authentification
*/
func httpBasicAuth(req *http.Request, resp http.ResponseWriter, upload *utils.Upload) (err error) {
	if upload.ProtectedByPassword {
		if req.Header.Get("Authorization") == "" {
			err = errors.New("Missing Authorization header")
		} else {
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
			resp.Header().Set("WWW-Authenticate", "Basic realm=\"plik\"")
			http.Error(resp, "Please provide valid credentials to download this file", 401)
		}
	}
	return
}

/*
	No redirection if user agent is a CLI tool
*/

var userAgents []string = []string{"wget", "curl", "python-urllib", "libwwww-perl", "php", "pycurl"}

func redirect(req *http.Request, resp http.ResponseWriter, err error, status int) {
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

/*
	Cleaning subroutine :
		-> Ask metadata backend a list of expired upload
		-> For each upload :
			- Remove each Files
			- Remove upload from metadatas
		-> Sleep random amout of time
*/

func UploadsCleaningRoutine() {
	for {

		// Sleep between 2 hours and 3 hours
		// This is a dirty trick to avoid frontends doing this at the same time
		// We are currently searching for a better way, maybe a centralized lock.

		randSleep := rand.Intn(3600) + 7200
		log.Printf("[CLEAN] Will clean old uploads in %d seconds.", randSleep)
		time.Sleep(time.Duration(randSleep) * time.Second)

		// Get uploads that needs remove
		log.Printf("[CLEAN] Purging old uploads...")

		uploadsId, err := metadata_backend.GetMetadataBackend().GetUploadsToRemove()
		if err != nil {
			log.Printf("Failed to get uploads to remove : %s")
		} else {

			// Remove them
			for _, uploadId := range uploadsId {

				log.Printf(" - Removing upload %s...", uploadId)

				// Get metadatas
				upload, err := metadata_backend.GetMetadataBackend().Get(uploadId)
				if err != nil {
					log.Printf(" -> Failed to get infos for upload: %s", err)
					continue
				}

				// Remove from data backend
				err = data_backend.GetDataBackend().RemoveUpload(upload)
				if err != nil {
					log.Printf(" -> Failed to remove upload : %s", err)
				}

				// Remove from metadata backend
				err = metadata_backend.GetMetadataBackend().Remove(upload)
				if err != nil {
					log.Printf(" -> Failed to remove upload : %s", err)
				}
			}
		}
	}
}
