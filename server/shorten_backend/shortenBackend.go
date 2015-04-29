package shorten_backend

import (
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/shorten_backend/isgd"
	"github.com/root-gg/plik/server/shorten_backend/ovhto"
	"github.com/root-gg/plik/server/shorten_backend/w000t"
)

var shortenBackend ShortenBackend

type ShortenBackend interface {
	Shorten(ctx *common.PlikContext, longUrl string) (string, error)
}

func GetShortenBackend() ShortenBackend {
	if shortenBackend == nil {
		Initialize()
	}
	return shortenBackend
}

func Initialize() {
	if common.Config.ShortenBackend != "" {
		if shortenBackend == nil {
			switch common.Config.ShortenBackend {
			case "ovh.to":
				shortenBackend = ovhto.NewOvhToShortenBackend(common.Config.ShortenBackendConfig)
			case "w000t.me":
				shortenBackend = w000t.NewW000tMeShortenBackend(common.Config.ShortenBackendConfig)
			case "is.gd":
				shortenBackend = isgd.NewIsGdShortenBackend(common.Config.ShortenBackendConfig)
			default:
				common.Log().Fatalf("Invalid shorten backend %s", common.Config.DataBackend)
			}
		}
	}
}
