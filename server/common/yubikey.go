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

package common

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var yubikeyId string = "13042"
var yubikeyValidationApi string = "https://api.yubico.com/wsapi/verify"
var yubikeyTimeout = time.Duration(time.Second * 2)
var yubikeyClient = http.Client{Timeout: yubikeyTimeout}

/**
    Check on Yubikey API if a token is valid
    Returns :
        - Boolean : whether or not the token is valid
        - Error   : If an error happens and prevent us to do the check
**/

func YubikeyCheckToken(ctx *PlikContext, token string) (valid bool, err error) {

	// Get
	resp, err := yubikeyClient.Get(yubikeyValidationApi + "?id=" + yubikeyId + "&otp=" + token)
	if err != nil {
		return
	}

	// Read response
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	// Get status line in response
	status := ""
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "status=") {
			statusArray := strings.Split(line, "=")
			if len(statusArray) > 1 {
				status = strings.TrimSpace(statusArray[1])
			}
		}
	}

	// Right Token?
	if status == "OK" {
		ctx.Debugf("Yubikey OTP %s is valid", token)
		valid = true
	} else {
		err = ctx.EWarningf("%s", status)
	}

	return
}
