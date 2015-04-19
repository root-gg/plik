package ovhto

//
//// OVH.TO Shortening Backend
//

import (
	"encoding/json"
	"github.com/root-gg/plik/server/utils"
	"io/ioutil"
	"log"
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

func (sb *ShortenBackendOvhTo) Shorten(longUrl string) (string, error) {
	b := strings.NewReader(`{"longURL":"` + longUrl + `"}`)
	resp, err := client.Post(sb.Url, "application/json", b)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// Get body
	bodyStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Decode it
	responseMap := make(map[string]string)
	err = json.Unmarshal(bodyStr, &responseMap)
	if err != nil {
		return "", err
	}

	// Got url ? :)
	if responseMap["shortURL"] != "" {
		log.Printf(" - [SHORT] Shortlink successfully created : %s", responseMap["shortURL"])
		return responseMap["shortURL"], nil
	}

	return longUrl, nil
}
