package common

import (
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

// DetectMIME detects the MIME type of the given data and determines whether it
// is text-like. A MIME type is considered text-like if it or any of its
// ancestors in the MIME hierarchy starts with "text/" (e.g. application/json
// inherits from text/plain).
func DetectMIME(data []byte) (mimeType string, isText bool) {
	detected := mimetype.Detect(data)
	mimeType = detected.String()
	for cur := detected; cur != nil; cur = cur.Parent() {
		if strings.HasPrefix(cur.String(), "text/") {
			isText = true
			break
		}
	}
	return
}

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
	UploadID string `json:"-" gorm:"size:256;index:idx_file_upload_id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	Name     string `json:"fileName"`

	Status string `json:"status"`

	Md5       string `json:"fileMd5"`
	Type      string `json:"fileType"`
	Size      int64  `json:"fileSize"`
	IsText    bool   `json:"isText"`
	Reference string `json:"reference"`

	BackendDetails string `json:"-"`

	DownloadCount    int64      `json:"downloadCount"`
	LastDownloadedAt *time.Time `json:"lastDownloadedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// NewFile instantiate a new object
// and generate a random id
func NewFile() (file *File) {
	file = new(File)
	file.GenerateID()
	return
}

// GenerateID generate a new File ID
func (file *File) GenerateID() {
	file.ID = GenerateRandomID(16)
}

// Sanitize clear some fields to hide sensitive information from the API.
func (file *File) Sanitize() {
	file.BackendDetails = ""
}
