package sdk

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/root-gg/plik/server/common"
)

type PlikUpload = common.Upload

// Upload override from a Plik common.Upload
// to add some handy getters in SDK
type Upload struct {
	*PlikUpload
	Files  map[string]*File `json:"files"`
	client *Client
}

// UploadStatus
type UploadStatus struct {
	Error error
	File  *File
}

// UploadOptions is a handy class to manage upload options from SDK
// instead of being set in the common.Upload object
type UploadOptions struct {
	TTL                 int
	Comments            string
	Stream              bool
	OneShot             bool
	Removable           bool
	ProtectedByPassword bool
	User                string
	Token               string
	Login               string
	Password            string
	ProtectedByYubikey  bool
	Yubikey             string
}

func (u *Upload) URL() string {

	if u.client == nil {
		return ""
	}

	return fmt.Sprintf("%s/#/?id=%s", u.client.BaseURL.String(), u.ID)
}

// Create will call the API to create the
// upload object on Plik, with upload ID, and token
func (u *Upload) Create() (err error) {

	if u.ID == "" {
		req, err := u.client.newRequest("POST", "/upload", u)
		if err != nil {
			return err
		}

		returnedUpload := new(Upload)
		returnedUpload.client = u.client
		_, err = u.client.do(req, &returnedUpload)

		for k, returnedFile := range returnedUpload.Files {
			for _, file := range u.Files {
				if returnedFile.Reference == file.Reference {
					returnedUpload.Files[k] = file
					returnedUpload.Files[k].ID = returnedFile.ID
					continue
				}
			}
		}

		*u = *returnedUpload
	}

	return
}

// AddFile will upload a file to the specified plik upload
// from a standard io.Reader and a filename since we do not have it in the io.Reader
func (u *Upload) AddFile(r io.Reader, fileName string) (file *File, err error) {

	// Creates a new file
	file = new(File)
	file.File = common.NewFile()
	file.Reference = file.ID
	file.Name = fileName
	file.reader = r
	file.client = u.client
	file.upload = u

	// Add it to the current Plik Upload
	u.Files[file.Reference] = file

	return file, err
}

// AddFileFromPath will upload a file to the specified plik upload
// from a path on the current host file system
func (u *Upload) AddFileFromPath(path string) (file *File, err error) {

	// Open file on disk
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return u.AddFile(f, filepath.Base(f.Name()))
}

// Upload will start uploading all the files created
// earlier in the upload object. It will spawn a gorouting per upload
func (u *Upload) Upload() (err error) {

	// Create upload if not created already
	err = u.Create()
	if err != nil {
		return
	}

	errChan := make(chan *UploadStatus, len(u.Files))
	waigGroup := new(sync.WaitGroup)
	waigGroup.Add(len(u.Files))

	for _, file := range u.Files {
		go func(fileToUpload *File) {
			defer waigGroup.Done()

			path := fmt.Sprintf("/file/%s/%s/%s", u.ID, fileToUpload.ID, fileToUpload.Name)
			req, err := u.client.newFileRequest("POST", path, fileToUpload.Name, fileToUpload.reader)
			if err != nil {
				errChan <- &UploadStatus{Error: err, File: fileToUpload}
				return
			}

			_, err = u.client.do(req, &fileToUpload)
			if err != nil {
				errChan <- &UploadStatus{Error: err, File: fileToUpload}
				return
			}

			errChan <- &UploadStatus{Error: nil, File: fileToUpload}

		}(file)
	}

	waigGroup.Wait()
	close(errChan)

	finalErr := ""
	for status := range errChan {
		if status.Error != nil {
			finalErr += fmt.Sprintf("file %s: %s", status.File.ID, status.Error)
		}
	}

	if finalErr != "" {
		return errors.New(finalErr)
	}

	return
}

// Remove will call the API to delete the upload
// and all the associated files on Plik
func (u *Upload) Remove() (err error) {

	if u.ID != "" {
		path := fmt.Sprintf("/upload/%s", u.ID)
		req, err := u.client.newRequest("DELETE", path, nil)
		if err != nil {
			return err
		}

		_, err = u.client.do(req, &u)
	}

	return
}
