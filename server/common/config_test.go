/*
 * Charles-Antoine Mathieu <charles-antoine.mathieu@ovh.net>
 */

package common

import "testing"

// Test loading the default configuration
func TestLoadConfig(t *testing.T) {
	_, err := LoadConfiguration("../plikd.cfg")
	if err != nil {
		t.Error(err)
	}
}
