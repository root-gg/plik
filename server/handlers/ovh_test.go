package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestOVHLogin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	origin := "https://plik.root.gg"
	req.Header.Set("referer", origin)

	ovhUserConsentResponse := &ovhUserConsentResponse{
		ValidationURL: "http://127.0.0.1:8765/auth/validation",
		ConsumerKey:   "consumerKey",
	}

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/auth/credential" {
			responseBody, err := json.Marshal(ovhUserConsentResponse)
			require.NoError(t, err, "unable to marshal OVH user consent response")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start OVH api mock server")

	rr := ctx.NewRecorder(req)
	OvhLogin(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")

	_, err = url.Parse(string(respBody))
	require.NoError(t, err, "unable to parse OVH auth url")

	var stateString string
	a := rr.Result().Cookies()
	require.NotEmpty(t, a)
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "plik-ovh-session" {
			stateString = cookie.Value
		}
	}
	require.NotEqual(t, "", stateString, "context.TestPanic(t, rr,")

	ovhAuthCookie, err := jwt.Parse(stateString, func(t *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method : %v", t.Header["alg"])
		}

		return []byte(ctx.GetConfig().OvhAPISecret), nil
	})
	require.NoError(t, err, "unable to parse OVH session string")

	// Get OVH consumer key from session
	ovhConsumerKey, ok := ovhAuthCookie.Claims.(jwt.MapClaims)["ovh-consumer-key"]
	require.True(t, ok, "missing ovh-consumer-key")
	require.Equal(t, ovhUserConsentResponse.ConsumerKey, ovhConsumerKey)

	// Get OVH API endpoint
	ovhAuthEndpoint, ok := ovhAuthCookie.Claims.(jwt.MapClaims)["ovh-api-endpoint"]
	require.True(t, ok, "missing ovh-api-endpoint")
	require.NotEqual(t, ovhUserConsentResponse.ValidationURL, ovhAuthEndpoint)
}

func TestOVHLoginInvalidOVHResponse(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	origin := "https://plik.root.gg"
	req.Header.Set("referer", origin)

	handler := func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusInternalServerError)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start OVH api mock server")

	rr := ctx.NewRecorder(req)
	OvhLogin(ctx, rr, req)
	context.TestInternalServerError(t, rr, "error with OVH API")
}

func TestOVHLoginInvalidOVHResponse2(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	origin := "https://plik.root.gg"
	req.Header.Set("referer", origin)

	handler := func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("invalid json"))
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start OVH api mock server")

	rr := ctx.NewRecorder(req)
	OvhLogin(ctx, rr, req)
	context.TestInternalServerError(t, rr, "error with OVH API")

}

func TestOVHLoginAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = false
	ctx.GetConfig().OvhAuthentication = false

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("referer", "http://plik.root.gg")

	rr := ctx.NewRecorder(req)
	OvhLogin(ctx, rr, req)
	context.TestBadRequest(t, rr, "authentication is disabled")
}

func TestOVHLoginOVHAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = false

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("referer", "http://plik.root.gg")

	rr := ctx.NewRecorder(req)
	OvhLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "OVH authentication is disabled")
}

func TestOVHLoginMissingReferer(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OvhLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing referer header")
}

func TestOVHCallback(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	sessionString, err := session.SignedString([]byte(ctx.GetConfig().OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	ovhUserResponse := &ovhUserResponse{
		Nichandle: "plik",
		FirstName: "plik",
		LastName:  "root-gg",
		Email:     "plik@root.gg",
	}

	user := common.NewUser("ovh", "plik")
	user.ID = "ovh:plik"
	user.Login = ovhUserResponse.Nichandle
	user.Name = ovhUserResponse.FirstName + " " + ovhUserResponse.LastName
	user.Email = ovhUserResponse.Email
	err = ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/me" {
			require.Equal(t, ctx.GetConfig().OvhAPIKey, req.Header.Get("X-Ovh-Application"))
			require.Equal(t, "consumerKey", req.Header.Get("X-Ovh-Consumer"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Timestamp"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Signature"))

			responseBody, err := json.Marshal(ovhUserResponse)
			require.NoError(t, err, "unable to marshal OVH user response")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start OVH api mock server")

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, 301, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")

	var sessionCookie string
	var xsrfCookie string
	a := rr.Result().Cookies()
	require.NotEmpty(t, a)
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "plik-session" {
			sessionCookie = cookie.Value
		}
		if cookie.Name == "plik-xsrf" {
			xsrfCookie = cookie.Value
		}
	}

	require.NotEqual(t, "", sessionCookie, "missing plik session cookie")
	require.NotEqual(t, "", xsrfCookie, "missing plik xsrf cookie")
}

func TestOVHCallbackCreateUser(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetWhitelisted(true)

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	sessionString, err := session.SignedString([]byte(ctx.GetConfig().OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	ovhUserResponse := &ovhUserResponse{
		Nichandle: "plik",
		FirstName: "plik",
		LastName:  "root-gg",
		Email:     "plik@root.gg",
	}

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/me" {
			require.Equal(t, ctx.GetConfig().OvhAPIKey, req.Header.Get("X-Ovh-Application"))
			require.Equal(t, "consumerKey", req.Header.Get("X-Ovh-Consumer"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Timestamp"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Signature"))

			responseBody, err := json.Marshal(ovhUserResponse)
			require.NoError(t, err, "unable to marshal OVH user response")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start OVH api mock server")

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, 301, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")

	var sessionCookie string
	var xsrfCookie string
	a := rr.Result().Cookies()
	require.NotEmpty(t, a)
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "plik-session" {
			sessionCookie = cookie.Value
		}
		if cookie.Name == "plik-xsrf" {
			xsrfCookie = cookie.Value
		}
	}

	require.NotEqual(t, "", sessionCookie, "missing plik session cookie")
	require.NotEqual(t, "", xsrfCookie, "missing plik xsrf cookie")

	user, err := ctx.GetMetadataBackend().GetUser("ovh:plik")
	require.NotNil(t, user, "missing user")
	require.Equal(t, ovhUserResponse.Email, user.Email, "invalid user email")
	require.Equal(t, ovhUserResponse.FirstName+" "+ovhUserResponse.LastName, user.Name, "invalid user name")
}

func TestOVHCallbackCreateUserNotWhitelisted(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetWhitelisted(false)

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	sessionString, err := session.SignedString([]byte(ctx.GetConfig().OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	ovhUserResponse := &ovhUserResponse{
		Nichandle: "plik",
		FirstName: "plik",
		LastName:  "root-gg",
		Email:     "plik@root.gg",
	}

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/me" {
			require.Equal(t, ctx.GetConfig().OvhAPIKey, req.Header.Get("X-Ovh-Application"))
			require.Equal(t, "consumerKey", req.Header.Get("X-Ovh-Consumer"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Timestamp"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Signature"))

			responseBody, err := json.Marshal(ovhUserResponse)
			require.NoError(t, err, "unable to marshal OVH user response")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start OVH api mock server")

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "unable to create user from untrusted source IP address")
}

func TestOVHCallbackAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = false

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "authentication is disabled")
}

func TestOVHCallbackMissingOvhAPIConfigParam(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)
	context.TestInternalServerError(t, rr, "missing OVH API credentials")
}

func TestOVHCallbackMissingOvhSessionCookie(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing OVH session cookie")
}

func TestOVHCallbackMissingSessionString(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	sessionString, err := session.SignedString([]byte(ctx.GetConfig().OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestBadRequest(t, rr, "invalid OVH session cookie : missing ovh-consumer-key")
}

func TestOVHCallbackMissingOvhApiEndpoint(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	sessionString, err := session.SignedString([]byte(ctx.GetConfig().OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid OVH session cookie : missing ovh-api-endpoint")
}

func TestOVHCallbackMissingOvhApi(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	sessionString, err := session.SignedString([]byte(ctx.GetConfig().OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)
	context.TestInternalServerError(t, rr, "error with OVH API")
}

func TestOVHCallbackInvalidOvhSessionCookie(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = "invalid session cookie"
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid OVH session cookie")
}

func TestOVHCallbackInvalidOvhApiResponse(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	sessionString, err := session.SignedString([]byte(ctx.GetConfig().OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	handler := func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusInternalServerError)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start OVH api mock server")

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)
	context.TestInternalServerError(t, rr, "error with OVH API")
}

func TestOVHCallbackInvalidOvhApiResponseJson(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Authentication = true
	ctx.GetConfig().OvhAuthentication = true
	ctx.GetConfig().OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	ctx.GetConfig().OvhAPIKey = "ovh_api_key"
	ctx.GetConfig().OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	sessionString, err := session.SignedString([]byte(ctx.GetConfig().OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	handler := func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte("invalid json"))
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start OVH api mock server")

	rr := ctx.NewRecorder(req)
	OvhCallback(ctx, rr, req)
	context.TestInternalServerError(t, rr, "error with OVH API")
}
