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

package common

import (
	"crypto/rand"
	"math/big"
	"time"
)

var (
	randRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

// Upload object
type Upload struct {
	ID       string `json:"id" bson:"id"`
	Creation int64  `json:"uploadDate" bson:"uploadDate"`
	TTL      int    `json:"ttl" bson:"ttl"`

	DownloadDomain string `json:"downloadDomain" bson:"-"`
	RemoteIP       string `json:"uploadIp,omitempty" bson:"uploadIp"`
	Comments       string `json:"comments" bson:"comments"`

	Files map[string]*File `json:"files" bson:"files"`

	UploadToken string `json:"uploadToken,omitempty" bson:"uploadToken"`
	User        string `json:"user,omitempty" bson:"user"`
	Token       string `json:"token,omitempty" bson:"token"`
	IsAdmin     bool   `json:"admin"`

	Stream    bool `json:"stream" bson:"stream"`
	OneShot   bool `json:"oneShot" bson:"oneShot"`
	Removable bool `json:"removable" bson:"removable"`

	ProtectedByPassword bool   `json:"protectedByPassword" bson:"protectedByPassword"`
	Login               string `json:"login,omitempty" bson:"login"`
	Password            string `json:"password,omitempty" bson:"password"`

	ProtectedByYubikey bool   `json:"protectedByYubikey" bson:"protectedByYubikey"`
	Yubikey            string `json:"yubikey,omitempty" bson:"yubikey"`

	//ShortURL       string `json:"shortUrl" bson:"shortUrl"` removed v1.2.1
}

// NewUpload instantiate a new upload object
func NewUpload() (upload *Upload) {
	upload = new(Upload)
	upload.Files = make(map[string]*File)
	return
}

// Create fills token, id, date
// We have split in two functions because, the unmarshalling made
// in http handlers would erase the fields
func (upload *Upload) Create() {
	upload.ID = GenerateRandomID(16)
	upload.Creation = time.Now().Unix()
	if upload.Files == nil {
		upload.Files = make(map[string]*File)
	}
	upload.UploadToken = GenerateRandomID(32)
}

// Sanitize removes sensible information from
// object. Used to hide information in API.
func (upload *Upload) Sanitize() {
	upload.RemoteIP = ""
	upload.Password = ""
	upload.Yubikey = ""
	upload.UploadToken = ""
	upload.User = ""
	upload.Token = ""
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
	if upload.TTL > 0 {
		if time.Now().Unix() >= (upload.Creation + int64(upload.TTL)) {
			return true
		}
	}
	return false
}
