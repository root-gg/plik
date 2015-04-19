package shorten_backend

import (
	"github.com/root-gg/plik/server/shorten_backend/ovhto"
	"github.com/root-gg/plik/server/shorten_backend/w000t"
	"github.com/root-gg/plik/server/utils"
)

var shortenBackendType string = ""
var shortenBackend Shorten

type Shorten interface {
	Shorten(longUrl string) (string, error)
}

func GetShortenBackend() Shorten {
	if shortenBackendType != "" {
		if shortenBackend == nil {
			switch shortenBackendType {
			case "ovh.to":
				shortenBackend = ovhto.NewOvhToShortenBackend(utils.Config.ShortenBackendConfig)

			case "w000t.me":
				shortenBackend = w000t.NewW000tMeShortenBackend(utils.Config.ShortenBackendConfig)
			}
		}
		return shortenBackend
	} else {
		return nil
	}
}
