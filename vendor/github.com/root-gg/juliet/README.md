
Juliet is a lightweight middleware chaining helper that pass a Context (map) object
from a middleware to the next one.

This work is inspired by [Stack](https://github.com/alexedwards/stack) by Alex Edwards 
and [Alice](https://github.com/justinas/alice) by Justinas Stankevicius.

Godoc is here : https://godoc.org/github.com/root-gg/juliet   

And there is a working example in the examples package :   

```go
package main

import (
	"net/http"
	"log"
	"net"
	"fmt"

	"github.com/root-gg/juliet"
)

// Juliet is a lightweight middleware chaining helper that pass a Context (map) object
// from a middleware to the next one.
//
// Middlewre is a pattern where http request/response are passed through many handlers to reuse code.

// For example this classic middleware log the requested url
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Our middleware logic goes here...
		log.Println(r.URL.String())

		// Pass the request to the next middleware / handler
		next.ServeHTTP(w, r)
	})
}

// Juliet adds a context parameter to the middleware pattern that will be passed along the Chain.
// For example this middleware puts the request's source IP address in the context.
func getSourceIpMiddleware(ctx *juliet.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Get the source IP from the request remote address
		ip, _, err := net.SplitHostPort(r.RemoteAddr)

		// You can handle failure at any point of the chain by not calling next.ServeHTTP
		if err != nil {
			http.Error(w, "Unable to parse source ip address", 500)
			return
		}

		// Save the source ip in the context
		ctx.Set("ip", ip)

		// Pass the request to the next middleware / handler
		next.ServeHTTP(w, r)
	})
}

// As a context is nothing more that a map[interface{}]interface{} with syntactic sugar you have to ensure you
// check types of values you get from the context. To keep your code clean you can write helpers to do that and keep
// type safety everywhere.
func getSourceIp(ctx *juliet.Context) string {
	if sourceIP, ok := ctx.Get("ip"); ok {
		return sourceIP.(string)
	}
	return ""
}

// The last link of a middleware chain is the application Handler
func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong\n"))
}

// Juliet can also pass the context parameter to application Handlers
func ipHandler(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	// write http response
	resp.Write([]byte(fmt.Sprintf("your IP address is : %s\n", getSourceIp(ctx))))
}

func main(){
	// Juliet links middleware and handler with chain objects
	chain := juliet.NewChain()

	// Chain objects are immutable and any operation on it returns a new chain object.
	// You can append one or more middleware to the chain at a time using the Append method.
	chain = chain.Append(getSourceIpMiddleware)

	// You can append a middleware to the beginning of the chain with the AppendChain method.
	// When working with a non context-aware ( Juliet ) middleware you have to use the Adapt method.
	chain = juliet.NewChain(juliet.Adapt(logMiddleware)).AppendChain(chain)

	// Now we have this middleware chain : logUrl > getSourceIp > ApplicationHandler
	// We could have built it in one pass this way :
	//  chain := juliet.NewChain(juliet.Adapt(logMiddleware),getSourceIpMiddleware)

	// It's now time to add some application handlers and to bind everything to some HTTP route.

	// With a context handler
	http.Handle("/ip", chain.Then(ipHandler))

	// With a classic http.HandlerFunc
	http.Handle("/ping", chain.ThenHandlerFunc(pingHandler))

	// With a classic http.Handler
	http.Handle("/404", chain.ThenHandler(http.NotFoundHandler()))

	log.Fatal(http.ListenAndServe(":1234", nil))
}

// $ go run main.go
// 2016/02/01 12:20:39 /ip
// 2016/02/01 12:20:44 /ping
// 2016/02/01 12:20:56 /404
//
// $ curl 127.0.0.1:1234/ip
// your IP address is : 127.0.0.1
// $ curl 127.0.0.1:1234/ping
// pong
// $ curl 127.0.0.1:1234/404
// 404 page not found
```
