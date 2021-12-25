package context

import (
	"fmt"

	"github.com/root-gg/plik/server/common"
)

// ConfigureUploadFromContext assign context values to upload
// This can't be in common.Upload because of common <-> context circular dependency
func (ctx *Context) ConfigureUploadFromContext(upload *common.Upload) (err error) {

	// Set upload remote IP
	if ctx.GetSourceIP() != nil {
		upload.RemoteIP = ctx.GetSourceIP().String()
	}

	// Set upload user
	user := ctx.GetUser()
	if user == nil {
		if ctx.GetConfig().NoAnonymousUploads {
			return fmt.Errorf("anonymous uploads are disabled")
		}
	} else {
		if !ctx.GetConfig().Authentication {
			return fmt.Errorf("authentication is disabled")
		}
		upload.User = user.ID
		token := ctx.GetToken()
		if token != nil {
			upload.Token = token.Token
		}
	}

	// Set upload TTL
	err = upload.SetTTL(ctx.GetConfig().DefaultTTL, ctx.GetMaxTTL())
	if err != nil {
		return err
	}

	return nil
}

// GetMaxFileSize Return the maximum allowed file size
func (ctx *Context) GetMaxFileSize() int64 {
	user := ctx.GetUser()
	if user != nil {
		if user.MaxFileSize > 0 {
			return user.MaxFileSize
		}
	}

	return ctx.GetConfig().MaxFileSize
}

// GetMaxTTL Return the maximum allowed upload TTL
func (ctx *Context) GetMaxTTL() int {
	user := ctx.GetUser()
	if user != nil {
		if user.MaxTTL != 0 {
			return user.MaxTTL
		}
	}

	return ctx.GetConfig().MaxTTL
}
