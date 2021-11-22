package utils

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

func Md5sum(str string) (md5sum string, err error) {
	h := md5.New()
	_, err = io.WriteString(h, str)
	if err != nil {
		return
	}
	md5sum = fmt.Sprintf("%x", h.Sum(nil))
	return
}

func FileMd5sum(filePath string) (md5sum string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	h := md5.New()
	if _, err = io.Copy(h, file); err != nil {
		return
	}

	md5sum = fmt.Sprintf("%x", h.Sum(nil))
	return
}
