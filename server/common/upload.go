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
	"math/rand"
	"time"
)

var (
	randRunes = []rune("abcdefghijklmnopqrstABCDEFGHIJKLMNOP0123456789")
)

// Upload object
type Upload struct {
	ID          string           `json:"id" bson:"id"`
	Creation    int64            `json:"uploadDate" bson:"uploadDate"`
	Comments    string           `json:"comments" bson:"comments"`
	Files       map[string]*File `json:"files" bson:"files"`
	RemoteIP    string           `json:"uploadIp,omitempty" bson:"uploadIp"`
	ShortURL    string           `json:"shortUrl" bson:"shortUrl"`
	UploadToken string           `json:"uploadToken,omitempty" bson:"uploadToken"`
	TTL         int              `json:"ttl" bson:"ttl"`

	Stream    bool `json:"stream" bson:"stream"`
	OneShot   bool `json:"oneShot" bson:"oneShot"`
	Removable bool `json:"removable" bson:"removable"`

	ProtectedByPassword bool   `json:"protectedByPassword" bson:"protectedByPassword"`
	Login               string `json:"login,omitempty" bson:"login"`
	Password            string `json:"password,omitempty" bson:"password"`

	ProtectedByYubikey bool   `json:"protectedByYubikey" bson:"protectedByYubikey"`
	Yubikey            string `json:"yubikey,omitempty" bson:"yubikey"`
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
	for _, file := range upload.Files {
		file.Sanitize()
	}
}

// GenerateRandomID generates a random string with specified length.
// Used to generate upload id, tokens, ...
func GenerateRandomID(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = randRunes[rand.Intn(len(randRunes))]
	}

	return string(b)
}
