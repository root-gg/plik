package tar

import (
	"github.com/root-gg/utils"
)

// BackendConfig object
type BackendConfig struct {
	Tar      string
	Compress string
	Options  string
}

// NewTarBackendConfig instantiate a new Backend Configuration
// from config map passed as argument
func NewTarBackendConfig(config map[string]interface{}) (tb *BackendConfig) {
	tb = new(BackendConfig)
	tb.Tar = "/bin/tar"
	tb.Compress = "gzip"
	utils.Assign(tb, config)
	return
}
