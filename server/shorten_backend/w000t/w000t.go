package w000t

//
//// W000T.ME Shortening Backend
//

import (
	"github.com/root-gg/plik/server/utils"
	"io/ioutil"
	"log"
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

func (sb *ShortenBackendW000t) Shorten(longUrl string) (string, error) {
	str := `{"w000t":{"long_url":"` + longUrl + `", "status":"hidden"}, "token":"` + sb.Token + `" }`
	b := strings.NewReader(str)
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

	// Got url ? :)
	if strings.HasPrefix(string(bodyStr), "http") {
		log.Printf(" - [SHORT] Shortlink successfully created : %s", string(bodyStr))
		return string(bodyStr), nil
	}

	return longUrl, nil
}
