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

package middleware

import (
	"net/http"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadataBackend"
)

// Authenticate verify that a request has either a whitelisted url or a valid auth token
func Authenticate(ctx *juliet.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := common.GetLogger(ctx)
		log.Debug("Authenticate handler")

		// Get source IP address from context
		sourceIP := common.GetSourceIP(ctx)
		if sourceIP == nil {
			// This should never append
			log.Critical("Missing sourceIP in Authenticate handler")
			common.Fail(ctx, req, resp, "Internal error", 500)
			return
		}

		if common.IsWhitelisted(ctx) {
			// Source IP address is in the whitelist
			next.ServeHTTP(resp, req)
			return
		}

		// Check if a valid token had been provided
		if common.Config.TokenAuthentication {
			token := req.Header.Get("X-AuthToken")
			if token != "" {
				ok, err := metadataBackend.GetMetaDataBackend().ValidateToken(ctx, token)
				if err != nil {
					log.Warningf("Unable to validate token %s : %s", token, err)
					common.Fail(ctx, req, resp, "Unable to validate token", 403)
					return
				}
				if !ok {
					log.Warningf("Invalid token %s : %s", token, err)
					common.Fail(ctx, req, resp, "Invalid token", 403)

					return
				}

				// Valid token
				next.ServeHTTP(resp, req)
				return
			}
		}

		// Invalid source IP address + no token
		log.Warningf("Unauthorized source IP address")
		common.Fail(ctx, req, resp, "Unauthorized source IP address", 403)
		return
	})
}
