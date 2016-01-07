/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package metadataBackend

import (
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadataBackend/bolt"
	"github.com/root-gg/plik/server/metadataBackend/file"
	"github.com/root-gg/plik/server/metadataBackend/mongo"
)

var metadataBackend MetadataBackend

// MetadataBackend interface describes methods that metadata backends
// must implements to be compatible with plik.
type MetadataBackend interface {
	Create(ctx *common.PlikContext, u *common.Upload) (err error)
	Get(ctx *common.PlikContext, id string) (u *common.Upload, err error)
	AddOrUpdateFile(ctx *common.PlikContext, u *common.Upload, file *common.File) (err error)
	RemoveFile(ctx *common.PlikContext, u *common.Upload, file *common.File) (err error)
	Remove(ctx *common.PlikContext, u *common.Upload) (err error)
	GetUploadsToRemove(ctx *common.PlikContext) (ids []string, err error)

	// Tokens
	SaveToken(ctx *common.PlikContext, t *common.Token) (err error)
	GetToken(ctx *common.PlikContext, token string) (t *common.Token, err error)
	ValidateToken(ctx *common.PlikContext, token string) (ok bool, err error)
	RevokeToken(ctx *common.PlikContext, token string) (err error)
}

// GetMetaDataBackend is a singleton pattern.
// Init static backend if not already and return it
func GetMetaDataBackend() MetadataBackend {
	if metadataBackend == nil {
		Initialize()
	}
	return metadataBackend
}

// Initialize backend from type found in configuration
func Initialize() {
	if metadataBackend == nil {
		switch common.Config.MetadataBackend {
		case "file":
			metadataBackend = file.NewFileMetadataBackend(common.Config.MetadataBackendConfig)
		case "mongo":
			metadataBackend = mongo.NewMongoMetadataBackend(common.Config.MetadataBackendConfig)
		case "bolt":
			metadataBackend = bolt.NewBoltMetadataBackend(common.Config.MetadataBackendConfig)
		default:
			common.Log().Fatalf("Invalid metadata backend %s", common.Config.DataBackend)
		}
	}
}
