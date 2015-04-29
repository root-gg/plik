package ovhto

//
//// OVH.TO Shortening Backend
//

import (
	"encoding/json"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var timeout = time.Duration(time.Second)
var client = http.Client{Timeout: timeout}

type ShortenBackendOvhTo struct {
	Url string
}

func NewOvhToShortenBackend(config map[string]interface{}) *ShortenBackendOvhTo {
	ovhtobackend := new(ShortenBackendOvhTo)
	ovhtobackend.Url = "http://ovh.to/shorten/"
	utils.Assign(ovhtobackend, config)
	return ovhtobackend
}

func (sb *ShortenBackendOvhTo) Shorten(ctx *common.PlikContext, longUrl string) (shortUrl string, err error) {
	defer ctx.Finalize(err)

	// Request short url
	b := strings.NewReader(`{"longURL":"` + longUrl + `"}`)
	resp, err := client.Post(sb.Url, "application/json", b)
	if err != nil {
		err = ctx.EWarningf("Unable to request short url from ovh.to : %s", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = ctx.EWarningf("Unable to read response from ovh.to : %s", err)
		return
	}

	// Deserialize json response
	responseMap := make(map[string]string)
	err = json.Unmarshal(bodyStr, &responseMap)
	if err != nil {
		err = ctx.EWarningf("Unable to deserialize json response \"%s\" from ovh.to : %s", bodyStr, err)
		return
	}

	// Got url ? :)
	if responseMap["shortURL"] == "" {
		err = ctx.EWarningf("Invalid response from ovh.to")
		return
	}

	ctx.Infof("Shortlink successfully created : %s", responseMap["shortURL"])
	return responseMap["shortURL"], nil
}
