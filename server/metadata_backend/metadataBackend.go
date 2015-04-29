package metadata_backend

import (
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadata_backend/file"
	"github.com/root-gg/plik/server/metadata_backend/mongo"
)

var metadataBackend MetadataBackend

type MetadataBackend interface {
	Create(ctx *common.PlikContext, u *common.Upload) (err error)
	Get(ctx *common.PlikContext, id string) (u *common.Upload, err error)
	AddOrUpdateFile(ctx *common.PlikContext, u *common.Upload, file *common.File) (err error)
	RemoveFile(ctx *common.PlikContext, u *common.Upload, file *common.File) (err error)
	Remove(ctx *common.PlikContext, u *common.Upload) (err error)
	GetUploadsToRemove(ctx *common.PlikContext) (ids []string, err error)
}

func GetMetaDataBackend() MetadataBackend {
	if metadataBackend == nil {
		Initialize()
	}
	return metadataBackend
}

func Initialize() {
	if metadataBackend == nil {
		switch common.Config.MetadataBackend {
		case "file":
			metadataBackend = file.NewFileMetadataBackend(common.Config.MetadataBackendConfig)
		case "mongo":
			metadataBackend = mongo.NewMongoMetadataBackend(common.Config.MetadataBackendConfig)
		default:
			common.Log().Fatalf("Invalid metadata backend %s", common.Config.DataBackend)
		}
	}
}
