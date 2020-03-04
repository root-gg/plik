package context

import (
	"net/http"
)

// Middleware is a constructor to close a Context into a middleware
type Middleware func(ctx *Context, next http.Handler) http.Handler

// Handler is a constructor to close a Context into a http.Handler
type Handler func(ctx *Context) http.Handler

// HandlerFunc is a constructor to close a Context into a http.HandlerFunc
type HandlerFunc func(ctx *Context, resp http.ResponseWriter, req *http.Request)

// Chain link context middleware to each other.
// Chain are immutable and any operation on them returns a new Chain object.
type Chain struct {
	parent     *Chain
	middleware Middleware
}

// NewChain creates a new contextMiddleware chain.
func NewChain(cms ...Middleware) (chain *Chain) {
	chain = new(Chain)
	if len(cms) > 0 {
		chain.middleware = cms[0]
		if len(cms) > 1 {
			chain = chain.Append(cms[1:]...)
		}
	}
	return
}

// append add a contextMiddleware to the chain.
func (chain *Chain) append(cm Middleware) (newChain *Chain) {
	newChain = NewChain(cm)
	newChain.parent = chain
	return newChain
}

// Append add contextMiddleware(s) to the chain.
func (chain *Chain) Append(cms ...Middleware) (newChain *Chain) {
	newChain = chain
	for _, cm := range cms {
		newChain = newChain.append(cm)
	}

	return newChain
}

// Adapt add context to a middleware so it can be added to the chain.
func Adapt(fn func(http.Handler) http.Handler) Middleware {
	return func(ctx *Context, h http.Handler) http.Handler {
		return fn(h)
	}
}

// head returns the top/first middleware of the Chain.
func (chain *Chain) head() (head *Chain) {
	// Find the head of the chain
	head = chain
	for head.parent != nil {
		head = head.parent
	}
	return
}

// copy duplicate the whole chain of contextMiddleware.
func (chain *Chain) copy() (newChain *Chain) {
	newChain = NewChain(chain.middleware)
	if chain.parent != nil {
		newChain.parent = chain.parent.copy()
	}
	return
}

// AppendChain duplicates a chain and links it to the current chain.
func (chain *Chain) AppendChain(tail *Chain) (newChain *Chain) {
	// Copy the chain to attach
	newChain = tail.copy()

	// Attach the chain to extend to the new tail
	newChain.head().parent = chain

	// Return the new tail
	return
}

// Then add a HandlerFunc to the end of the chain
// and returns a http.Handler compliant Handler
func (chain *Chain) Then(fn HandlerFunc) (ch *ChainHandler) {
	ch = newHandler(chain, adaptContextHandlerFunc(fn))
	return
}

// ThenHandler add a http.Handler to the end of the chain
// and returns a http.Handler compliant Handler
func (chain *Chain) ThenHandler(handler http.Handler) (ch *ChainHandler) {
	ch = newHandler(chain, adaptHandler(handler))
	return
}

// ThenHandlerFunc add a http.HandlerFunc to the end of the chain
// and returns a http.Handler compliant Handler
func (chain *Chain) ThenHandlerFunc(fn http.HandlerFunc) (ch *ChainHandler) {
	ch = newHandler(chain, adaptHandlerFunc(fn))
	return
}

// ChainHandler holds a chain and a final handler.
// It satisfy the http.Handler interface and can be
// served directly by a net/http server.
type ChainHandler struct {
	chain   *Chain
	handler Handler
}

// New Handler creates a new handler chain.
func newHandler(chain *Chain, handler Handler) (ch *ChainHandler) {
	ch = new(ChainHandler)
	ch.chain = chain
	ch.handler = handler
	return
}

// ServeHTTP builds the chain of handlers in order, closing the context along the way and executes it.
func (ch *ChainHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ctx := &Context{}

	// Build the context handler chain
	handler := ch.handler(ctx)
	chain := ch.chain
	for chain != nil {
		if chain.middleware != nil {
			handler = chain.middleware(ctx, handler)
		}
		chain = chain.parent
	}

	handler.ServeHTTP(resp, req)
}

// Adapt a HandlerFunc into a Handler.
func adaptContextHandlerFunc(fn HandlerFunc) Handler {
	return func(ctx *Context) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fn(ctx, w, r)
		})
	}
}

// Adapt a http.Handler into a Handler.
func adaptHandler(h http.Handler) Handler {
	return func(ctx *Context) http.Handler {
		return h
	}
}

// Adapt a http.HandlerFunc into a Handler.
func adaptHandlerFunc(fn func(w http.ResponseWriter, r *http.Request)) Handler {
	return adaptHandler(http.HandlerFunc(fn))
}
