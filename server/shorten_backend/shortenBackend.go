package shorten_backend

import (
	"github.com/root-gg/plik/server/shorten_backend/isgd"
	"github.com/root-gg/plik/server/shorten_backend/ovhto"
	"github.com/root-gg/plik/server/shorten_backend/w000t"
	"github.com/root-gg/plik/server/utils"
)

var shortenBackend Shorten

type Shorten interface {
	Shorten(longUrl string) (string, error)
}

func GetShortenBackend() Shorten {
	if shortenBackend == nil {
		switch utils.Config.ShortenBackend {
		case "ovh.to":
			shortenBackend = ovhto.NewOvhToShortenBackend(utils.Config.ShortenBackendConfig)

		case "w000t.me":
			shortenBackend = w000t.NewW000tMeShortenBackend(utils.Config.ShortenBackendConfig)

		case "is.gd":
			shortenBackend = isgd.NewIsGdShortenBackend(utils.Config.ShortenBackendConfig)
		}
	}
	return shortenBackend
}
