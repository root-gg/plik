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

package w000t

//
//// W000T.ME Shortening Backend
//

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
)

var (
	timeout = time.Duration(time.Second)
	client  = http.Client{Timeout: timeout}
)

// ShortenBackendW000t object
type ShortenBackendW000t struct {
	URL   string
	Token string
}

// NewW000tMeShortenBackend instantiate a shorten backend with
// configuration passed as argument
func NewW000tMeShortenBackend(config map[string]interface{}) *ShortenBackendW000t {
	w000t := new(ShortenBackendW000t)
	w000t.URL = "https://w000t.me/w000ts.text"
	w000t.Token = ""
	utils.Assign(w000t, config)
	return w000t
}

// Shorten implementation for w000t.me shorten backend
func (sb *ShortenBackendW000t) Shorten(ctx *common.PlikContext, longURL string) (shortURL string, err error) {
	defer ctx.Finalize(err)

	// Request short url
	str := `{"w000t":{"long_url":"` + longURL + `", "status":"hidden"}, "token":"` + sb.Token + `" }`
	b := strings.NewReader(str)
	resp, err := client.Post(sb.URL, "application/json", b)
	if err != nil {
		err = ctx.EWarningf("Unable to request short url from w000t.me : %s", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = ctx.EWarningf("Unable to read response from w000t.me : %s", err)
		return
	}

	// Got url ? :)
	if !strings.HasPrefix(string(respBody), "http") {
		err = ctx.EWarningf("Invalid response from w000t.me")
		return
	}

	ctx.Infof("Shortlink successfully created : %s", string(respBody))
	return string(respBody), nil
}
