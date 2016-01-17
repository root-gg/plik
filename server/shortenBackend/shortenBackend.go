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

package shortenBackend

import (
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/shortenBackend/isgd"
	"github.com/root-gg/plik/server/shortenBackend/w000t"
)

var shortenBackend ShortenBackend

// ShortenBackend interface describes methods that shorten backends
// must implements to be compatible with plik.
type ShortenBackend interface {
	Shorten(ctx *juliet.Context, longURL string) (string, error)
}

// GetShortenBackend is a singleton pattern.
// Init static backend if not already and return it
func GetShortenBackend() ShortenBackend {
	if shortenBackend == nil {
		Initialize()
	}
	return shortenBackend
}

// Initialize backend from type found in configuration
func Initialize() {
	if common.Config.ShortenBackend != "" {
		if shortenBackend == nil {
			switch common.Config.ShortenBackend {
			case "w000t.me":
				shortenBackend = w000t.NewW000tMeShortenBackend(common.Config.ShortenBackendConfig)
			case "is.gd":
				shortenBackend = isgd.NewIsGdShortenBackend(common.Config.ShortenBackendConfig)
			default:
				common.Logger().Fatalf("Invalid shorten backend %s", common.Config.DataBackend)
			}
		}
	}
}
