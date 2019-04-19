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
	"github.com/root-gg/plik/server/data/weedfs"
	"github.com/root-gg/plik/server/handlers"
	"github.com/root-gg/plik/server/metadata"
	"github.com/root-gg/plik/server/metadata/bolt"
	"github.com/root-gg/plik/server/metadata/mongo"
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

	// TODO find a better solution maybe ?
	startOnce    sync.Once
	shutdownOnce sync.Once

	done chan struct{}
}

// NewPlikServer create a new Plik Server instance
func NewPlikServer(config *common.Configuration) (ps *PlikServer) {
	ps = new(PlikServer)
	ps.config = config

	ps.logger = logger.NewLogger().SetMinLevel(logger.INFO).SetMinLevelFromString(config.LogLevel)
	if config.LogLevel == "DEBUG" {
		ps.logger.SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel | logger.FshortFile | logger.FshortFunction)
	} else {
		ps.logger.SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel)
	}

	ps.done = make(chan struct{})
	return ps
}

// Start a Plik Server instance
func (ps *PlikServer) Start() (err error) {
	ps.startOnce.Do(func() {
		err = ps.start()
	})
	return
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

	go ps.uploadsCleaningRoutine()

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

	log.Infof("Starting http server at %s://%s", proto, address)

	go func() {
		// Todo find error handling ?
		err = ps.httpServer.ListenAndServe()
		if err != nil {
			log.Fatalf("Unable to start HTTP server : %s", err)
		}
	}()

	return nil
}

// Shutdown a Plik Server instance
func (ps *PlikServer) Shutdown() (err error) {
	ps.shutdownOnce.Do(func() {
		err = ps.shutdown()
	})
	return
}

func (ps *PlikServer) shutdown() (err error) {
	close(ps.done)
	if ps.httpServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	err = ps.httpServer.Shutdown(ctx)
	if err != nil {
		err = ps.httpServer.Close()
	}

	return err
}

func (ps *PlikServer) getHTTPHandler() (handler http.Handler) {
	// Initialize middleware chain
	stdChain := juliet.NewChainWithContextBuilder(ps.newContext, middleware.SourceIP, middleware.Log)

	// Get user from session cookie
	authChain := stdChain.Append(middleware.Authenticate(false), middleware.Impersonate)

	// Get user from session cookie or X-PlikToken header
	tokenChain := stdChain.Append(middleware.Authenticate(true), middleware.Impersonate)

	// Redirect on error for webapp
	stdChainWithRedirect := juliet.NewChain(middleware.RedirectOnFailure).AppendChain(stdChain)
	authChainWithRedirect := juliet.NewChain(middleware.RedirectOnFailure).AppendChain(authChain)

	getFileChain := juliet.NewChain(middleware.Upload, middleware.Yubikey, middleware.File)

	// HTTP Api routes configuration
	router := mux.NewRouter()
	router.Handle("/config", stdChain.Then(handlers.GetConfiguration)).Methods("GET")
	router.Handle("/version", stdChain.Then(handlers.GetVersion)).Methods("GET")
	router.Handle("/upload", tokenChain.Then(handlers.CreateUpload)).Methods("POST")
	router.Handle("/upload/{uploadID}", authChain.Append(middleware.Upload).Then(handlers.GetUpload)).Methods("GET")
	router.Handle("/upload/{uploadID}", authChain.Append(middleware.Upload).Then(handlers.RemoveUpload)).Methods("DELETE")
	router.Handle("/file/{uploadID}", tokenChain.Append(middleware.Upload).Then(handlers.AddFile)).Methods("POST")
	router.Handle("/file/{uploadID}/{fileID}/{filename}", tokenChain.Append(middleware.Upload, middleware.File).Then(handlers.AddFile)).Methods("POST")
	router.Handle("/file/{uploadID}/{fileID}/{filename}", authChain.Append(middleware.Upload, middleware.File).Then(handlers.RemoveFile)).Methods("DELETE")
	router.Handle("/file/{uploadID}/{fileID}/{filename}", authChainWithRedirect.AppendChain(getFileChain).Then(handlers.GetFile)).Methods("HEAD", "GET")
	router.Handle("/file/{uploadID}/{fileID}/{filename}/yubikey/{yubikey}", authChainWithRedirect.AppendChain(getFileChain).Then(handlers.GetFile)).Methods("HEAD", "GET")
	router.Handle("/stream/{uploadID}/{fileID}/{filename}", tokenChain.Append(middleware.Upload, middleware.File).Then(handlers.AddFile)).Methods("POST")
	router.Handle("/stream/{uploadID}/{fileID}/{filename}", authChain.Append(middleware.Upload, middleware.File).Then(handlers.RemoveFile)).Methods("DELETE")
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
			config := mongo.NewMongoMetadataBackendConfig(ps.config.MetadataBackendConfig)
			ps.metadataBackend, err = mongo.NewMongoMetadataBackend(config)
			if err != nil {
				return err
			}
		case "bolt":
			config := bolt.NewBoltMetadataBackendConfig(ps.config.MetadataBackendConfig)
			ps.metadataBackend, err = bolt.NewBoltMetadataBackend(config)
			if err != nil {
				return err
			}
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
			config := file.NewFileBackendConfig(ps.config.DataBackendConfig)
			ps.dataBackend = file.NewFileBackend(config)
		case "swift":
			config := swift.NewSwitftBackendConfig(ps.config.DataBackendConfig)
			ps.dataBackend = swift.NewSwiftBackend(config)
		case "weedfs":
			config := weedfs.NewWeedFsBackendConfig(ps.config.DataBackendConfig)
			ps.dataBackend = weedfs.NewWeedFsBackend(config)
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
		config := stream.NewStreamBackendConfig(ps.config.StreamBackendConfig)
		ps.streamBackend = stream.NewStreamBackend(config)
	}

	return nil
}

// UploadsCleaningRoutine periodicaly remove expired uploads
func (ps *PlikServer) uploadsCleaningRoutine() {
	log := ps.logger
	ctx := ps.newContext()
	for {
		select {
		case <-ps.done:
			return
		default:
			// Sleep between 2 hours and 3 hours
			// This is a dirty trick to avoid frontends doing this at the same time
			r, _ := rand.Int(rand.Reader, big.NewInt(3600))
			randomSleep := r.Int64() + 7200

			log.Infof("Will clean old uploads in %d seconds.", randomSleep)
			time.Sleep(time.Duration(randomSleep) * time.Second)
			log.Infof("Cleaning expired uploads...")

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
	}
}

func (ps *PlikServer) newContext() *juliet.Context {
	ctx := juliet.NewContext()
	ctx.Set("config", ps.config)
	ctx.Set("logger", ps.logger.Copy())
	ctx.Set("metadata_backend", ps.metadataBackend)
	ctx.Set("data_backend", ps.dataBackend)
	ctx.Set("stream_backend", ps.streamBackend)
	return ctx
}
