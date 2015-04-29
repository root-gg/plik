package common

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

var yubikeyId string = "13042"
var yubikeyValidationApi string = "https://api.yubico.com/wsapi/verify"

/**
    Check on Yubikey API if a token is valid
    Returns :
        - Boolean : whether or not the token is valid
        - Error   : If an error happens and prevent us to do the check
**/

func YubikeyCheckToken(token string) (valid bool, err error) {

	// Get
	resp, err := http.Get(yubikeyValidationApi + "?id=" + yubikeyId + "&otp=" + token)
	if err != nil {
		return
	}

	// Read response
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	// Right Token?
	if strings.Contains(string(body), "status=OK") {
		valid = true
	} else {
		err = errors.New(string(body))
	}

	return
}
