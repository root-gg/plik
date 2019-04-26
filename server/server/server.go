/* The MIT License (MIT)

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
THE SOFTWARE. */

package server

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/root-gg/juliet"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/root-gg/plik/server/data/file"
	"github.com/root-gg/plik/server/data/stream"
	"github.com/root-gg/plik/server/data/swift"
	data_test "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/data/weedfs"
	"github.com/root-gg/plik/server/handlers"
	"github.com/root-gg/plik/server/metadata"
	"github.com/root-gg/plik/server/metadata/bolt"
	"github.com/root-gg/plik/server/metadata/mongo"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/root-gg/plik/server/middleware"
)

// PlikServer is a Plik Server instance
type PlikServer struct {
	config *common.Configuration
	logger *logger.Logger

	metadataBackend metadata.Backend
	dataBackend     data.Backend
	streamBackend   data.Backend

	httpServer *http.Server

	mu      sync.Mutex
	started bool
	done    bool

	cleaningRandomDelay int
	cleaningMinOffset   int
}

// NewPlikServer create a new Plik Server instance
func NewPlikServer(config *common.Configuration) (ps *PlikServer) {
	ps = new(PlikServer)
	ps.config = config

	ps.logger = logger.NewLogger().SetMinLevelFromString(config.LogLevel)
	if ps.logger.MinLevel == logger.DEBUG {
		ps.logger.SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel | logger.FshortFile | logger.FshortFunction)
	} else {
		ps.logger.SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel)
	}

	ps.cleaningRandomDelay = 3600
	ps.cleaningMinOffset = 7200

	return ps
}

// Start a Plik Server instance
func (ps *PlikServer) Start() (err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.done {
		return errors.New("can't start a shutdown Plik server")
	}

	if ps.started {
		return errors.New("can't start a Plik server twice")
	}

	// You get only one try
	ps.started = true

	return ps.start()
}

func (ps *PlikServer) start() (err error) {
	log := ps.logger

	// TODO what if the server has been shutdown before ???

	log.Infof("Starting plikd server v" + common.GetBuildInfo().Version)

	// Initialize backends
	err = ps.initializeMetadataBackend()
	if err != nil {
		return fmt.Errorf("Unable to initialize metadata backend : %s", err)
	}

	err = ps.initializeDataBackend()
	if err != nil {
		return fmt.Errorf("Unable to initialize data backend : %s", err)
	}

	err = ps.initializeStreamBackend()
	if err != nil {
		return fmt.Errorf("Unable to initialize stream backend : %s", err)
	}

	if ps.config.IsAutoClean() {
		go ps.uploadsCleaningRoutine()
	}

	handler := ps.getHTTPHandler()

	var proto string
	address := ps.config.ListenAddress + ":" + strconv.Itoa(ps.config.ListenPort)
	if ps.config.SslEnabled {
		proto = "https"

		// Load cert
		cert, err := tls.LoadX509KeyPair(ps.config.SslCert, ps.config.SslKey)
		if err != nil {
			return fmt.Errorf("Unable to load ssl certificate : %s", err)
		}

		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS10, Certificates: []tls.Certificate{cert}}
		ps.httpServer = &http.Server{Addr: address, Handler: handler, TLSConfig: tlsConfig}
	} else {
		proto = "http"
		ps.httpServer = &http.Server{Addr: address, Handler: handler}
	}

	log.Infof("Starting server at %s://%s", proto, address)

	// Start HTTP Server
	go func() {
		err := ps.httpServer.ListenAndServe()
		if err != nil {
			ps.mu.Lock()
			defer ps.mu.Unlock()
			if !ps.done {
				log.Fatalf("Unable to start HTTP server : %s", err)
			}
		}
	}()

	err = common.CheckHTTPServer(ps.GetConfig().ListenPort)
	if err != nil {
		return err
	}

	return nil
}

// Shutdown gracefully shutdown a Plik Server instance with a timeout grace period for connexions to close
func (ps *PlikServer) Shutdown(timeout time.Duration) (err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if !ps.started {
		return nil
	}

	if ps.done {
		return errors.New("can't shutdown a Plik server twice")
	}

	return ps.shutdown(timeout)
}

// ShutdownNow a Plik Server instance abruptly closing all connection immediately
func (ps *PlikServer) ShutdownNow() (err error) {
	return ps.Shutdown(0)
}

func (ps *PlikServer) shutdown(timeout time.Duration) (err error) {
	ps.logger.Info("Shutdown server at " + ps.GetConfig().GetServerURL().String())
	ps.done = true

	if ps.httpServer == nil {
		return
	}

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err = ps.httpServer.Shutdown(ctx)
		if err == nil {
			return
		}
	}

	return ps.httpServer.Close()
}

func (ps *PlikServer) getHTTPHandler() (handler http.Handler) {
	// Initialize middleware chain
	stdChain := juliet.NewChainWithContextBuilder(ps.NewContext, middleware.SourceIP, middleware.Log)

	// Get user from session cookie
	authChain := stdChain.Append(middleware.Authenticate(false), middleware.Impersonate)

	// Get user from session cookie or X-PlikToken header
	tokenChain := stdChain.Append(middleware.Authenticate(true), middleware.Impersonate)

	// Redirect on error for webapp
	stdChainWithRedirect := juliet.NewChain(middleware.RedirectOnFailure).AppendChain(stdChain)
	authChainWithRedirect := juliet.NewChain(middleware.RedirectOnFailure).AppendChain(tokenChain)

	getFileChain := juliet.NewChain(middleware.Upload, middleware.Yubikey, middleware.File)

	// HTTP Api routes configuration
	router := mux.NewRouter()
	router.Handle("/config", stdChain.Then(handlers.GetConfiguration)).Methods("GET")
	router.Handle("/version", stdChain.Then(handlers.GetVersion)).Methods("GET")
	router.Handle("/upload", tokenChain.Then(handlers.CreateUpload)).Methods("POST")
	router.Handle("/upload/{uploadID}", authChain.Append(middleware.Upload).Then(handlers.GetUpload)).Methods("GET")
	router.Handle("/upload/{uploadID}", tokenChain.Append(middleware.Upload).Then(handlers.RemoveUpload)).Methods("DELETE")
	router.Handle("/file/{uploadID}", tokenChain.Append(middleware.Upload).Then(handlers.AddFile)).Methods("POST")
	router.Handle("/file/{uploadID}/{fileID}/{filename}", tokenChain.Append(middleware.Upload, middleware.File).Then(handlers.AddFile)).Methods("POST")
	router.Handle("/file/{uploadID}/{fileID}/{filename}", tokenChain.Append(middleware.Upload, middleware.File).Then(handlers.RemoveFile)).Methods("DELETE")
	router.Handle("/file/{uploadID}/{fileID}/{filename}", authChainWithRedirect.AppendChain(getFileChain).Then(handlers.GetFile)).Methods("HEAD", "GET")
	router.Handle("/file/{uploadID}/{fileID}/{filename}/yubikey/{yubikey}", authChainWithRedirect.AppendChain(getFileChain).Then(handlers.GetFile)).Methods("HEAD", "GET")
	router.Handle("/stream/{uploadID}/{fileID}/{filename}", tokenChain.Append(middleware.Upload, middleware.File).Then(handlers.AddFile)).Methods("POST")
	router.Handle("/stream/{uploadID}/{fileID}/{filename}", authChainWithRedirect.AppendChain(getFileChain).Then(handlers.GetFile)).Methods("HEAD", "GET")
	router.Handle("/stream/{uploadID}/{fileID}/{filename}/yubikey/{yubikey}", authChainWithRedirect.AppendChain(getFileChain).Then(handlers.GetFile)).Methods("HEAD", "GET")
	router.Handle("/archive/{uploadID}/{filename}", authChainWithRedirect.Append(middleware.Upload, middleware.Yubikey).Then(handlers.GetArchive)).Methods("HEAD", "GET")
	router.Handle("/archive/{uploadID}/{filename}/yubikey/{yubikey}", authChainWithRedirect.Append(middleware.Upload, middleware.Yubikey).Then(handlers.GetArchive)).Methods("HEAD", "GET")
	router.Handle("/auth/google/login", authChain.Then(handlers.GoogleLogin)).Methods("GET")
	router.Handle("/auth/google/callback", stdChainWithRedirect.Then(handlers.GoogleCallback)).Methods("GET")
	router.Handle("/auth/ovh/login", authChain.Then(handlers.OvhLogin)).Methods("GET")
	router.Handle("/auth/ovh/callback", stdChainWithRedirect.Then(handlers.OvhCallback)).Methods("GET")
	router.Handle("/auth/logout", authChain.Then(handlers.Logout)).Methods("GET")
	router.Handle("/me", authChain.Then(handlers.UserInfo)).Methods("GET")
	router.Handle("/me", authChain.Then(handlers.DeleteAccount)).Methods("DELETE")
	router.Handle("/me/token", authChain.Then(handlers.CreateToken)).Methods("POST")
	router.Handle("/me/token/{token}", authChain.Then(handlers.RevokeToken)).Methods("DELETE")
	router.Handle("/me/uploads", authChain.Then(handlers.GetUserUploads)).Methods("GET")
	router.Handle("/me/uploads", authChain.Then(handlers.RemoveUserUploads)).Methods("DELETE")
	router.Handle("/me/stats", authChain.Then(handlers.GetUserStatistics)).Methods("GET")
	router.Handle("/stats", authChain.Then(handlers.GetServerStatistics)).Methods("GET")
	router.Handle("/users", authChain.Then(handlers.GetUsers)).Methods("GET")
	router.Handle("/qrcode", stdChain.Then(handlers.GetQrCode)).Methods("GET")
	router.PathPrefix("/clients/").Handler(http.StripPrefix("/clients/", http.FileServer(http.Dir("../clients"))))
	router.PathPrefix("/changelog/").Handler(http.StripPrefix("/changelog/", http.FileServer(http.Dir("../changelog"))))
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

	handler = common.StripPrefix(ps.config.Path, router)
	return handler
}

// WithMetadataBackend configure the metadata backend to use ( call before Start() )
func (ps *PlikServer) WithMetadataBackend(backend metadata.Backend) *PlikServer {
	if ps.metadataBackend == nil {
		ps.metadataBackend = backend
	}
	return ps
}

// Initialize metadata backend from type found in configuration
func (ps *PlikServer) initializeMetadataBackend() (err error) {
	if ps.metadataBackend == nil {
		switch ps.config.MetadataBackend {
		case "mongo":
			config := mongo.NewConfig(ps.config.MetadataBackendConfig)
			ps.metadataBackend, err = mongo.NewBackend(config)
			if err != nil {
				return err
			}
		case "bolt":
			config := bolt.NewConfig(ps.config.MetadataBackendConfig)
			ps.metadataBackend, err = bolt.NewBackend(config)
			if err != nil {
				return err
			}
		case "testing":
			ps.metadataBackend = metadata_test.NewBackend()
		default:
			return fmt.Errorf("Invalid metadata backend %s", ps.config.MetadataBackend)
		}
	}

	return nil
}

// WithDataBackend configure the data backend to use ( call before Start() )
func (ps *PlikServer) WithDataBackend(backend data.Backend) *PlikServer {
	if ps.dataBackend == nil {
		ps.dataBackend = backend
	}
	return ps
}

// Initialize data backend from type found in configuration
func (ps *PlikServer) initializeDataBackend() (err error) {
	if ps.dataBackend == nil {
		switch ps.config.DataBackend {
		case "file":
			config := file.NewConfig(ps.config.DataBackendConfig)
			ps.dataBackend = file.NewBackend(config)
		case "swift":
			config := swift.NewConfig(ps.config.DataBackendConfig)
			ps.dataBackend = swift.NewBackend(config)
		case "weedfs":
			config := weedfs.NewConfig(ps.config.DataBackendConfig)
			ps.dataBackend = weedfs.NewBackend(config)
		case "testing":
			ps.dataBackend = data_test.NewBackend()
		default:
			return fmt.Errorf("Invalid data backend %s", ps.config.DataBackend)
		}
	}

	return nil
}

// WithStreamBackend configure the stream backend to use ( call before Start() )
func (ps *PlikServer) WithStreamBackend(backend data.Backend) *PlikServer {
	if ps.streamBackend == nil {
		ps.streamBackend = backend
	}
	return ps
}

// Initialize data backend from type found in configuration
func (ps *PlikServer) initializeStreamBackend() (err error) {
	if ps.streamBackend == nil && ps.config.StreamMode {
		config := stream.NewConfig(ps.config.StreamBackendConfig)
		ps.streamBackend = stream.NewBackend(config)
	}

	return nil
}

// UploadsCleaningRoutine periodicaly remove expired uploads
func (ps *PlikServer) uploadsCleaningRoutine() {
	log := ps.logger
	for {
		if ps.done {
			break
		}
		// Sleep between 2 hours and 3 hours
		// This is a dirty trick to avoid frontends doing this at the same time
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(ps.cleaningRandomDelay)))
		randomSleep := r.Int64() + int64(ps.cleaningMinOffset)

		log.Infof("Will clean old uploads in %d seconds.", randomSleep)
		time.Sleep(time.Duration(randomSleep) * time.Second)
		log.Infof("Cleaning expired uploads...")
		ps.Clean()
	}
}

// Clean removes expired uploads from the servers
func (ps *PlikServer) Clean() {
	log := ps.logger
	ctx := ps.NewContext()

	// Get uploads that needs to be removed
	uploadIds, err := ps.metadataBackend.GetUploadsToRemove(ctx)
	if err != nil {
		log.Warningf("Failed to get expired uploads : %s", err)
	} else {
		// Remove them
		for _, uploadID := range uploadIds {
			log.Infof("Removing expired upload %s", uploadID)
			// Get upload metadata
			upload, err := ps.metadataBackend.Get(ctx, uploadID)
			if err != nil {
				log.Warningf("Unable to get infos for upload: %s", err)
				continue
			}

			// Remove from data backend
			err = ps.dataBackend.RemoveUpload(ctx, upload)
			if err != nil {
				log.Warningf("Unable to remove upload data : %s", err)
				continue
			}

			// Remove from metadata backend
			err = ps.metadataBackend.Remove(ctx, upload)
			if err != nil {
				log.Warningf("Unable to remove upload metadata : %s", err)
			}
		}
	}
}

// GetConfig return the server configuration
func (ps *PlikServer) GetConfig() *common.Configuration {
	return ps.config
}

// GetMetadataBackend return the configured Backend
func (ps *PlikServer) GetMetadataBackend() metadata.Backend {
	return ps.metadataBackend
}

// GetDataBackend return the configured DataBackend
func (ps *PlikServer) GetDataBackend() data.Backend {
	return ps.dataBackend
}

// GetStreamBackend return the configured StreamBackend
func (ps *PlikServer) GetStreamBackend() data.Backend {
	return ps.streamBackend
}

// NewContext return a new scoped context to pass along
func (ps *PlikServer) NewContext() *juliet.Context {
	ctx := juliet.NewContext()
	ctx.Set("config", ps.config)
	ctx.Set("logger", ps.logger.Copy())
	ctx.Set("metadata_backend", ps.metadataBackend)
	ctx.Set("data_backend", ps.dataBackend)
	ctx.Set("stream_backend", ps.streamBackend)
	return ctx
}
