package common

import (
	"crypto/rand"
	"math/big"
	"time"

	"gorm.io/gorm"
)

var (
	randRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

// Upload object
type Upload struct {
	ID  string `json:"id"`
	TTL int    `json:"ttl"`

	DownloadDomain string `json:"downloadDomain" gorm:"-"`
	RemoteIP       string `json:"uploadIp,omitempty"`
	Comments       string `json:"comments"`

	Files []*File `json:"files"`

	UploadToken string `json:"uploadToken,omitempty"`
	User        string `json:"user,omitempty" gorm:"index:idx_upload_user"`
	Token       string `json:"token,omitempty" gorm:"index:idx_upload_user_token"`

	IsAdmin bool `json:"admin" gorm:"-"`

	Stream    bool `json:"stream"`
	OneShot   bool `json:"oneShot"`
	Removable bool `json:"removable"`

	ProtectedByPassword bool   `json:"protectedByPassword"`
	Login               string `json:"login,omitempty"`
	Password            string `json:"password,omitempty"`

	CreatedAt time.Time      `json:"createdAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index:idx_upload_deleted_at"`
	ExpireAt  *time.Time     `json:"expireAt" gorm:"index:idx_upload_expire_at"`
}

// NewUpload creates a new upload object
func NewUpload() (upload *Upload) {
	upload = &Upload{}
	upload.GenerateID()
	upload.GenerateUploadToken()
	return upload
}

// GenerateID generate a new Upload ID and UploadToken
func (upload *Upload) GenerateID() {
	upload.ID = GenerateRandomID(16)
}

// GenerateUploadToken generate a new UploadToken
func (upload *Upload) GenerateUploadToken() {
	upload.UploadToken = GenerateRandomID(32)
}

// NewFile creates a new file and add it to the current upload
func (upload *Upload) NewFile() (file *File) {
	file = NewFile()
	upload.Files = append(upload.Files, file)
	file.UploadID = upload.ID
	return file
}

// GetFile get file with ID from upload files. Return nil if not found
func (upload *Upload) GetFile(ID string) (file *File) {
	for _, file := range upload.Files {
		if file.ID == ID {
			return file
		}
	}

	return nil
}

// GetFileByReference get file with Reference from upload files. Return nil if not found
func (upload *Upload) GetFileByReference(ref string) (file *File) {
	for _, file := range upload.Files {
		if file.Reference == ref {
			return file
		}
	}

	return nil
}

// Sanitize clear some fields to hide sensible information from the API.
func (upload *Upload) Sanitize(config *Configuration) {
	upload.RemoteIP = ""
	upload.Login = ""
	upload.Password = ""
	upload.User = ""
	upload.Token = ""

	if !upload.IsAdmin {
		upload.UploadToken = ""
	}

	upload.DownloadDomain = config.DownloadDomain
	for _, file := range upload.Files {
		file.Sanitize()
	}
}

// GenerateRandomID generates a random string with specified length.
// Used to generate upload id, tokens, ...
func GenerateRandomID(length int) string {
	max := *big.NewInt(int64(len(randRunes)))
	b := make([]rune, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, &max)
		b[i] = randRunes[n.Int64()]
	}

	return string(b)
}

// IsExpired check if the upload is expired
func (upload *Upload) IsExpired() bool {
	if upload.ExpireAt != nil {
		if time.Now().After(*upload.ExpireAt) {
			return true
		}
	}
	return false
}

// InitializeForTests initialize upload for database insert without config checks and override for testing purpose
func (upload *Upload) InitializeForTests() {
	if upload.ID == "" {
		upload.GenerateID()
	}

	if upload.ExpireAt == nil && upload.TTL > 0 {
		deadline := time.Now().Add(time.Duration(upload.TTL) * time.Second)
		upload.ExpireAt = &deadline
	}

	for _, file := range upload.Files {
		if file.ID == "" {
			file.GenerateID()
		}
		file.UploadID = upload.ID
		if file.Status == "" {
			file.Status = FileMissing
		}
	}
}
