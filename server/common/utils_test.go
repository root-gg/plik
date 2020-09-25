package common

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStripPrefixNoPrefix(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, "/prefix", req.URL.Path, "invalid request url")
}

func TestStripPrefixNoExactPrefix(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, 301, rr.Code, "invalid handler response status code")
	require.Equal(t, "/prefix/", rr.Result().Header.Get("Location"), "invalid location header")
}

func TestStripPrefix(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix/path", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, "/path", req.URL.Path, "invalid location header")
}

func TestStripPrefixNotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/invalid/path", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code, "invalid handler response status code")
}

func TestStripPrefixRootSlash(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix/path", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix/", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, "/path", req.URL.Path, "invalid location header")
}

func TestNewHTTPError(t *testing.T) {
	e := NewHTTPError("msg", fmt.Errorf("error"), http.StatusInternalServerError)
	require.Equal(t, "msg : error", e.Error())
}

func TestEncodeAuthBasicHeader(t *testing.T) {
	b64 := EncodeAuthBasicHeader("login", "password")
	out := make([]byte, 14)
	_, err := base64.StdEncoding.Decode(out, []byte(b64))
	require.NoError(t, err)
	require.Equal(t, "login:password", string(out))
}

func TestWriteJSONResponse(t *testing.T) {
	obj := &struct{ Foo string }{"Bar"}

	rr := httptest.NewRecorder()
	WriteJSONResponse(rr, obj)

	body, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err)
	require.NotNil(t, body)

	obj2 := &struct{ Foo string }{}
	err = json.Unmarshal(body, obj2)
	require.NoError(t, err)

	require.Equal(t, obj.Foo, obj2.Foo)
}

func TestHumanDuration(t *testing.T) {
	require.Equal(t, "0s", HumanDuration(time.Duration(0)))
	require.Equal(t, "10ms", HumanDuration(10*time.Millisecond))
	require.Equal(t, "1s10ms", HumanDuration(time.Second+10*time.Millisecond))
	require.Equal(t, "30s", HumanDuration(30*time.Second))
	require.Equal(t, "30m", HumanDuration(30*time.Minute))
	require.Equal(t, "30m3s", HumanDuration(30*time.Minute+3*time.Second))
	require.Equal(t, "1h", HumanDuration(time.Hour))
	require.Equal(t, "1h1s", HumanDuration(time.Hour+time.Second))
	require.Equal(t, "1h1m", HumanDuration(time.Hour+time.Minute))
	require.Equal(t, "1h1m1s", HumanDuration(time.Hour+time.Minute+time.Second))
	require.Equal(t, "1d", HumanDuration(24*time.Hour))
	require.Equal(t, "1d1m1s", HumanDuration(24*time.Hour+time.Minute+time.Second))
	require.Equal(t, "1d1h1m1s", HumanDuration(24*time.Hour+time.Hour+time.Minute+time.Second))
	require.Equal(t, "30d", HumanDuration(30*24*time.Hour))
	require.Equal(t, "1y", HumanDuration(365*24*time.Hour))
	require.Equal(t, "1y1d", HumanDuration(366*24*time.Hour))
	require.Equal(t, "1y1d1s", HumanDuration(366*24*time.Hour+time.Second))
	require.Equal(t, "1y1d1h1m1s", HumanDuration(366*24*time.Hour+3661*time.Second))
	require.Equal(t, "-10s", HumanDuration(-10*time.Second))
}
