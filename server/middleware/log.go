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
	"net/http/httputil"
	"strings"

	"github.com/root-gg/juliet"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/context"
)

// Log the http request
func Log(ctx *juliet.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := context.GetLogger(ctx)

		if log.LogIf(logger.DEBUG) {

			// Don't dump request body for file upload
			dumpBody := true
			if (strings.HasPrefix(req.URL.Path, "/file") ||
				strings.HasPrefix(req.URL.Path, "/stream")) &&
				req.Method == "POST" {
				dumpBody = false
			}

			// Dump the full http request
			dump, err := httputil.DumpRequest(req, dumpBody)
			if err == nil {
				log.Debug(string(dump))
			} else {
				log.Warningf("Unable to dump HTTP request : %s", err)
			}
		} else {
			log.Infof("%v %v", req.Method, req.RequestURI)
		}

		next.ServeHTTP(resp, req)
	})
}
