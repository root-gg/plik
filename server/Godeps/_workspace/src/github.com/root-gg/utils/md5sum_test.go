package utils

import (
	"testing"
)

func TestMd5sum(t *testing.T) {
	md5sum, err := Md5sum("Lorem ipsum dolor sit amet")
	if err != nil {
		t.Errorf("Unable to compute md5sum : %s", err)
	}
	sum := "fea80f2db003d4ebc4536023814aa885"
	if md5sum != sum {
		t.Errorf("Invalid md5sum got %s instead of %s", md5sum, sum)
	}
	return
}
