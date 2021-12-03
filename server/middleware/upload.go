package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/root-gg/utils"

	"github.com/root-gg/plik/server/context"
)

// Upload retrieve the requested upload metadata from the metadataBackend and save it to the request context.
func Upload(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := ctx.GetLogger()

		// Get the upload id from the url params
		vars := mux.Vars(req)
		uploadID := vars["uploadID"]
		if uploadID == "" {
			ctx.MissingParameter("upload id")
			return
		}

		// Get upload metadata
		upload, err := ctx.GetMetadataBackend().GetUpload(uploadID)
		if err != nil {
			ctx.InternalServerError("unable to get upload metadata", err)
			return
		}
		if upload == nil {
			ctx.NotFound("upload %s not found", uploadID)
			return
		}

		// Update request logger prefix
		prefix := fmt.Sprintf("%s[%s]", log.Prefix, uploadID)
		log.SetPrefix(prefix)

		// Test if upload is not expired
		if upload.IsExpired() {
			ctx.NotFound("upload %s has expired", uploadID)
			return
		}

		// Save upload in the request context
		ctx.SetUpload(upload)

		// Being admin of an upload means that you can :
		//  - Add files to the upload
		//  - Remove the upload
		// There are several ways to be considered admin of an upload
		//  - Providing the correct UploadToken (authenticated or not)
		//  - Being authenticated with an Admin user
		//  - Being authenticated with a cookie with the user having created the upload
		//  - Being authenticated with a token with the user and token having create the upload

		upload.IsAdmin = false
		uploadToken := req.Header.Get("X-UploadToken")
		if uploadToken != "" && uploadToken == upload.UploadToken {
			upload.IsAdmin = true
		} else {
			token := ctx.GetToken()
			if token != nil {
				// A user authenticated with a token can manage uploads created with such token
				if upload.Token == token.Token {
					upload.IsAdmin = true
				}
			} else {
				// Check if upload belongs to user or if user is admin
				if ctx.IsAdmin() {
					upload.IsAdmin = true
				} else {
					user := ctx.GetUser()
					if user != nil && upload.User == ctx.GetUser().ID {
						upload.IsAdmin = true
					}
				}
			}
		}

		forbidden := func(message string) {
			resp.Header().Set("WWW-Authenticate", "Basic realm=\"plik\"")

			message = fmt.Sprintf("please provide valid credentials to access this upload : %s", message)

			// Shouldn't redirect here to let the browser ask for credentials and retry
			ctx.SetRedirectOnFailure(false)
			ctx.Fail(message, nil, http.StatusUnauthorized)
		}

		// Handle basic auth if upload is password protected
		if upload.ProtectedByPassword && !upload.IsAdmin {
			if req.Header.Get("Authorization") == "" {
				forbidden("missing Authorization header")
				return
			}

			// Basic auth Authorization header must be set to
			// "Basic base64("login:password")". Only the md5sum
			// of the base64 string is saved in the upload metadata
			auth := strings.Split(req.Header.Get("Authorization"), " ")
			if len(auth) != 2 {
				forbidden("invalid Authorization header")
				return
			}
			if auth[0] != "Basic" {
				forbidden("invalid http authorization scheme")
				return
			}
			var md5sum string
			md5sum, err = utils.Md5sum(auth[1])
			if err != nil {
				forbidden("unable to hash credentials")
				return
			}
			if md5sum != upload.Password {
				forbidden("invalid credentials")
				return
			}
		}

		next.ServeHTTP(resp, req)
	})
}
