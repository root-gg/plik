package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/root-gg/utils"
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

// CreateUpload upload for database insert ( check configuration and default values, generate upload and file IDs, ... )
func CreateUpload(config *Configuration, params *Upload) (upload *Upload, err error) {
	upload = &Upload{}
	upload.ID = GenerateRandomID(16)
	upload.UploadToken = GenerateRandomID(32)

	// Set user configurable parameters
	err = upload.setParams(config, params)
	if err != nil {
		return nil, err
	}

	// Handle Basic Auth parameters
	err = upload.setBasicAuth(config, params.Login, params.Password)
	if err != nil {
		return nil, err
	}

	// Handle files
	err = upload.setFiles(config, params.Files)
	if err != nil {
		return nil, err
	}

	return upload, nil
}

func (upload *Upload) setParams(config *Configuration, params *Upload) (err error) {
	upload.OneShot = params.OneShot
	if upload.OneShot && !config.OneShot {
		return fmt.Errorf("one shot uploads are not enabled")
	}

	upload.Removable = params.Removable
	if upload.Removable && !config.Removable {
		return fmt.Errorf("removable uploads are not enabled")
	}

	upload.Stream = params.Stream
	if upload.Stream && !config.Stream {
		return fmt.Errorf("stream mode is not enabled")
	}

	upload.TTL = params.TTL
	upload.Comments = params.Comments

	return nil
}

func (upload *Upload) setFiles(config *Configuration, files []*File) (err error) {
	// Limit number of files per upload
	if len(files) > config.MaxFilePerUpload {
		return fmt.Errorf("too many files. maximum is %d", config.MaxFilePerUpload)
	}

	// Create and check files
	for _, fileParams := range files {
		file, err := CreateFile(config, upload, fileParams)
		if err != nil {
			return err
		}
		upload.Files = append(upload.Files, file)
	}

	return nil
}

func (upload *Upload) setBasicAuth(config *Configuration, login string, password string) (err error) {
	if !config.ProtectedByPassword && (login != "" || password != "") {
		return fmt.Errorf("password protection is not enabled")
	}

	if login != "" {
		upload.Login = login
	} else {
		upload.Login = "plik"
	}

	if password == "" {
		return nil
	}

	upload.ProtectedByPassword = true

	// Save only the md5sum of this string to authenticate further requests
	upload.Password, err = utils.Md5sum(EncodeAuthBasicHeader(login, password))
	if err != nil {
		return fmt.Errorf("unable to generate password hash : %s", err)
	}

	return nil
}

// SetTTL adjust TTL parameters accordingly to default and max TTL
func (upload *Upload) SetTTL(defaultTTL int, maxTTL int) (err error) {
	upload.CreatedAt = time.Now()

	// TTL = Time in second before the upload expiration
	// >0 	-> TTL specified
	// 0 	-> No TTL specified : default value from configuration
	// <0	-> No expiration
	if upload.TTL == 0 {
		upload.TTL = defaultTTL
	}

	if maxTTL > 0 {
		if upload.TTL < 0 {
			return fmt.Errorf("cannot set infinite TTL (maximum allowed is : %d)", maxTTL)
		}
		if upload.TTL > maxTTL {
			return fmt.Errorf("invalid TTL. (maximum allowed is : %d)", maxTTL)
		}
	}

	if upload.TTL > 0 {
		deadline := upload.CreatedAt.Add(time.Duration(upload.TTL) * time.Second)
		upload.ExpireAt = &deadline
	}

	return nil
}

// InitializeForTests initialize upload for database insert without config checks and override for testing purpose
func (upload *Upload) InitializeForTests() {
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
