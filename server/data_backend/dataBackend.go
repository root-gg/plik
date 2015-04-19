package data_backend

import (
	"github.com/root-gg/plik/server/data_backend/file"
	"github.com/root-gg/plik/server/data_backend/swift"
	"github.com/root-gg/plik/server/data_backend/weedfs"
	"github.com/root-gg/plik/server/utils"
	"io"
)

var dataBackend DataBackend

type DataBackend interface {
	GetFile(u *utils.Upload, id string) (rc io.ReadCloser, err error)
	AddFile(u *utils.Upload, file *utils.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error)
	RemoveFile(u *utils.Upload, id string) (err error)
	RemoveUpload(u *utils.Upload) (err error)
}

func GetDataBackend() DataBackend {
	if dataBackend == nil {
		switch utils.Config.DataBackend {
		case "file":
			dataBackend = file.NewFileBackend(utils.Config.DataBackendConfig)
		case "swift":
			dataBackend = swift.NewSwiftBackend(utils.Config.DataBackendConfig)
		case "weedfs":
			dataBackend = weedfs.NewWeedFsBackend(utils.Config.DataBackendConfig)
		}
	}

	return dataBackend
}
