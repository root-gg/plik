package common

import (
	"encoding/json"
	"fmt"
	"time"
)

// UploadV1 upload object compatible with Plik <1.3
type UploadV1 struct {
	ID       string `json:"id"`
	Creation int64  `json:"uploadDate"`
	TTL      int    `json:"ttl"`

	DownloadDomain string `json:"downloadDomain"`
	RemoteIP       string `json:"uploadIp,omitempty"`
	Comments       string `json:"comments"`

	Files map[string]*File `json:"files"`

	UploadToken string `json:"uploadToken,omitempty"`
	User        string `json:"user,omitempty"`
	Token       string `json:"token,omitempty"`
	IsAdmin     bool   `json:"admin"`

	Stream    bool `json:"stream"`
	OneShot   bool `json:"oneShot"`
	Removable bool `json:"removable"`

	ProtectedByPassword bool   `json:"protectedByPassword"`
	Login               string `json:"login,omitempty"`
	Password            string `json:"password,omitempty"`

	ProtectedByYubikey bool   `json:"protectedByYubikey"`
	Yubikey            string `json:"yubikey,omitempty"`
}

// UnmarshalUpload unmarshal upload, if that fails try again with UploadV1 format with files in a map instead of an array
func UnmarshalUpload(bytes []byte, upload *Upload) (version int, err error) {
	err = json.Unmarshal(bytes, upload)
	if err == nil {
		return 0, nil
	}

	uploadV1 := &UploadV1{}
	err = json.Unmarshal(bytes, uploadV1)
	if err != nil {
		return -1, err
	}

	upload.TTL = uploadV1.TTL
	upload.Comments = uploadV1.Comments

	for _, file := range uploadV1.Files {
		upload.Files = append(upload.Files, file)
	}

	upload.Stream = uploadV1.Stream
	upload.OneShot = uploadV1.OneShot
	upload.Removable = uploadV1.Removable
	upload.Login = uploadV1.Login
	upload.Password = uploadV1.Password

	return 1, nil
}

// MarshalUpload unmarshal upload if version is (1) marshal using UploadV1 format
func MarshalUpload(upload *Upload, version int) (bytes []byte, err error) {
	if version == 0 {
		return json.Marshal(upload)
	}

	if version == 1 {
		uploadV1 := &UploadV1{}

		uploadV1.ID = upload.ID
		uploadV1.Creation = upload.CreatedAt.Unix()
		uploadV1.TTL = upload.TTL
		uploadV1.DownloadDomain = upload.DownloadDomain
		uploadV1.Comments = upload.Comments
		uploadV1.UploadToken = upload.UploadToken

		uploadV1.Stream = upload.Stream
		uploadV1.OneShot = upload.OneShot
		uploadV1.Removable = upload.Removable

		uploadV1.ProtectedByPassword = upload.ProtectedByPassword

		uploadV1.Files = make(map[string]*File)
		for _, file := range upload.Files {
			uploadV1.Files[file.ID] = file
		}

		return json.Marshal(uploadV1)
	}

	return nil, fmt.Errorf("invalid version %d", version)
}

//UploadGormV1 upload object compatible with Plik [1.3->1.3.2]
type UploadGormV1 struct {
	ID  string `json:"id"`
	TTL int    `json:"ttl"`

	DownloadDomain string `json:"downloadDomain"`
	RemoteIP       string `json:"uploadIp,omitempty"`
	Comments       string `json:"comments"`

	Files []*FileGormV1 `json:"files"`

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

func (v1 UploadGormV1) ToUpload() (upload *Upload) {
	upload = &Upload{}
	upload.ID = v1.ID
	upload.TTL = v1.TTL
	//upload.DownloadDomain = v1.DownloadDomain
	upload.RemoteIP = v1.RemoteIP
	upload.Comments = v1.Comments

	for _, file := range v1.Files {
		upload.Files = append(upload.Files, file.ToFile())
	}

	upload.UploadToken = v1.UploadToken
	upload.User = v1.User
	upload.Token = v1.Token
	//upload.IsAdmin = v1.IsAdmin

	upload.Stream = v1.Stream
	upload.OneShot = v1.OneShot
	upload.Removable = v1.Removable

	//upload.ProtectedByPassword = v1.ProtectedByPassword
	upload.Login = v1.Login
	upload.Password = v1.Password

	upload.CreatedAt = v1.CreatedAt
	if v1.DeletedAt != nil {
		//upload.DeletedAt = gorm.DeletedAt{Time: *v1.DeletedAt, Valid: true}
	}
	upload.ExpireAt = v1.ExpireAt

	return upload
}

type FileGormV1 struct {
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

func (v1 FileGormV1) ToFile() (file *File) {
	file = &File{}

	file.ID = v1.ID
	file.UploadID = v1.UploadID
	file.Name = v1.Name
	file.Status = v1.Status
	file.Md5 = v1.Md5
	file.Type = v1.Type
	file.Reference = v1.Reference

	file.BackendDetails = v1.BackendDetails
	file.CreatedAt = v1.CreatedAt

	return file
}
