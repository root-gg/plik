package context

import "github.com/root-gg/plik/server/common"

// IsAdmin get context user admin status
func (ctx *Context) IsAdmin() bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	user := ctx.user

	if ctx.originalUser != nil {
		user = ctx.originalUser
	}

	return user != nil && user.IsAdmin
}

// GetOriginalUser get original user in the context
func (ctx *Context) GetOriginalUser() (user *common.User) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if ctx.originalUser != nil {
		return ctx.originalUser
	}

	return ctx.user
}

// SaveOriginalUser save the current user in the context
func (ctx *Context) SaveOriginalUser() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.originalUser = ctx.user
}
