/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
)

func TestOVHLogin(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

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
			require.NoError(t, err, "unable to marshal ovh user consent response")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(500)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start ovh api mock server")

	rr := httptest.NewRecorder()
	OvhLogin(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")

	_, err = url.Parse(string(respBody))
	require.NoError(t, err, "unable to parse ovh auth url")

	var stateString string
	a := rr.Result().Cookies()
	require.NotEmpty(t, a)
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "plik-ovh-session" {
			stateString = cookie.Value
		}
	}
	require.NotEqual(t, "", stateString, "missing ovh session cookie")

	ovhAuthCookie, err := jwt.Parse(stateString, func(t *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method : %v", t.Header["alg"])
		}

		return []byte(context.GetConfig(ctx).OvhAPISecret), nil
	})
	require.NoError(t, err, "unable to parse ovh session string")

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
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	origin := "https://plik.root.gg"
	req.Header.Set("referer", origin)

	handler := func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(500)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start ovh api mock server")

	rr := httptest.NewRecorder()
	OvhLogin(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Error with OVH API : 500 Internal Server Error")
}

func TestOVHLoginInvalidOVHResponse2(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	origin := "https://plik.root.gg"
	req.Header.Set("referer", origin)

	handler := func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("invalid json"))
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start ovh api mock server")

	rr := httptest.NewRecorder()
	OvhLogin(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to unserialize OVH API response")
}

func TestOVHLoginAuthDisabled(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = false
	context.GetConfig(ctx).OvhAuthentication = false

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("referer", "http://plik.root.gg")

	rr := httptest.NewRecorder()
	OvhLogin(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Authentication is disabled")
}

func TestOVHLoginOVHAuthDisabled(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = false

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("referer", "http://plik.root.gg")

	rr := httptest.NewRecorder()
	OvhLogin(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Missing OVH API credentials")
}

func TestOVHLoginMissingReferer(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true

	req, err := http.NewRequest("GET", "/auth/ovh/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	OvhLogin(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Missing referer header")
}

func TestOVHCallback(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	sessionString, err := session.SignedString([]byte(context.GetConfig(ctx).OvhAPISecret))
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

	user := common.NewUser()
	user.ID = "ovh:plik"
	user.Login = ovhUserResponse.Nichandle
	user.Name = ovhUserResponse.FirstName + " " + ovhUserResponse.LastName
	user.Email = ovhUserResponse.Email
	err = addTestUser(ctx, user)
	require.NoError(t, err, "unable to add test user")

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/me" {
			require.Equal(t, context.GetConfig(ctx).OvhAPIKey, req.Header.Get("X-Ovh-Application"))
			require.Equal(t, "consumerKey", req.Header.Get("X-Ovh-Consumer"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Timestamp"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Signature"))

			responseBody, err := json.Marshal(ovhUserResponse)
			require.NoError(t, err, "unable to marshal ovh user response")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(500)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start ovh api mock server")

	rr := httptest.NewRecorder()
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
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("IsWhitelisted", true)

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	sessionString, err := session.SignedString([]byte(context.GetConfig(ctx).OvhAPISecret))
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
			require.Equal(t, context.GetConfig(ctx).OvhAPIKey, req.Header.Get("X-Ovh-Application"))
			require.Equal(t, "consumerKey", req.Header.Get("X-Ovh-Consumer"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Timestamp"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Signature"))

			responseBody, err := json.Marshal(ovhUserResponse)
			require.NoError(t, err, "unable to marshal ovh user response")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(500)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start ovh api mock server")

	rr := httptest.NewRecorder()
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

	user, err := context.GetMetadataBackend(ctx).GetUser(ctx, "ovh:plik", "")
	require.NotNil(t, user, "missing user")
	require.Equal(t, ovhUserResponse.Email, user.Email, "invalid user email")
	require.Equal(t, ovhUserResponse.FirstName+" "+ovhUserResponse.LastName, user.Name, "invalid user name")
}

func TestOVHCallbackCreateUserNotWhitelisted(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("IsWhitelisted", false)

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	sessionString, err := session.SignedString([]byte(context.GetConfig(ctx).OvhAPISecret))
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
			require.Equal(t, context.GetConfig(ctx).OvhAPIKey, req.Header.Get("X-Ovh-Application"))
			require.Equal(t, "consumerKey", req.Header.Get("X-Ovh-Consumer"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Timestamp"))
			require.NotEqual(t, "", req.Header.Get("X-Ovh-Signature"))

			responseBody, err := json.Marshal(ovhUserResponse)
			require.NoError(t, err, "unable to marshal ovh user response")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(500)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start ovh api mock server")

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Unable to create user from untrusted source IP address")
}

func TestOVHCallbackAuthDisabled(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = false

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Authentication is disabled")
}

func TestOVHCallbackMissingOvhAPIConfigParam(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Missing OVH API credentials")
}

func TestOVHCallbackMissingOvhSessionCookie(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Missing OVH session cookie")
}

func TestOVHCallbackMissingSessionString(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	sessionString, err := session.SignedString([]byte(context.GetConfig(ctx).OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestFail(t, rr, http.StatusBadRequest, "Invalid OVH session cookie : missing ovh-consumer-key")
}

func TestOVHCallbackMissingOvhApiEndpoint(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	sessionString, err := session.SignedString([]byte(context.GetConfig(ctx).OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Invalid OVH session cookie : missing ovh-api-endpoint")
}

func TestOVHCallbackMissingOvhApi(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	sessionString, err := session.SignedString([]byte(context.GetConfig(ctx).OvhAPISecret))
	require.NoError(t, err, "unable to generate session string")

	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = int(time.Now().Add(5 * time.Minute).Unix())
	ovhAuthCookie.Path = "/"
	req.AddCookie(ovhAuthCookie)

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Error with ovh API")
}

func TestOVHCallbackInvalidOvhSessionCookie(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

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

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Invalid OVH session cookie")
}

func TestOVHCallbackInvalidOvhApiResponse(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	sessionString, err := session.SignedString([]byte(context.GetConfig(ctx).OvhAPISecret))
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
		resp.WriteHeader(500)
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start ovh api mock server")

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Error with OVH API")
}

func TestOVHCallbackInvalidOvhApiResponseJson(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	context.GetConfig(ctx).Authentication = true
	context.GetConfig(ctx).OvhAuthentication = true
	context.GetConfig(ctx).OvhAPIEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
	context.GetConfig(ctx).OvhAPIKey = "ovh_api_key"
	context.GetConfig(ctx).OvhAPISecret = "ovh_api_secret"

	req, err := http.NewRequest("GET", "/auth/ovh/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	session := jwt.New(jwt.SigningMethodHS256)
	session.Claims.(jwt.MapClaims)["ovh-consumer-key"] = "consumerKey"
	session.Claims.(jwt.MapClaims)["ovh-api-endpoint"] = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	sessionString, err := session.SignedString([]byte(context.GetConfig(ctx).OvhAPISecret))
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
		resp.WriteHeader(500)
		resp.Write([]byte("invalid json"))
	}

	shutdown, err := common.StartAPIMockServer(http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start ovh api mock server")

	rr := httptest.NewRecorder()
	OvhCallback(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Error with OVH API")
}
