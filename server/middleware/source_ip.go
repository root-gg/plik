package middleware

import (
	"fmt"
	"net"
	"net/http"

	"github.com/root-gg/plik/server/context"
)

// SourceIP extract the source IP address from the request and save it to the request context
func SourceIP(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := ctx.GetLogger()
		config := ctx.GetConfig()

		var sourceIPstr string
		if config.SourceIPHeader != "" && req.Header.Get(config.SourceIPHeader) != "" {
			// Get source ip from header if behind reverse proxy.
			sourceIPstr = req.Header.Get(config.SourceIPHeader)
		} else {
			var err error
			sourceIPstr, _, err = net.SplitHostPort(req.RemoteAddr)
			if err != nil {
				ctx.InternalServerError("unable to parse source IP address", err)
				return
			}
		}

		// Parse source IP address
		sourceIP := net.ParseIP(sourceIPstr)
		if sourceIP == nil {
			ctx.InvalidParameter("IP address")
			return
		}

		// Save source IP address in the context
		ctx.SetSourceIP(sourceIP)

		// Update request logger prefix
		prefix := fmt.Sprintf("%s[%s]", log.Prefix, sourceIP.String())
		log.SetPrefix(prefix)

		next.ServeHTTP(resp, req)
	})
}
