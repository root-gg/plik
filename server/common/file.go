package common

import (
	"fmt"
	"time"
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
	UploadID string `json:"-"  gorm:"type:varchar(255) REFERENCES uploads(id) ON UPDATE RESTRICT ON DELETE RESTRICT"`
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

// Sanitize removes sensible information from
// object. Used to hide information in API.
func (file *File) Sanitize() {
	file.BackendDetails = ""
}

// PrepareInsert prepares a new file object to be persisted in DB ( create file ID, link upload ID, check name, ... )
func (file *File) PrepareInsert(upload *Upload) (err error) {
	if upload == nil {
		return fmt.Errorf("missing upload")
	}

	if upload.ID == "" {
		return fmt.Errorf("upload not initialized")
	}

	file.UploadID = upload.ID

	if file.Name == "" {
		return fmt.Errorf("missing file name")
	}

	// Check file name length
	if len(file.Name) > 1024 {
		return fmt.Errorf("file name %s... is too long, maximum length is 1024 characters", file.Name[:20])
	}

	file.GenerateID()
	file.Status = FileMissing

	return nil
}
