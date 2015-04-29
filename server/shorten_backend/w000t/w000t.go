package w000t

//
//// W000T.ME Shortening Backend
//

import (
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var timeout = time.Duration(time.Second)
var client = http.Client{Timeout: timeout}

type ShortenBackendW000t struct {
	Url   string
	Token string
}

func NewW000tMeShortenBackend(config map[string]interface{}) *ShortenBackendW000t {
	w000t := new(ShortenBackendW000t)
	w000t.Url = "https://w000t.me/w000ts.text"
	w000t.Token = ""
	utils.Assign(w000t, config)
	return w000t
}

func (sb *ShortenBackendW000t) Shorten(ctx *common.PlikContext, longUrl string) (shortUrl string, err error) {
	defer ctx.Finalize(err)

	// Request short url
	str := `{"w000t":{"long_url":"` + longUrl + `", "status":"hidden"}, "token":"` + sb.Token + `" }`
	b := strings.NewReader(str)
	resp, err := client.Post(sb.Url, "application/json", b)
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
