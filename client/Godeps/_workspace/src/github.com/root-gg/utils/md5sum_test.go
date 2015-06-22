package utils

import (
	"testing"
	"os"
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

func TestFileMd5sum(t *testing.T) {
	path := os.TempDir() + "/" + "testFileMd5Sum"
	f, err := os.Create(path)
	if err != nil {
		t.Errorf("Unable to open test file %s : %s", path, err)
	}
	_, err = f.Write([]byte("Lorem ipsum dolor sit amet"))
	if err != nil {
		t.Errorf("Unable to write test file %s : %s", path, err)
	}
	err = f.Close()
	if err != nil {
		t.Errorf("Unable to close test file %s : %s", path, err)
	}
	md5sum, err := FileMd5sum(path)
	if err != nil {
		t.Errorf("Unable to compute md5sum : %s", err)
	}
	sum := "fea80f2db003d4ebc4536023814aa885"
	if md5sum != sum {
		t.Errorf("Invalid md5sum got %s instead of %s", md5sum, sum)
	}
	err = os.Remove(path)
	return
}
