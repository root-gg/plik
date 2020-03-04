package zip

import (
	"github.com/root-gg/utils"
)

// BackendConfig object
type BackendConfig struct {
	Zip     string
	Options string
}

// NewZipBackendConfig instantiate a new Backend Configuration
// from config map passed as argument
func NewZipBackendConfig(config map[string]interface{}) (zb *BackendConfig) {
	zb = new(BackendConfig)
	zb.Zip = "/bin/zip"
	utils.Assign(zb, config)
	return
}
