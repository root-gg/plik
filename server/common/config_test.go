/*
 * Charles-Antoine Mathieu <charles-antoine.mathieu@ovh.net>
 */

package common

import "testing"

// Test loading the default configuration
func TestLoadConfig(*testing.T) {
	LoadConfiguration("../plikd.cfg")
}
