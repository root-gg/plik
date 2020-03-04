package context

// IsWhitelisted get isWhitelisted from the context.
func (ctx *Context) IsWhitelisted() bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	if ctx.isWhitelisted != nil {
		// Return cached result
		return *ctx.isWhitelisted
	}

	if ctx.user != nil {
		// IP Restriction does not apply to authenticated users
		return true
	}

	// Check if the IP is whitelisted
	isWhitelisted := ctx.config.IsWhitelisted(ctx.sourceIP)

	// Cache result
	ctx.isWhitelisted = &isWhitelisted

	return isWhitelisted
}

// SetWhitelisted set isWhitelisted in the context
func (ctx *Context) SetWhitelisted(isWhitelisted bool) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.isWhitelisted = &isWhitelisted
}
