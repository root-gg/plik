package isgd

//
//// IS.GD Shortening Backend
//

import (
	"github.com/root-gg/plik/server/common"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var timeout = time.Duration(time.Second)
var client = http.Client{Timeout: timeout}

type ShortenBackendIsGd struct {
	Url   string
	Token string
}

func NewIsGdShortenBackend(_ map[string]interface{}) *ShortenBackendIsGd {
	isgd := new(ShortenBackendIsGd)
	isgd.Url = "http://is.gd/create.php?format=simple"
	return isgd
}

func (sb *ShortenBackendIsGd) Shorten(ctx *common.PlikContext, longUrl string) (shortUrl string, err error) {
	defer ctx.Finalize(err)

	// Request short url
	resp, err := client.Get(sb.Url + "&url=" + url.QueryEscape(longUrl))
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
