/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

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
