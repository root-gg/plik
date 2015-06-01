package utils

import (
	"testing"
)

type TestDumper struct {
	Foo string
}

func TestDump(t *testing.T) {
	Dump(TestDumper{"bar"})
}

func TestSdump(t *testing.T) {
	dump := Sdump(TestDumper{"bar"})
	expected := "{\n  \"Foo\": \"bar\"\n}"
	if dump != expected {
		t.Errorf("Invalid dump got %s instead of %s", dump, expected)
	}
}
