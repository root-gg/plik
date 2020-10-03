package handlers

import (
	"fmt"
	"image/png"
	"net/http"
	"net/url"
	"strconv"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// GetVersion return the build information.
func GetVersion(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	bi := common.GetBuildInfo()
	if ctx.GetConfig().EnhancedWebSecurity {
		// Remove sensible info from BuildInfo
		bi.Sanitize()
	}
	common.WriteJSONResponse(resp, bi)
}

// GetConfiguration return the server configuration
func GetConfiguration(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	common.WriteJSONResponse(resp, ctx.GetConfig())
}

// Logout return the server configuration
func Logout(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	common.Logout(resp, ctx.GetAuthenticator())
}

// GetQrCode return a QRCode for the requested URL
func GetQrCode(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Check params
	urlParam := req.FormValue("url")
	sizeParam := req.FormValue("size")

	// Parse int on size
	sizeInt, err := strconv.Atoi(sizeParam)
	if err != nil {
		sizeInt = 250
	}
	if sizeInt <= 0 {
		ctx.BadRequest("QRCode size must be positive")
		return
	}
	if sizeInt > 1000 {
		ctx.BadRequest("QRCode size must be lower than 1000")
		return
	}

	// Generate QRCode png from url
	qrcode, err := qr.Encode(urlParam, qr.H, qr.Auto)
	if err != nil {
		ctx.InternalServerError("unable to generate QRCode", err)
		return
	}

	// Scale QRCode png size
	qrcode, err = barcode.Scale(qrcode, sizeInt, sizeInt)
	if err != nil {
		ctx.InternalServerError("unable to scale QRCode : %s", err)
		return
	}

	resp.Header().Add("Content-Type", "image/png")
	err = png.Encode(resp, qrcode)
	if err != nil {
		ctx.InternalServerError("unable to encore png : %s", err)
		return
	}
}

// If a download domain is specified verify that the request comes from this specific domain
func checkDownloadDomain(ctx *context.Context) bool {
	log := ctx.GetLogger()
	config := ctx.GetConfig()
	req := ctx.GetReq()
	resp := ctx.GetResp()

	if config.GetDownloadDomain() != nil {
		if req.Host != config.GetDownloadDomain().Host {
			downloadURL := fmt.Sprintf("%s://%s%s",
				config.GetDownloadDomain().Scheme,
				config.GetDownloadDomain().Host,
				req.RequestURI)
			log.Warningf("invalid download domain %s, expected %s", req.Host, config.GetDownloadDomain().Host)
			http.Redirect(resp, req, downloadURL, http.StatusMovedPermanently)
			return false
		}
	}

	return true
}

func getRedirectURL(ctx *context.Context, callbackPath string) (redirectURL string, err error) {
	req := ctx.GetReq()

	referer := req.Header.Get("referer")
	if referer == "" {
		return "", common.NewHTTPError("missing referer header", nil, http.StatusBadRequest)
	}

	originURL, err := url.Parse(referer)
	if err != nil {
		return "", common.NewHTTPError("invalid referer header", nil, http.StatusBadRequest)
	}

	redirectURL = fmt.Sprintf("%s://%s", originURL.Scheme, originURL.Host)
	if ctx.GetConfig().Path != "" {
		redirectURL += ctx.GetConfig().Path
	}
	redirectURL += callbackPath

	return redirectURL, nil
}

func handleHTTPError(ctx *context.Context, err error) {
	if httpError, ok := err.(common.HTTPError); ok {
		ctx.Fail(httpError.Message, httpError.Err, httpError.StatusCode)
	} else {
		ctx.InternalServerError("unexpected error", err)
	}
}
