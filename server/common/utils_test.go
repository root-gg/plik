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
