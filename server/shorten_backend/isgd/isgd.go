package isgd

//
//// IS.GD Shortening Backend
//

import (
	"io/ioutil"
	"log"
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

func (sb *ShortenBackendIsGd) Shorten(longUrl string) (string, error) {

	resp, err := client.Get(sb.Url + "&url=" + url.QueryEscape(longUrl))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// Get body
	bodyStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Got url ? :)
	if strings.HasPrefix(string(bodyStr), "http") {
		log.Printf(" - [SHORT] Shortlink successfully created : %s", string(bodyStr))
		return string(bodyStr), nil
	}

	return longUrl, nil
}
