package juliet

import (
	"fmt"
)

// Context hold a map[interface{}]interface{} to pass along the middleware chain.
type Context struct {
	values map[interface{}]interface{}
}

// NewContext creates a new context instance.
func NewContext() (ctx *Context) {
	ctx = new(Context)
	ctx.values = make(map[interface{}]interface{})
	return
}

// Get returns the value matching the key from the context.
func (ctx *Context) Get(key interface{}) (value interface{}, ok bool) {
	value, ok = ctx.values[key]
	return
}

// Set adds a value to the context or overrides a parent value.
func (ctx *Context) Set(key interface{}, val interface{}) {
	ctx.values[key] = val
}

// Delete remove a value from the context.
func (ctx *Context) Delete(key interface{}) {
	delete(ctx.values, key)
}

// Clear remove all values from the context.
func (ctx *Context) Clear() {
	for key := range ctx.values {
		delete(ctx.values, key)
	}
}

// Copy creates a new copy of the context.
func (ctx *Context) Copy() *Context {
	nc := NewContext()
	for key, value := range ctx.values {
		nc.values[key] = value
	}
	return nc
}

// String returns a string representation of the context values.
func (ctx *Context) String() (str string) {
	for key, value := range ctx.values {
		str += fmt.Sprintf("%v => %v\n", key, value)
	}
	return
}
