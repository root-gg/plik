package utils

import (
	"fmt"
	"path"
	"testing"
)

func TestGetCaller(t *testing.T) {
	file, line, function := GetCaller(1)
	filename := path.Base(file)
	if filename != "caller_test.go" {
		t.Errorf("Invalid file name %s instead of %s", filename, "caller_test.go")
	}
	if line != 10 {
		t.Errorf("Invalid line %d instead of %d", line, 10)
	}
	if function != "github.com/root-gg/utils.TestGetCaller" {
		t.Errorf("Invalid function %s instead of %s", function, "github.com/root-gg/utils.TestGetCaller")
	}
	fmt.Printf("%s:%d : %s\n", file, line, function)
	return
}

func TestParseFunction(t *testing.T) {
	_, _, fct := GetCaller(1)
	pkg, function := ParseFunction(fct)
	if pkg != "github.com/root-gg/utils" {
		t.Errorf("Invalid package name %s instead of %s", pkg, "github.com/root-gg/utils")
	}
	if function != "TestParseFunction" {
		t.Errorf("Invalid package name %s instead of %s", function, "TestParseFunction")
	}
}
