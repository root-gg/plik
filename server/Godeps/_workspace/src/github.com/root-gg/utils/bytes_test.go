package utils

import (
	"testing"
)

func TestBytesToString(t *testing.T) {

	// Test for all units
	testBytes := BytesToString(123)               // Should get : 123 B
	testKiloBytes := BytesToString(4755)          // Should get : 4.64 KB
	testMegaBytes := BytesToString(6541615)       // Should get : 6.24 MB
	testGigaBytes := BytesToString(2571257332)    // Should get : 2.39 GB

	if testBytes != "123 B" {
		t.Errorf("Unexpected return for %s, got %s, expecting %s", "BytesToString(123)", testBytes, "123 B")
	} else if testKiloBytes != "4.64 KB" {
		t.Errorf("Unexpected return for %s, got %s, expecting %s", "BytesToString(4755)", testBytes, "4.64 KB")
	} else if testMegaBytes != "6.24 MB" {
		t.Errorf("Unexpected return for %s, got %s, expecting %s", "BytesToString(6541615)", testBytes, "6.24 MB")
	} else if testGigaBytes != "2.39 GB" {
		t.Errorf("Unexpected return for %s, got %s, expecting %s", "BytesToString(2571257332)", testBytes, "2.39 GB")
	}
}
