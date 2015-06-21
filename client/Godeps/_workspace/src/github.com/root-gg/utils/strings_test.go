package utils

import (
	"testing"
)

func TestChomp(t *testing.T) {
	str := "foo\n"
	result := Chomp(str)
	if result != "foo" {
		t.Errorf("Invalid string chomp got %s instead of %s", result, "foo")
	}
	str = "bar"
	result = Chomp(str)
	if result != "bar" {
		t.Errorf("Invalid string chomp got %s instead of %s", result, "bar")
	}
}
