package plik

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/handlers"
)

// oidcAvailable checks if the OIDC provider configured in the given config is reachable
func oidcAvailable(config *common.Configuration) bool {
	if config == nil || config.OIDCProviderURL == "" {
		return false
	}
	resp, err := http.Get(config.OIDCProviderURL + "/.well-known/openid-configuration")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// newBrowserClient creates an http.Client with a cookie jar (simulates a browser)
func newBrowserClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar: jar,
		// Don't follow redirects automatically for most tests
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// insecureCookieJar wraps a cookiejar.Jar and strips the Secure flag from cookies.
// Keycloak hardcodes SameSite=None on auth session cookies which requires Secure=true
// per the cookie spec. Go's cookiejar silently drops Secure cookies for HTTP URLs.
// This wrapper strips Secure so cookies propagate correctly in the test HTTP environment.
type insecureCookieJar struct {
	jar *cookiejar.Jar
}

func newInsecureCookieJar() *insecureCookieJar {
	jar, _ := cookiejar.New(nil)
	return &insecureCookieJar{jar: jar}
}

func (j *insecureCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, c := range cookies {
		c.Secure = false
	}
	j.jar.SetCookies(u, cookies)
}

func (j *insecureCookieJar) Cookies(u *url.URL) []*http.Cookie {
	return j.jar.Cookies(u)
}

// getCookie returns the named cookie from the response, or nil
func getCookie(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// ---- Local Authentication Tests (run with make test) ----

func TestLocalLoginBrowser(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	// Create a local user with a hashed password
	user := common.NewUser(common.ProviderLocal, "testuser")
	user.Login = "testuser"
	hash, err := common.HashPassword("testpassword")
	require.NoError(t, err)
	user.Password = hash
	user.Email = "test@example.com"
	user.Name = "Test User"

	err = start(ps)
	require.NoError(t, err, "unable to start Plik server")

	baseURL := ps.GetConfig().GetServerURL().String()
	err = ps.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")

	client := newBrowserClient()

	// Step 1: POST /auth/local/login with JSON credentials
	loginBody := `{"login":"testuser","password":"testpassword"}`
	resp, err := client.Post(baseURL+"/auth/local/login", "application/json", strings.NewReader(loginBody))
	require.NoError(t, err, "login request failed")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	require.Equal(t, "ok", string(body))

	// Verify session cookies are set
	sessionCookie := getCookie(resp, common.SessionCookieName)
	require.NotNil(t, sessionCookie, "missing session cookie")
	xsrfCookie := getCookie(resp, common.XSRFCookieName)
	require.NotNil(t, xsrfCookie, "missing xsrf cookie")

	// Step 2: GET /me using session cookies
	// We need to set cookies on the jar since CheckRedirect prevents auto-follow
	serverURL, _ := url.Parse(baseURL)
	client.Jar.SetCookies(serverURL, []*http.Cookie{sessionCookie, xsrfCookie})

	meResp, err := client.Get(baseURL + "/me")
	require.NoError(t, err, "get /me failed")
	defer meResp.Body.Close()

	require.Equal(t, http.StatusOK, meResp.StatusCode, "expected 200 on /me")

	var meUser common.User
	err = json.NewDecoder(meResp.Body).Decode(&meUser)
	require.NoError(t, err, "unable to decode /me response")
	require.Equal(t, common.ProviderLocal, meUser.Provider)
	require.Equal(t, "testuser", meUser.Login)
	require.Equal(t, "test@example.com", meUser.Email)

	// Step 3: GET /auth/logout
	logoutResp, err := client.Get(baseURL + "/auth/logout")
	require.NoError(t, err, "logout request failed")
	defer logoutResp.Body.Close()

	// Step 4: Verify /me returns 401 after logout
	// Clear cookies to simulate cleared session
	jar, _ := cookiejar.New(nil)
	client.Jar = jar

	meResp2, err := client.Get(baseURL + "/me")
	require.NoError(t, err, "get /me after logout failed")
	defer meResp2.Body.Close()
	require.Equal(t, http.StatusUnauthorized, meResp2.StatusCode, "expected 401 after logout")
}

func TestLocalLoginBrowserInvalidPassword(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	// Create a local user
	user := common.NewUser(common.ProviderLocal, "testuser2")
	hash, err := common.HashPassword("correctpassword")
	require.NoError(t, err)
	user.Password = hash

	err = start(ps)
	require.NoError(t, err, "unable to start Plik server")

	baseURL := ps.GetConfig().GetServerURL().String()
	err = ps.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")

	client := newBrowserClient()

	// POST with wrong password
	loginBody := `{"login":"testuser2","password":"wrongpassword"}`
	resp, err := client.Post(baseURL+"/auth/local/login", "application/json", strings.NewReader(loginBody))
	require.NoError(t, err, "login request failed")
	defer resp.Body.Close()

	require.Equal(t, http.StatusForbidden, resp.StatusCode, "expected 403 for invalid password")

	// Verify no session cookies are set
	sessionCookie := getCookie(resp, common.SessionCookieName)
	require.Nil(t, sessionCookie, "session cookie should not be set on invalid login")
}

func TestLocalLoginBrowserDisabled(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	// Disable local login
	ps.GetConfig().FeatureLocalLogin = common.FeatureDisabled
	_ = ps.GetConfig().Initialize(nil)

	err := start(ps)
	require.NoError(t, err, "unable to start Plik server")

	baseURL := ps.GetConfig().GetServerURL().String()
	client := newBrowserClient()

	loginBody := `{"login":"testuser","password":"testpassword"}`
	resp, err := client.Post(baseURL+"/auth/local/login", "application/json", strings.NewReader(loginBody))
	require.NoError(t, err, "login request failed")
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected 400 when local login is disabled")
}

// ---- OIDC Authentication Tests (require Keycloak) ----

func TestOIDCLoginBrowser(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	if !oidcAvailable(ps.GetConfig()) {
		t.Skip("OIDC provider not available, skipping OIDC test")
	}

	err := start(ps)
	require.NoError(t, err, "unable to start Plik server")

	baseURL := ps.GetConfig().GetServerURL().String()
	// Create a shared cookie jar that strips Secure flag from Keycloak cookies
	// (Keycloak hardcodes SameSite=None;Secure even over HTTP)
	jar := newInsecureCookieJar()

	// Step 1: GET /auth/oidc/login to get the authorization URL
	// Must include Referer header (getRedirectURL requires it to build callback URL)
	noRedirectClient := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	loginReq, err := http.NewRequest("GET", baseURL+"/auth/oidc/login", nil)
	require.NoError(t, err)
	loginReq.Header.Set("Referer", baseURL+"/")
	loginResp, err := noRedirectClient.Do(loginReq)
	require.NoError(t, err, "oidc login request failed")
	defer loginResp.Body.Close()

	require.Equal(t, http.StatusOK, loginResp.StatusCode, "expected 200 from /auth/oidc/login")

	authURLBytes, err := io.ReadAll(loginResp.Body)
	require.NoError(t, err)
	authURL := string(authURLBytes)
	require.Contains(t, authURL, ps.GetConfig().OIDCProviderURL, "auth URL should point to OIDC provider")

	// Step 2: Follow the auth URL through Keycloak redirects to get the login page
	// Use a client that follows redirects to accumulate all Keycloak session cookies
	followRedirectClient := &http.Client{Jar: jar}
	keycloakResp, err := followRedirectClient.Get(authURL)
	require.NoError(t, err, "unable to reach Keycloak authorization URL")
	defer keycloakResp.Body.Close()

	require.Equal(t, http.StatusOK, keycloakResp.StatusCode, "expected 200 from Keycloak login page")

	// Step 3: Parse the login form to find the action URL
	loginPageBody, err := io.ReadAll(keycloakResp.Body)
	require.NoError(t, err)
	loginPage := string(loginPageBody)

	formActionURL := extractFormAction(loginPage)
	require.NotEmpty(t, formActionURL, "unable to find form action URL in Keycloak login page")

	// Step 4: POST credentials to Keycloak (with same cookie jar)
	formData := url.Values{
		"username": {"testuser"},
		"password": {"password"},
	}
	submitResp, err := noRedirectClient.PostForm(formActionURL, formData)
	require.NoError(t, err, "unable to submit credentials to Keycloak")
	defer submitResp.Body.Close()

	// Keycloak should redirect back to Plik's callback URL
	if submitResp.StatusCode != http.StatusFound && submitResp.StatusCode != http.StatusSeeOther {
		submitBody, _ := io.ReadAll(submitResp.Body)
		t.Fatalf("expected redirect from Keycloak, got %d, body snippet: %.500s", submitResp.StatusCode, string(submitBody))
	}

	callbackURL := submitResp.Header.Get("Location")
	require.NotEmpty(t, callbackURL, "missing Location header in Keycloak redirect")
	require.Contains(t, callbackURL, "/auth/oidc/callback", "callback URL should point to Plik's OIDC callback")

	// Step 5: Follow the callback URL to Plik
	callbackResp, err := noRedirectClient.Get(callbackURL)
	require.NoError(t, err, "unable to call Plik OIDC callback")
	defer callbackResp.Body.Close()

	// Plik should redirect to /#/login and set auth cookies
	require.True(t, callbackResp.StatusCode == http.StatusMovedPermanently || callbackResp.StatusCode == http.StatusFound,
		"expected redirect from Plik callback, got %d", callbackResp.StatusCode)

	sessionCookie := getCookie(callbackResp, common.SessionCookieName)
	require.NotNil(t, sessionCookie, "missing session cookie after OIDC callback")
	xsrfCookie := getCookie(callbackResp, common.XSRFCookieName)
	require.NotNil(t, xsrfCookie, "missing xsrf cookie after OIDC callback")

	// Step 6: Use the session cookies to GET /me
	serverURL, _ := url.Parse(baseURL)
	jar.SetCookies(serverURL, []*http.Cookie{sessionCookie, xsrfCookie})

	meResp, err := noRedirectClient.Get(baseURL + "/me")
	require.NoError(t, err, "get /me failed")
	defer meResp.Body.Close()

	require.Equal(t, http.StatusOK, meResp.StatusCode, "expected 200 on /me after OIDC login")

	var meUser common.User
	err = json.NewDecoder(meResp.Body).Decode(&meUser)
	require.NoError(t, err, "unable to decode /me response")
	require.Equal(t, common.ProviderOIDC, meUser.Provider, "expected OIDC provider")
	require.Equal(t, "testuser@example.com", meUser.Email, "expected email from Keycloak user")
}

func TestOIDCLoginRedirectURL(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize(nil)

	if !oidcAvailable(ps.GetConfig()) {
		t.Skip("OIDC provider not available, skipping OIDC test")
	}

	err := start(ps)
	require.NoError(t, err, "unable to start Plik server")

	baseURL := ps.GetConfig().GetServerURL().String()
	client := newBrowserClient()
	req, err := http.NewRequest("GET", baseURL+"/auth/oidc/login", nil)
	require.NoError(t, err)
	req.Header.Set("Referer", baseURL+"/")
	resp, err := client.Do(req)
	require.NoError(t, err, "oidc login request failed")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200, body: %s", string(body))

	authURL := string(body)

	// Verify the URL points to the configured Keycloak provider
	require.Contains(t, authURL, ps.GetConfig().OIDCProviderURL+"/protocol/openid-connect/auth")
	require.Contains(t, authURL, "client_id=plik")
	require.Contains(t, authURL, "redirect_uri=")
	// URL-decode the auth URL to check the redirect_uri contains the test server port
	decodedURL, err := url.QueryUnescape(authURL)
	require.NoError(t, err)
	require.Contains(t, decodedURL, fmt.Sprintf("127.0.0.1:%d", ps.GetConfig().ListenPort))
}

// extractFormAction parses HTML to find the action attribute of the login form
func extractFormAction(html string) string {
	// Look for the login form action URL
	// Keycloak login pages have a form with id="kc-form-login"
	// The action attribute contains the URL to POST credentials to
	actionIdx := strings.Index(html, "action=\"")
	if actionIdx == -1 {
		return ""
	}
	actionIdx += len("action=\"")
	endIdx := strings.Index(html[actionIdx:], "\"")
	if endIdx == -1 {
		return ""
	}
	actionURL := html[actionIdx : actionIdx+endIdx]
	// Keycloak HTML-encodes &amp; in the action URL
	actionURL = strings.ReplaceAll(actionURL, "&amp;", "&")
	return actionURL
}

// Force import of handlers package so that handler routes (including OIDC endpoints)
// are registered via init(). Without this, the test binary would not include the
// handler code and OIDC routes would return 404.
var _ = handlers.OIDCLogin
