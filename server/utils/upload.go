package utils

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
	UploadToken string           `json:"uploadToken" bson:"uploadToken"`
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
