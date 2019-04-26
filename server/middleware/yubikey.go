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

	"github.com/gorilla/mux"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/context"
)

// Yubikey verify that a valid OTP token has been provided
func Yubikey(ctx *juliet.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := context.GetLogger(ctx)
		config := context.GetConfig(ctx)

		// Get upload from context
		upload := context.GetUpload(ctx)
		if upload == nil {
			// This should never append
			log.Critical("Missing upload in yubikey middleware")
			context.Fail(ctx, req, resp, "Internal error", 500)
			return
		}

		// If upload is yubikey protected, user must send an OTP when he wants to get a file.
		if upload.Yubikey != "" {

			// Error if yubikey is disabled on server, and enabled on upload
			if !config.YubikeyEnabled {
				log.Warningf("Got a Yubikey upload but Yubikey backend is disabled")
				context.Fail(ctx, req, resp, "Yubikey are disabled on this server", 403)
				return
			}

			vars := mux.Vars(req)
			token := vars["yubikey"]
			if token == "" {
				log.Warningf("Missing yubikey token")
				context.Fail(ctx, req, resp, "Invalid yubikey token", 401)
				return
			}
			if len(token) != 44 {
				log.Warningf("Invalid yubikey token : %s", token)
				context.Fail(ctx, req, resp, "Invalid yubikey token", 401)
				return
			}
			if token[:12] != upload.Yubikey {
				log.Warningf("Invalid yubikey device : %s", token)
				context.Fail(ctx, req, resp, "Invalid yubikey token", 401)
				return
			}

			_, isValid, err := config.GetYubiAuth().Verify(token)
			if err != nil {
				log.Warningf("Failed to validate yubikey token : %s", err)
				context.Fail(ctx, req, resp, "Invalid yubikey token", 500)
				return
			}
			if !isValid {
				log.Warningf("Invalid yubikey token : %s", token)
				context.Fail(ctx, req, resp, "Invalid yubikey token", 401)
				return
			}
		}

		next.ServeHTTP(resp, req)
	})
}
