package exporter

import (
	"github.com/root-gg/plik/server/common"
	"time"
)

// Upload metadata in 1.3 format
type Upload struct {
	ID  string
	TTL int

	DownloadDomain string
	RemoteIP       string
	Comments       string

	Files []*File

	UploadToken string
	User        string
	Token       string
	IsAdmin     bool

	Stream    bool
	OneShot   bool
	Removable bool

	ProtectedByPassword bool
	Login               string
	Password            string

	CreatedAt time.Time
	DeletedAt *time.Time
	ExpireAt  *time.Time
}

// AdaptUpload for 1.3 metadata format
func AdaptUpload(upload *common.Upload) (u *Upload, err error) {
	u = &Upload{}
	u.ID = upload.ID
	u.TTL = upload.TTL
	u.DownloadDomain = upload.DownloadDomain
	u.RemoteIP = upload.RemoteIP
	u.Comments = upload.Comments
	u.UploadToken = upload.UploadToken
	u.UploadToken = upload.User
	u.Token = upload.Token
	u.Stream = upload.Stream
	u.OneShot = upload.OneShot
	u.Removable = upload.Removable
	u.ProtectedByPassword = upload.ProtectedByPassword
	u.Login = upload.Login
	u.Password = upload.Password
	u.CreatedAt = time.Unix(upload.Creation, 0)

	if u.TTL > 0 {
		deadline := u.CreatedAt.Add(time.Duration(u.TTL) * time.Second)
		u.ExpireAt = &deadline
	}

	// Adapt files
	for _, file := range upload.Files {
		f, err := AdaptFile(u, file)
		if err != nil {
			return nil, err
		}
		u.Files = append(u.Files, f)
	}

	return u, nil
}
