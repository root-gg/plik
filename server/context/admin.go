package context

// IsAdmin get context user admin status
func (ctx *Context) IsAdmin() bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.user != nil && ctx.user.IsAdmin
}
