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

type Upload struct {
	Id          string           `json:"id" bson:"id"`
	Creation    int64            `json:"uploadDate" bson:"uploadDate"`
	Comments    string           `json:"comments" bson:"comments"`
	Files       map[string]*File `json:"files" bson:"files"`
	RemoteIp    string           `json:"uploadIp,omitempty" bson:"uploadIp"`
	ShortUrl    string           `json:"shortUrl" bson:"shortUrl"`
	UploadToken string           `json:"uploadToken,omitempty" bson:"uploadToken"`
	Ttl         int              `json:"ttl" bson:"ttl"`

	OneShot   bool `json:"oneShot" bson:"oneShot"`
	Removable bool `json:"removable" bson:"removable"`

	ProtectedByPassword bool   `json:"protectedByPassword" bson:"protectedByPassword"`
	Login               string `json:"login,omitempty" bson:"login"`
	Password            string `json:"password,omitempty" bson:"password"`

	ProtectedByYubikey bool   `json:"protectedByYubikey" bson:"protectedByYubikey"`
	Yubikey            string `json:"yubikey,omitempty" bson:"yubikey"`
}

func NewUpload() (upload *Upload) {
	upload = new(Upload)
	upload.Files = make(map[string]*File)
	return
}

func (upload *Upload) Create() {
	upload.Id = GenerateRandomId(16)
	upload.Creation = time.Now().Unix()
	upload.Files = make(map[string]*File)
	upload.UploadToken = GenerateRandomId(32)
}

func (upload *Upload) Sanitize() {
	upload.RemoteIp = ""
	upload.Password = ""
	upload.Yubikey = ""
	upload.UploadToken = ""
	for _, file := range upload.Files {
		file.Sanitize()
	}
}

var randRunes = []rune("abcdefghijklmnopqrstABCDEFGHIJKLMNOP0123456789")

func GenerateRandomId(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = randRunes[rand.Intn(len(randRunes))]
	}

	return string(b)
}
