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

package isgd

//
//// IS.GD Shortening Backend
//

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/root-gg/plik/server/common"
)

var (
	timeout = time.Duration(time.Second)
	client  = http.Client{Timeout: timeout}
)

// ShortenBackendIsGd object
type ShortenBackendIsGd struct {
	URL   string
	Token string
}

// NewIsGdShortenBackend instantiate a shorten backend with
// configuration passed as argument
func NewIsGdShortenBackend(_ map[string]interface{}) *ShortenBackendIsGd {
	isgd := new(ShortenBackendIsGd)
	isgd.URL = "http://is.gd/create.php?format=simple"
	return isgd
}

// Shorten implementation for is.gd shorten backend
func (sb *ShortenBackendIsGd) Shorten(ctx *common.PlikContext, longURL string) (shortURL string, err error) {
	defer ctx.Finalize(err)

	// Request short url
	resp, err := client.Get(sb.URL + "&url=" + url.QueryEscape(longURL))
	if err != nil {
		err = ctx.EWarningf("Unable to request short url from is.gd : %s", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = ctx.EWarningf("Unable to read response from is.gd : %s", err)
		return
	}

	// Got url ? :)
	if !strings.HasPrefix(string(respBody), "http") {
		err = ctx.EWarningf("Invalid response from is.gd")
		return
	}

	ctx.Infof("Shortlink successfully created : %s", string(respBody))
	return string(respBody), nil
}
