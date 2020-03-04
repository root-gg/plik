package plik

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/root-gg/plik/server/common"
)

// File contains all relevant info needed to upload data to a Plik server
type File struct {
	Name string
	Size int64

	reader io.ReadCloser // Byte stream to upload
	upload *Upload       // Link to upload and client

	lock     sync.Mutex   // The two following fields need to be protected
	metadata *common.File // File metadata returned by the server

	callback func(metadata *common.File, err error) // Callback to execute once the file has been uploaded

	done chan struct{} // Used to synchronize Upload() calls
	err  error         // If an error occurs during a Upload() call this will be set
}

// NewFileFromReader creates a File from a filename and an io.ReadCloser
func newFileFromReadCloser(upload *Upload, name string, reader io.ReadCloser) *File {
	file := &File{}
	file.upload = upload
	file.Name = name
	file.reader = reader
	return file
}

// NewFileFromReader creates a File from a filename and an io.Reader
func newFileFromReader(upload *Upload, name string, reader io.Reader) *File {
	return newFileFromReadCloser(upload, name, ioutil.NopCloser(reader))
}

// NewFileFromPath creates a File from a filesystem path
func newFileFromPath(upload *Upload, path string) (file *File, err error) {

	// Test if file exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("file %s not found", path)
	}

	// Check mode
	if !fileInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("unhandled file mode %s for file %s", fileInfo.Mode().String(), path)
	}

	// Open file
	fh, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s : %s", path, err)
	}

	filename := filepath.Base(path)
	file = newFileFromReader(upload, filename, fh)
	file.Size = fileInfo.Size()

	return file, err
}

// newFileFromParams create a new file object from the give file parameters
func newFileFromParams(upload *Upload, params *common.File) *File {
	file := &File{}
	file.upload = upload
	file.metadata = params
	file.Name = params.Name
	file.Size = params.Size
	return file
}

// Metadata return the file metadata returned by the server
func (file *File) Metadata() (details *common.File) {
	file.lock.Lock()
	defer file.lock.Unlock()

	return file.metadata
}

// getParams return a common.File to be passed to internal methods
func (file *File) getParams() (params *common.File) {
	file.lock.Lock()
	defer file.lock.Unlock()

	params = &common.File{}
	params.Name = file.Name

	if file.metadata != nil {
		params.ID = file.metadata.ID
	}

	return params
}

// ID return the file ID if any
func (file *File) Error() error {
	file.lock.Lock()
	defer file.lock.Unlock()

	return file.err
}

// ready ensure that only one upload occurs ( like sync.Once )
// the first call to Upload() will proceed ( abort false ) and must close the done channel once done
// subsequent calls to Upload() will abort ( abort true ) and must :
//  - wait on the done channel for the former Upload() call to complete ( if not nil )
//  - return the error in upload.err
func (file *File) ready() (done chan struct{}, abort bool) {
	file.lock.Lock()
	defer file.lock.Unlock()

	// Upload is in progress or finished
	if file.done != nil {
		return file.done, true
	}

	if file.metadata == nil {
		file.metadata = &common.File{Status: common.FileMissing}
	}

	// File does not need to be uploaded
	// TODO : maybe it would be better/simpler to rely only on file.reader == nil ?
	if file.metadata.Status != common.FileMissing || file.reader == nil {
		return nil, true
	}

	// Set status to uploading
	file.metadata.Status = common.FileUploading

	// Grab the lock by setting this channel
	file.done = make(chan struct{})
	return file.done, false
}

// Upload uploads a single file.
func (file *File) Upload() (err error) {

	// initialize the upload if not already done
	err = file.upload.Create()
	if err != nil {
		return err
	}

	// synchronize
	done, abort := file.ready()
	if abort {
		if done != nil {
			<-done
		}
		return file.err
	}

	// Upload file to the server
	defer func() { _ = file.reader.Close() }()
	fileMetadata, err := file.upload.client.uploadFile(file.upload.getParams(), file.getParams(), file.reader)

	// update file with API call result
	file.lock.Lock()
	if err == nil {
		file.metadata = fileMetadata
	} else {
		file.err = err
	}
	file.lock.Unlock()

	// notify that we are done
	close(done)

	// execute registered callbacks
	callback := file.callback
	if callback != nil {
		callback(fileMetadata, err)
	}

	return err
}

// GetURL returns the URL to download the file
func (file *File) GetURL() (URL *url.URL, err error) {

	// Get upload metadata
	uploadMetadata := file.upload.Metadata()
	if uploadMetadata == nil || uploadMetadata.ID == "" {
		return nil, fmt.Errorf("upload has not been created yet")
	}

	// Get file metadata
	fileMetadata := file.Metadata()
	if fileMetadata == nil || fileMetadata.ID == "" {
		return nil, fmt.Errorf("file has not been uploaded yet")
	}

	mode := "file"
	if uploadMetadata.Stream {
		mode = "stream"
	}

	var domain string
	if uploadMetadata.DownloadDomain != "" {
		domain = uploadMetadata.DownloadDomain
	} else {
		domain = file.upload.client.URL
	}

	fileURL := fmt.Sprintf("%s/%s/%s/%s/%s", domain, mode, uploadMetadata.ID, fileMetadata.ID, fileMetadata.Name)

	// Parse to get a nice escaped url
	return url.Parse(fileURL)
}

// WrapReader a convenient function to alter the content of the file on the file ( encrypt / display progress / ... )
func (file *File) WrapReader(wrapper func(reader io.ReadCloser) io.ReadCloser) {
	file.reader = wrapper(file.reader)
}

// UploadCallback to be executed once the file has been uploaded
type UploadCallback func(metadata *common.File, err error)

// RegisterUploadCallback a callback to be executed after the file have been uploaded
func (file *File) RegisterUploadCallback(callback UploadCallback) {
	file.callback = callback
}

// Download downloads all the upload files in a zip archive
func (file *File) Download() (reader io.ReadCloser, err error) {
	return file.upload.client.downloadFile(file.upload.getParams(), file.getParams())
}

// Delete remove the upload and all the associated files from the remote server
func (file *File) Delete() (err error) {
	return file.upload.client.removeFile(file.upload.getParams(), file.getParams())
}
