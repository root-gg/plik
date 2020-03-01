package common

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestErrorReader(t *testing.T) {
	e := errors.New("io error")
	reader := NewErrorReader(e)
	_, err := ioutil.ReadAll(reader)
	RequireError(t, err, e.Error())
}

func TestTestTimeout(t *testing.T) {
	err := TestTimeout(func() {}, time.Second)
	require.NoError(t, err, "invalid error")

	err = TestTimeout(func() { time.Sleep(2 * time.Second) }, time.Second)
	RequireError(t, err, "timeout")
}

func TestAPIMockServer(t *testing.T) {
	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		_, _ = resp.Write([]byte("ok"))
	})
	cancel, err := StartAPIMockServer(handler)
	defer cancel()
	require.NoError(t, err, "unable to start server")

	req, err := http.NewRequest("GET", "http://127.0.0.1:"+strconv.Itoa(APIMockServerDefaultPort), nil)
	require.NoError(t, err, "unable to create HTTP request")

	resp, err := getHTTPClient().Do(req)
	require.NoError(t, err, "unable to execute HTTP request")
	require.Equal(t, http.StatusOK, resp.StatusCode, "invalid HTTP response status")

	defer resp.Body.Close()
	value, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err, "unable to read HTTP response")
	require.Equal(t, "ok", string(value), "invalid HTTP response content")
}

func TestAPIMockServerTwice(t *testing.T) {
	cancel, err := StartAPIMockServer(DummyHandler)
	defer cancel()
	require.NoError(t, err, "unable to start server")

	cancel2, err := StartAPIMockServer(DummyHandler)
	defer cancel2()
	require.Error(t, err, "able to start server twice")
}

func TestAPIMockServerNoServer(t *testing.T) {
	err := CheckHTTPServer(APIMockServerDefaultPort)
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "timeout", "invalid error")
}
