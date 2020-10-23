package handlers

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

type ovhError struct {
	ErrorCode string `json:"errorCode"`
	HTTPCode  string `json:"httpCode"`
	Message   string `json:"message"`
}

type ovhUserConsentResponse struct {
	ValidationURL string `json:"validationUrl"`
	ConsumerKey   string `json:"consumerKey"`
}

type ovhUserResponse struct {
	Nichandle string `json:"nichandle"`
	Email     string `json:"email"`
	FirstName string `json:"firstname"`
	LastName  string `json:"name"`
}

func decodeOVHResponse(resp *http.Response) ([]byte, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body : %s", err)
	}

	if resp.StatusCode > 399 {
		// Decode OVH error information from response
		if body != nil && len(body) > 0 {
			var ovhErr ovhError
			err := json.Unmarshal(body, &ovhErr)
			if err == nil {
				return nil, fmt.Errorf("%s : %s", resp.Status, ovhErr.Message)
			}
			return nil, fmt.Errorf("%s : %s : %s", resp.Status, "unable to deserialize OVH error", string(body))
		}
		return nil, fmt.Errorf("%s", resp.Status)
	}

	return body, nil
}

// OvhLogin return OVH api user consent URL.
func OvhLogin(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	config := ctx.GetConfig()

	if !config.Authentication {
		ctx.BadRequest("authentication is disabled")
		return
	}

	if !config.OvhAuthentication {
		ctx.BadRequest("OVH authentication is disabled")
		return
	}

	// Get redirection URL from the referrer header
	redirectURL, err := getRedirectURL(ctx, "/auth/ovh/callback")
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	// Prepare auth request
	ovhReqBody := "{\"accessRules\":[{\"method\":\"GET\",\"path\":\"/me\"}], \"redirection\":\"" + redirectURL + "\"}"
	u := fmt.Sprintf("%s/auth/credential", config.OvhAPIEndpoint)

	ovhReq, err := http.NewRequest("POST", u, strings.NewReader(ovhReqBody))
	if err != nil {
		ctx.InvalidParameter("unable to create POST request to %s : %s", u, err)
	}
	ovhReq.Header.Add("X-Ovh-Application", config.OvhAPIKey)
	ovhReq.Header.Add("Content-type", "application/json")

	// Do request
	client := &http.Client{}
	ovhResp, err := client.Do(ovhReq)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", u), err)
		return
	}
	defer ovhResp.Body.Close()
	ovhRespBody, err := decodeOVHResponse(ovhResp)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", u), err)
		return
	}

	var userConsentResponse ovhUserConsentResponse
	err = json.Unmarshal(ovhRespBody, &userConsentResponse)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", u), err)
		return
	}

	// Generate session jwt
	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = userConsentResponse.ConsumerKey
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = config.OvhAPIEndpoint

	sessionString, err := session.SignedString([]byte(config.OvhAPISecret))
	if err != nil {
		ctx.InternalServerError("unable to sign OVH session cookie", err)
		return
	}

	// Store temporary session jwt in secure cookie
	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	http.SetCookie(resp, ovhAuthCookie)

	_, _ = resp.Write([]byte(userConsentResponse.ValidationURL))
}

// Remove temporary session cookie
func cleanOvhAuthSessionCookie(resp http.ResponseWriter) {
	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = ""
	ovhAuthCookie.MaxAge = -1
	ovhAuthCookie.Path = "/"
	http.SetCookie(resp, ovhAuthCookie)
}

// OvhCallback authenticate OVH user.
func OvhCallback(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	config := ctx.GetConfig()

	// Remove temporary OVH auth session cookie
	cleanOvhAuthSessionCookie(resp)

	if !config.Authentication {
		ctx.BadRequest("authentication is disabled")
		return
	}

	if config.OvhAPIKey == "" || config.OvhAPISecret == "" || config.OvhAPIEndpoint == "" {
		ctx.InternalServerError("missing OVH API credentials", nil)
		return
	}

	// Get state from secure cookie
	ovhSessionCookie, err := req.Cookie("plik-ovh-session")
	if err != nil || ovhSessionCookie == nil {
		ctx.MissingParameter("OVH session cookie")
		return
	}

	// Parse session cookie
	ovhAuthCookie, err := jwt.Parse(ovhSessionCookie.Value, func(t *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected siging method : %v", t.Header["alg"])
		}

		return []byte(config.OvhAPISecret), nil
	})
	if err != nil {
		ctx.InvalidParameter("OVH session cookie : %s", err)
		return
	}

	// Get OVH consumer key from session
	ovhConsumerKey, ok := ovhAuthCookie.Claims.(jwt.MapClaims)["ovh-consumer-key"]
	if !ok {
		ctx.InvalidParameter("OVH session cookie : missing ovh-consumer-key")

		return
	}

	// Get OVH API endpoint
	endpoint, ok := ovhAuthCookie.Claims.(jwt.MapClaims)["ovh-api-endpoint"]
	if !ok {
		ctx.InvalidParameter("OVH session cookie : missing ovh-api-endpoint")
		return
	}

	// Prepare OVH API /me request
	url := endpoint.(string) + "/me"
	ovhReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", url), err)
		return
	}

	timestamp := time.Now().Unix()
	ovhReq.Header.Add("X-Ovh-Application", config.OvhAPIKey)
	ovhReq.Header.Add("X-Ovh-Timestamp", fmt.Sprintf("%d", timestamp))
	ovhReq.Header.Add("X-Ovh-Consumer", ovhConsumerKey.(string))

	// Sign request
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s+%s+%s+%s+%s+%d",
		config.OvhAPISecret,
		ovhConsumerKey.(string),
		"GET",
		url,
		"",
		timestamp,
	)))
	ovhReq.Header.Add("X-Ovh-Signature", fmt.Sprintf("$1$%x", h.Sum(nil)))

	// Do request
	client := &http.Client{}
	ovhResp, err := client.Do(ovhReq)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", url), err)
		return
	}
	defer ovhResp.Body.Close()
	ovhRespBody, err := decodeOVHResponse(ovhResp)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", url), err)
		return
	}

	// deserialize response
	var userInfo ovhUserResponse
	err = json.Unmarshal(ovhRespBody, &userInfo)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", url), err)
		return
	}

	// Get user from metadata backend
	user, err := ctx.GetMetadataBackend().GetUser(common.GetUserID(common.ProviderOVH, userInfo.Nichandle))
	if err != nil {
		ctx.InternalServerError("unable to get user from metadata backend", err)
		return
	}

	if user == nil {
		if ctx.IsWhitelisted() {
			// Create new user
			user = common.NewUser(common.ProviderOVH, userInfo.Nichandle)
			user.Login = userInfo.Nichandle
			user.Name = userInfo.FirstName + " " + userInfo.LastName
			user.Email = userInfo.Email

			// Save user to metadata backend
			err = ctx.GetMetadataBackend().CreateUser(user)
			if err != nil {
				ctx.InternalServerError("unable to create user in metadata backend", err)
				return
			}
		} else {
			ctx.Forbidden("unable to create user from untrusted source IP address")
			return
		}
	}

	// Set Plik session cookie and xsrf cookie
	sessionCookie, xsrfCookie, err := ctx.GetAuthenticator().GenAuthCookies(user)
	if err != nil {
		ctx.InternalServerError("unable to generate session cookies", err)
	}
	http.SetCookie(resp, sessionCookie)
	http.SetCookie(resp, xsrfCookie)

	http.Redirect(resp, req, config.Path+"/#/login", http.StatusMovedPermanently)
}
