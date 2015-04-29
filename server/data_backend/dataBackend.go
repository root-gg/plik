package data_backend

import (
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data_backend/file"
	"github.com/root-gg/plik/server/data_backend/swift"
	"github.com/root-gg/plik/server/data_backend/weedfs"
	"io"
)

var dataBackend DataBackend

type DataBackend interface {
	GetFile(ctx *common.PlikContext, u *common.Upload, id string) (rc io.ReadCloser, err error)
	AddFile(ctx *common.PlikContext, u *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error)
	RemoveFile(ctx *common.PlikContext, u *common.Upload, id string) (err error)
	RemoveUpload(ctx *common.PlikContext, u *common.Upload) (err error)
}

func GetDataBackend() DataBackend {
	if dataBackend == nil {
		Initialize()
	}
	return dataBackend
}

func Initialize() {
	if dataBackend == nil {
		switch common.Config.DataBackend {
		case "file":
			dataBackend = file.NewFileBackend(common.Config.DataBackendConfig)
		case "swift":
			dataBackend = swift.NewSwiftBackend(common.Config.DataBackendConfig)
		case "weedfs":
			dataBackend = weedfs.NewWeedFsBackend(common.Config.DataBackendConfig)
		default:
			common.Log().Fatalf("Invalid data backend %s", common.Config.DataBackend)
		}
	}
}
