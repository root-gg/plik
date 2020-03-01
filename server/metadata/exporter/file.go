package exporter

import (
	"github.com/root-gg/plik/server/common"
	"time"
)

// File object in 1.3 metadata format
type File struct {
	ID       string
	UploadID string
	Name     string

	Status string

	Md5       string
	Type      string
	Size      int64
	Reference string

	BackendDetails string

	CreatedAt time.Time
}

// AdaptFile for 1.3 metadata format
func AdaptFile(u *Upload, file *common.File) (f *File, err error) {
	f = &File{}
	f.ID = file.ID
	f.UploadID = u.ID
	f.Name = file.Name
	f.Status = file.Status
	f.Md5 = file.Md5
	f.Type = file.Type
	f.Size = file.CurrentSize
	f.Reference = file.Reference
	f.CreatedAt = u.CreatedAt

	if f.Status == "downloaded" {
		f.Status = "deleted"
	}

	return f, nil
}
