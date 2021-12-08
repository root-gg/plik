package context

import (
	"fmt"

	"github.com/root-gg/plik/server/common"
)

// ConfigureUploadFromContext assign context values to upload
// This can't be in common.Upload because of common <-> context circular dependency
func (ctx *Context) ConfigureUploadFromContext(upload *common.Upload) (err error) {
	if ctx.GetSourceIP() != nil {
		// Set upload remote IP
		upload.RemoteIP = ctx.GetSourceIP().String()
	}

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

	return nil
}
