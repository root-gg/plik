package common

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ErrorReader impement io.Reader and return err for every read call attempted
type ErrorReader struct {
	err error
}

// NewErrorReader return a new ErrorReader
func NewErrorReader(err error) (reader *ErrorReader) {
	reader = new(ErrorReader)
	reader.err = err
	return reader
}

// NewErrorReaderString return a new ErrorReader from the provided string
func NewErrorReaderString(err string) (reader *ErrorReader) {
	return NewErrorReader(errors.New(err))
}

// Read method to implement io.Reader
func (reader *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, reader.err
}

// TestTimeout execute a function and return an error if the defined timeout happen before
func TestTimeout(f func(), duration time.Duration) (err error) {
	c := make(chan struct{})
	go func() {
		f()
		close(c)
	}()
	select {
	case <-time.After(duration):
		return errors.New("timeout")
	case <-c:
		return nil
	}
}

// DummyHandler is a dummy http.Handler
var DummyHandler = http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {})

// APIMockServerDefaultPort is the default port to use for testing HTTP server
var APIMockServerDefaultPort = 44142

// StartAPIMockServer starts a new temporary API Server to be used in tests
func StartAPIMockServer(next http.Handler) (shutdown func(), err error) {
	return StartAPIMockServerCustomPort(APIMockServerDefaultPort, next)
}

// StartAPIMockServerCustomPort starts a new temporary API Server using a custom port
// Adds a middleware that handle the /not_found path called by the CheckHTTPServer function
func StartAPIMockServerCustomPort(port int, next http.Handler) (shutdown func(), err error) {
	shutdown = func() {}
	tcpListener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return shutdown, err
	}

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/not_found" {
			resp.WriteHeader(http.StatusNotFound)
			resp.Write([]byte("not found"))
			return
		}

		next.ServeHTTP(resp, req)
	})

	httpServer := &http.Server{Handler: handler}

	shutdown = func() {
		err = httpServer.Close()
		if err != nil {
			panic(err)
		}
	}

	go httpServer.Serve(tcpListener)

	err = CheckHTTPServer(port)
	if err != nil {
		return shutdown, err
	}

	return shutdown, nil
}

var httpClient *http.Client

func getHTTPClient() *http.Client {
	if httpClient == nil {
		httpClient = &http.Client{Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	}
	return httpClient
}

// CheckHTTPServer for HTTP Server to be UP and running
// HTTP Server should must respond 404 to URL/not_found to be considered ok
func CheckHTTPServer(port int) (err error) {
	URL := "http://127.0.0.1:" + strconv.Itoa(port) + "/not_found"

	errCh := make(chan error, 1)
	done := make(chan struct{})
	f := func() {
	LOOP:
		for {
			select {
			case <-done:
				break LOOP
			default:
				req, err := http.NewRequest("GET", URL, nil)
				if err != nil {
					errCh <- err
					break LOOP
				}

				resp, err := getHTTPClient().Do(req)
				if err != nil {
					time.Sleep(50 * time.Millisecond)
					continue LOOP
				}

				if resp.StatusCode != http.StatusNotFound {
					errCh <- errors.New("invalid response status code")
				}

				break LOOP
			}
		}
		close(errCh)
	}

	err = TestTimeout(f, time.Second)
	if err != nil {
		return fmt.Errorf("Server unreachable %s", err)
	}

	close(done)

	err = <-errCh
	if err != nil {
		return fmt.Errorf("Server unreachable %s", err)
	}

	return nil
}

// RequireError is a helper to test the error and it's message
func RequireError(t *testing.T, err error, message string) {
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), message, "invalid error")
}
