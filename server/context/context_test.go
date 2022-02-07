package context

import "github.com/root-gg/plik/server/common"

func newTestContext() *Context {
	return &Context{config: common.NewConfiguration()}
}
