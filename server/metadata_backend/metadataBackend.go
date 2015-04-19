package metadata_backend

import (
	"github.com/root-gg/plik/server/metadata_backend/file"
	"github.com/root-gg/plik/server/metadata_backend/mongo"
	"github.com/root-gg/plik/server/utils"
)

var metadataBackend MetadataBackend

type MetadataBackend interface {
	Create(u *utils.Upload) (err error)
	Get(id string) (u *utils.Upload, err error)
	AddOrUpdateFile(u *utils.Upload, file *utils.File) (err error)
	RemoveFile(u *utils.Upload, file *utils.File) (err error)
	Remove(u *utils.Upload) (err error)
	GetUploadsToRemove() (ids []string, err error)
}

func GetMetadataBackend() MetadataBackend {
	if metadataBackend == nil {
		switch utils.Config.MetadataBackend {
		case "file":
			metadataBackend = file.NewFileMetadataBackend(utils.Config.MetadataBackendConfig)
		case "mongo":
			metadataBackend = mongo.NewMongoMetadataBackend(utils.Config.MetadataBackendConfig)
		}
	}

	return metadataBackend
}
