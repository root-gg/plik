package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

var (
	randRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

// Upload object
type Upload struct {
	ID  string `json:"id"`
	TTL int    `json:"ttl"`

	DownloadDomain string `json:"downloadDomain"`
	RemoteIP       string `json:"uploadIp,omitempty"`
	Comments       string `json:"comments"`

	Files []*File `json:"files"`

	UploadToken string `json:"uploadToken,omitempty"`
	User        string `json:"user,omitempty" gorm:"index:idx_upload_user"`
	Token       string `json:"token,omitempty" gorm:"index:idx_upload_user_token"`
	IsAdmin     bool   `json:"admin"`

	Stream    bool `json:"stream"`
	OneShot   bool `json:"oneShot"`
	Removable bool `json:"removable"`

	ProtectedByPassword bool   `json:"protectedByPassword"`
	Login               string `json:"login,omitempty"`
	Password            string `json:"password,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	DeletedAt *time.Time `json:"-" gorm:"index:idx_upload_deleted_at"`
	ExpireAt  *time.Time `json:"expireAt" gorm:"index:idx_upload_expire_at"`
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

// Sanitize removes sensible information from
// object. Used to hide information in API.
func (upload *Upload) Sanitize() {
	upload.RemoteIP = ""
	upload.Login = ""
	upload.Password = ""
	upload.UploadToken = ""
	upload.User = ""
	upload.Token = ""
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

// PrepareInsert upload for database insert ( check configuration and default values, generate upload and file IDs, ... )
func (upload *Upload) PrepareInsert(config *Configuration) (err error) {
	upload.ID = GenerateRandomID(16)
	upload.UploadToken = GenerateRandomID(32)

	// Limit number of files per upload
	if len(upload.Files) > config.MaxFilePerUpload {
		return fmt.Errorf("too many files. maximum is %d", config.MaxFilePerUpload)
	}

	if config.NoAnonymousUploads && upload.User == "" {
		return fmt.Errorf("anonymous uploads are disabled")
	}

	if !config.Authentication && (upload.User != "" || upload.Token != "") {
		return fmt.Errorf("authentication is disabled")
	}

	if upload.OneShot && !config.OneShot {
		return fmt.Errorf("one shot uploads are not enabled")
	}

	if upload.Removable && !config.Removable {
		return fmt.Errorf("removable uploads are not enabled")
	}

	if upload.Stream && !config.Stream {
		upload.OneShot = false
		return fmt.Errorf("stream mode is not enabled")
	}

	if !config.ProtectedByPassword && (upload.Login != "" || upload.Password != "") {
		upload.ProtectedByPassword = true
		return fmt.Errorf("password protection is not enabled")
	}

	// TTL = Time in second before the upload expiration
	// 0 	-> No ttl specified : default value from configuration
	// -1	-> No expiration : checking with configuration if that's ok
	switch upload.TTL {
	case 0:
		upload.TTL = config.DefaultTTL
	case -1:
		if config.MaxTTL != -1 {
			return fmt.Errorf("cannot set infinite ttl (maximum allowed is : %d)", config.MaxTTL)
		}
	default:
		if upload.TTL <= 0 {
			return fmt.Errorf("invalid ttl")
		}
		if config.MaxTTL > 0 && upload.TTL > config.MaxTTL {
			return fmt.Errorf("invalid ttl. (maximum allowed is : %d)", config.MaxTTL)
		}
	}

	if upload.TTL > 0 {
		deadline := time.Now().Add(time.Duration(upload.TTL) * time.Second)
		upload.ExpireAt = &deadline
	}

	for _, file := range upload.Files {
		err = file.PrepareInsert(upload)
		if err != nil {
			return err
		}
	}

	return nil
}

// PrepareInsertForTests upload for database insert without config checks and override for testing purpose
func (upload *Upload) PrepareInsertForTests() {
	if upload.ID == "" {
		upload.ID = GenerateRandomID(16)
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
