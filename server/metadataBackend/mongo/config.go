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

package mongo

import (
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
)

// MetadataBackendConfig object
type MetadataBackendConfig struct {
	URL             string
	Database        string
	Collection      string
	TokenCollection string
	Username        string
	Password        string
	Ssl             bool
}

// NewMongoMetadataBackendConfig configures the backend
// from config passed as argument
func NewMongoMetadataBackendConfig(config map[string]interface{}) (mbc *MetadataBackendConfig) {
	mbc = new(MetadataBackendConfig)
	mbc.URL = "127.0.0.1:27017"
	mbc.Database = "plik"
	mbc.Collection = "meta"
	mbc.TokenCollection = "tokens"
	utils.Assign(mbc, config)
	return
}
