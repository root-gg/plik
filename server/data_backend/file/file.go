package file

import (
	"github.com/root-gg/plik/server/utils"
	"io"
	"log"
	"os"
)

type FileBackendConfig struct {
	Directory string
}

func NewFileBackendConfig(config map[string]interface{}) (this *FileBackendConfig) {
	this = new(FileBackendConfig)
	this.Directory = "files"
	utils.Assign(this, config)
	return
}

type FileBackend struct {
	Config *FileBackendConfig
}

func NewFileBackend(config map[string]interface{}) (this *FileBackend) {
	this = new(FileBackend)
	this.Config = NewFileBackendConfig(config)
	return
}

func (this *FileBackend) GetFile(upload *utils.Upload, id string) (io.ReadCloser, error) {
	log.Printf(" - [FILE] Try to get file %s on upload %s", id, upload.Id)

	// Get paths
	directory := this.getDirectoryFromUploadId(upload.Id)
	fullPath := directory + "/" + id

	// Stat
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (this *FileBackend) AddFile(upload *utils.Upload, file *utils.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	log.Println(" - [FILE] Begin upload of file on upload %s", upload.Id)

	// Get paths
	directory := this.getDirectoryFromUploadId(upload.Id)
	fullPath := directory + "/" + file.Id

	// Create directory
	if _, err := os.Stat(directory); err != nil {
		if err := os.MkdirAll(directory, 0777); err != nil {
			return backendDetails, err
		}

		log.Printf(" - [FILE] Folder %s successfully created", directory)
	}

	// Open
	out, err := os.Create(fullPath)
	if err != nil {
		return backendDetails, err
	}

	// Save file
	_, err = io.Copy(out, fileReader)
	if err != nil {
		return backendDetails, err
	}

	log.Printf(" - [FILE] File %s successfully created on disk", fullPath)
	return backendDetails, nil
}

func (this *FileBackend) RemoveFile(upload *utils.Upload, id string) error {

	// Get upload path
	fullPath := this.getDirectoryFromUploadId(upload.Id) + "/" + id

	// Remove
	err := os.Remove(fullPath)
	if err != nil {
		return err
	}

	return nil
}

func (this *FileBackend) RemoveUpload(upload *utils.Upload) error {

	// Get upload path
	fullPath := this.getDirectoryFromUploadId(upload.Id)

	// Remove
	err := os.RemoveAll(fullPath)
	if err != nil {
		return err
	}

	return nil
}

func (this *FileBackend) getDirectoryFromUploadId(uploadId string) string {

	if len(uploadId) > 2 {
		return this.Config.Directory + "/" + uploadId[:2] + "/" + uploadId
	}

	return this.Config.Directory + "/" + uploadId + "/" + uploadId
}
