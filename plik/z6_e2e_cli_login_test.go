package plik

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/handlers"
)

// loginAndGetXSRF logs in with local credentials and returns the browser client
// with session cookies set and the XSRF token value for use in POST headers.
func loginAndGetXSRF(t *testing.T, baseURL string, login string, password string) (*http.Client, string) {
	t.Helper()
	browserClient := newBrowserClient()
	loginBody := `{"login":"` + login + `","password":"` + password + `"}`
	loginResp, err := browserClient.Post(baseURL+"/auth/local/login", "application/json", strings.NewReader(loginBody))
	require.NoError(t, err, "browser login failed")
	defer loginResp.Body.Close()
	require.Equal(t, http.StatusOK, loginResp.StatusCode)

	sessionCookie := getCookie(loginResp, common.SessionCookieName)
	require.NotNil(t, sessionCookie, "missing session cookie")
	xsrfCookie := getCookie(loginResp, common.XSRFCookieName)
	require.NotNil(t, xsrfCookie, "missing xsrf cookie")

	serverURL, _ := url.Parse(baseURL)
	browserClient.Jar.SetCookies(serverURL, []*http.Cookie{sessionCookie, xsrfCookie})

	return browserClient, xsrfCookie.Value
}

func TestCLIAuth_FullFlow(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	user := common.NewUser(common.ProviderLocal, "cliuser")
	user.Login = "cliuser"
	hash, err := common.HashPassword("clipassword")
	require.NoError(t, err)
	user.Password = hash

	err = start(ps)
	require.NoError(t, err, "unable to start Plik server")

	baseURL := ps.GetConfig().GetServerURL().String()
	err = ps.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")

	// Step 1: CLI initiates auth session
	initBody, _ := json.Marshal(handlers.CLIAuthInitRequest{Hostname: "test-host"})
	initResp, err := http.Post(baseURL+"/auth/cli/init", "application/json", bytes.NewBuffer(initBody))
	require.NoError(t, err, "cli init request failed")
	defer initResp.Body.Close()
	require.Equal(t, http.StatusOK, initResp.StatusCode)

	var initResult handlers.CLIAuthInitResponse
	err = json.NewDecoder(initResp.Body).Decode(&initResult)
	require.NoError(t, err)
	require.NotEmpty(t, initResult.Code, "missing code")
	require.NotEmpty(t, initResult.Secret, "missing secret")
	require.NotEmpty(t, initResult.VerifyURL, "missing verify URL")
	require.Contains(t, initResult.VerifyURL, initResult.Code)
	require.Contains(t, initResult.VerifyURL, "hostname=test-host")
	require.Equal(t, 300, initResult.ExpiresIn)

	// Step 2: CLI polls — should be pending
	pollBody, _ := json.Marshal(handlers.CLIAuthPollRequest{Code: initResult.Code, Secret: initResult.Secret})
	pollResp, err := http.Post(baseURL+"/auth/cli/poll", "application/json", bytes.NewBuffer(pollBody))
	require.NoError(t, err, "cli poll request failed")
	defer pollResp.Body.Close()
	require.Equal(t, http.StatusOK, pollResp.StatusCode)

	var pollResult handlers.CLIAuthPollResponse
	err = json.NewDecoder(pollResp.Body).Decode(&pollResult)
	require.NoError(t, err)
	require.Equal(t, "pending", pollResult.Status)
	require.Empty(t, pollResult.Token)

	// Step 3: User logs in via browser and approves
	browserClient, xsrfToken := loginAndGetXSRF(t, baseURL, "cliuser", "clipassword")

	approveBody, _ := json.Marshal(handlers.CLIAuthApproveRequest{Code: initResult.Code, Comment: "my-workstation"})
	approveReq, err := http.NewRequest("POST", baseURL+"/auth/cli/approve", bytes.NewBuffer(approveBody))
	require.NoError(t, err)
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.Header.Set("X-XSRFToken", xsrfToken)
	approveResp, err := browserClient.Do(approveReq)
	require.NoError(t, err, "cli approve request failed")
	defer approveResp.Body.Close()
	require.Equal(t, http.StatusOK, approveResp.StatusCode)

	// Step 4: CLI polls — should now be approved with token
	pollBody2, _ := json.Marshal(handlers.CLIAuthPollRequest{Code: initResult.Code, Secret: initResult.Secret})
	pollResp2, err := http.Post(baseURL+"/auth/cli/poll", "application/json", bytes.NewBuffer(pollBody2))
	require.NoError(t, err, "cli poll request failed")
	defer pollResp2.Body.Close()
	require.Equal(t, http.StatusOK, pollResp2.StatusCode)

	var pollResult2 handlers.CLIAuthPollResponse
	err = json.NewDecoder(pollResp2.Body).Decode(&pollResult2)
	require.NoError(t, err)
	require.Equal(t, "approved", pollResult2.Status)
	require.NotEmpty(t, pollResult2.Token)

	// Step 5: Verify token is valid and belongs to the correct user
	token, err := ps.GetMetadataBackend().GetToken(pollResult2.Token)
	require.NoError(t, err)
	require.NotNil(t, token)
	require.Equal(t, "my-workstation", token.Comment)
	require.Equal(t, user.ID, token.UserID)

	// Step 6: Session should be consumed (one-time use)
	pollBody3, _ := json.Marshal(handlers.CLIAuthPollRequest{Code: initResult.Code, Secret: initResult.Secret})
	pollResp3, err := http.Post(baseURL+"/auth/cli/poll", "application/json", bytes.NewBuffer(pollBody3))
	require.NoError(t, err, "cli poll request failed")
	defer pollResp3.Body.Close()
	require.Equal(t, http.StatusNotFound, pollResp3.StatusCode)
}

func TestCLIAuth_DefaultComment(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	user := common.NewUser(common.ProviderLocal, "cliuser2")
	user.Login = "cliuser2"
	hash, err := common.HashPassword("pass")
	require.NoError(t, err)
	user.Password = hash

	err = start(ps)
	require.NoError(t, err)

	baseURL := ps.GetConfig().GetServerURL().String()
	err = ps.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	// Init session without hostname
	initBody, _ := json.Marshal(handlers.CLIAuthInitRequest{})
	initResp, err := http.Post(baseURL+"/auth/cli/init", "application/json", bytes.NewBuffer(initBody))
	require.NoError(t, err)
	defer initResp.Body.Close()
	require.Equal(t, http.StatusOK, initResp.StatusCode)

	var initResult handlers.CLIAuthInitResponse
	err = json.NewDecoder(initResp.Body).Decode(&initResult)
	require.NoError(t, err)

	// Verify URL should NOT contain hostname param when not provided
	require.NotContains(t, initResult.VerifyURL, "hostname=")

	// Login and approve without custom comment
	browserClient, xsrfToken := loginAndGetXSRF(t, baseURL, "cliuser2", "pass")

	approveBody, _ := json.Marshal(handlers.CLIAuthApproveRequest{Code: initResult.Code})
	approveReq, _ := http.NewRequest("POST", baseURL+"/auth/cli/approve", bytes.NewBuffer(approveBody))
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.Header.Set("X-XSRFToken", xsrfToken)
	approveResp, err := browserClient.Do(approveReq)
	require.NoError(t, err)
	defer approveResp.Body.Close()
	require.Equal(t, http.StatusOK, approveResp.StatusCode)

	// Poll for token
	pollBody, _ := json.Marshal(handlers.CLIAuthPollRequest{Code: initResult.Code, Secret: initResult.Secret})
	pollResp, err := http.Post(baseURL+"/auth/cli/poll", "application/json", bytes.NewBuffer(pollBody))
	require.NoError(t, err)
	defer pollResp.Body.Close()

	var pollResult handlers.CLIAuthPollResponse
	err = json.NewDecoder(pollResp.Body).Decode(&pollResult)
	require.NoError(t, err)
	require.Equal(t, "approved", pollResult.Status)

	// Default comment should be "CLI login"
	token, err := ps.GetMetadataBackend().GetToken(pollResult.Token)
	require.NoError(t, err)
	require.Equal(t, "CLI login", token.Comment)
}

func TestCLIAuth_PollWrongSecret(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	err := start(ps)
	require.NoError(t, err)

	baseURL := ps.GetConfig().GetServerURL().String()

	// Init session
	initBody, _ := json.Marshal(handlers.CLIAuthInitRequest{Hostname: "host"})
	initResp, err := http.Post(baseURL+"/auth/cli/init", "application/json", bytes.NewBuffer(initBody))
	require.NoError(t, err)
	defer initResp.Body.Close()

	var initResult handlers.CLIAuthInitResponse
	err = json.NewDecoder(initResp.Body).Decode(&initResult)
	require.NoError(t, err)

	// Poll with wrong secret
	pollBody, _ := json.Marshal(handlers.CLIAuthPollRequest{Code: initResult.Code, Secret: "wrong-secret"})
	pollResp, err := http.Post(baseURL+"/auth/cli/poll", "application/json", bytes.NewBuffer(pollBody))
	require.NoError(t, err)
	defer pollResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, pollResp.StatusCode)
}

func TestCLIAuth_ApproveUnauthenticated(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	err := start(ps)
	require.NoError(t, err)

	baseURL := ps.GetConfig().GetServerURL().String()

	// Init session
	initBody, _ := json.Marshal(handlers.CLIAuthInitRequest{})
	initResp, err := http.Post(baseURL+"/auth/cli/init", "application/json", bytes.NewBuffer(initBody))
	require.NoError(t, err)
	defer initResp.Body.Close()

	var initResult handlers.CLIAuthInitResponse
	err = json.NewDecoder(initResp.Body).Decode(&initResult)
	require.NoError(t, err)

	// Try to approve without being logged in
	approveBody, _ := json.Marshal(handlers.CLIAuthApproveRequest{Code: initResult.Code})
	approveResp, err := http.Post(baseURL+"/auth/cli/approve", "application/json", bytes.NewBuffer(approveBody))
	require.NoError(t, err)
	defer approveResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, approveResp.StatusCode)
}

func TestCLIAuth_AuthDisabled(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureDisabled
	_ = ps.GetConfig().Initialize(nil)

	err := start(ps)
	require.NoError(t, err)

	baseURL := ps.GetConfig().GetServerURL().String()

	// CLI init should fail when auth is disabled
	initBody, _ := json.Marshal(handlers.CLIAuthInitRequest{})
	initResp, err := http.Post(baseURL+"/auth/cli/init", "application/json", bytes.NewBuffer(initBody))
	require.NoError(t, err)
	defer initResp.Body.Close()
	require.Equal(t, http.StatusBadRequest, initResp.StatusCode)

	body, _ := io.ReadAll(initResp.Body)
	require.Contains(t, string(body), "authentication is disabled")
}

func TestCLIAuth_ApproveInvalidCode(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	user := common.NewUser(common.ProviderLocal, "cliuser3")
	user.Login = "cliuser3"
	hash, err := common.HashPassword("pass")
	require.NoError(t, err)
	user.Password = hash

	err = start(ps)
	require.NoError(t, err)

	baseURL := ps.GetConfig().GetServerURL().String()
	err = ps.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)

	// Login
	browserClient, xsrfToken := loginAndGetXSRF(t, baseURL, "cliuser3", "pass")

	// Approve with invalid code
	approveBody, _ := json.Marshal(handlers.CLIAuthApproveRequest{Code: "XXXX-YYYY"})
	approveReq, _ := http.NewRequest("POST", baseURL+"/auth/cli/approve", bytes.NewBuffer(approveBody))
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.Header.Set("X-XSRFToken", xsrfToken)
	approveResp, err := browserClient.Do(approveReq)
	require.NoError(t, err)
	defer approveResp.Body.Close()
	require.Equal(t, http.StatusNotFound, approveResp.StatusCode)
}

// Force import of handlers so CLI auth routes are registered
var _ = handlers.CLIAuthInit
