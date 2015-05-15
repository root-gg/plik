package utils

import (
	"crypto/md5"
	"fmt"
	"io"
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
