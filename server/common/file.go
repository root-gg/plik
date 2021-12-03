package common

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
)

// FileMissing when a file is waiting to be uploaded
const FileMissing = "missing"

// FileUploading when a file is being uploaded
const FileUploading = "uploading"

// FileUploaded when a file has been uploaded and is ready to be downloaded
const FileUploaded = "uploaded"

// FileRemoved when a file has been removed and can't be downloaded anymore but has not yet been deleted
const FileRemoved = "removed"

// FileDeleted when a file has been deleted from the data backend
const FileDeleted = "deleted"

// File object
type File struct {
	ID       string `json:"id"`
	UploadID string `json:"-" gorm:"size:256;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	Name     string `json:"fileName"`

	Status string `json:"status"`

	Md5       string `json:"fileMd5"`
	Type      string `json:"fileType"`
	Size      int64  `json:"fileSize"`
	Reference string `json:"reference"`

	BackendDetails string `json:"-"`

	CreatedAt time.Time `json:"createdAt"`
}

// NewFile instantiate a new object
// and generate a random id
func NewFile() (file *File) {
	file = new(File)
	file.ID = GenerateRandomID(16)
	return
}

// GenerateID generate a new File ID
func (file *File) GenerateID() {
	file.ID = GenerateRandomID(16)
}

// Sanitize clear some fields to hide sensible information from the API.
func (file *File) Sanitize() {
	file.BackendDetails = ""
}

// CreateFile prepares a new file object to be persisted in DB ( create file ID, link upload ID, check name, ... )
func CreateFile(config *Configuration, upload *Upload, params *File) (file *File, err error) {
	if upload.ID == "" {
		return nil, fmt.Errorf("upload not initialized")
	}

	file = NewFile()
	file.Status = FileMissing
	file.UploadID = upload.ID

	file.Name = params.Name
	file.Type = params.Type
	file.Size = params.Size
	file.Reference = params.Reference

	if file.Name == "" {
		return nil, fmt.Errorf("missing file name")
	}

	// Check file name length
	if len(file.Name) > 1024 {
		return nil, fmt.Errorf("file name %s... is too long, maximum length is 1024 characters", file.Name[:20])
	}

	// Check file size
	if file.Size > 0 && config.MaxFileSize > 0 && file.Size > config.MaxFileSize {
		return nil, fmt.Errorf("file is too big (%s), maximum file size is %s", humanize.Bytes(uint64(file.Size)), humanize.Bytes(uint64(config.MaxFileSize)))
	}

	return file, nil
}
