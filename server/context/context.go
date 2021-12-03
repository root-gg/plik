package context

import (
	"net"
	"net/http"
	"sync"

	"github.com/root-gg/logger"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/root-gg/plik/server/metadata"
)

// Context to be propagated throughout the middleware chain
type Context struct {
	config              *common.Configuration
	logger              *logger.Logger
	metadataBackend     *metadata.Backend
	dataBackend         data.Backend
	streamBackend       data.Backend
	authenticator       *common.SessionAuthenticator
	pagingQuery         *common.PagingQuery
	sourceIP            net.IP
	upload              *common.Upload
	file                *common.File
	user                *common.User
	token               *common.Token
	isWhitelisted       *bool
	isRedirectOnFailure bool
	isQuick             bool
	req                 *http.Request
	resp                http.ResponseWriter
	mu                  sync.RWMutex
}

// GetConfig get config from the context.
func (ctx *Context) GetConfig() *common.Configuration {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	if ctx.config == nil {
		panic("missing config from context")
	}

	return ctx.config
}

// SetConfig set config in the context
func (ctx *Context) SetConfig(config *common.Configuration) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.config = config
}

// GetLogger get logger from the context.
func (ctx *Context) GetLogger() *logger.Logger {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	if ctx.logger == nil {
		panic("missing logger from context")
	}

	return ctx.logger
}

// SetLogger set logger in the context
func (ctx *Context) SetLogger(logger *logger.Logger) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.logger = logger
}

// GetMetadataBackend get metadataBackend from the context.
func (ctx *Context) GetMetadataBackend() *metadata.Backend {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	if ctx.metadataBackend == nil {
		panic("missing metadataBackend from context")
	}

	return ctx.metadataBackend
}

// SetMetadataBackend set metadataBackend in the context
func (ctx *Context) SetMetadataBackend(metadataBackend *metadata.Backend) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.metadataBackend = metadataBackend
}

// GetDataBackend get dataBackend from the context.
func (ctx *Context) GetDataBackend() data.Backend {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	if ctx.dataBackend == nil {
		panic("missing dataBackend from context")
	}

	return ctx.dataBackend
}

// SetDataBackend set dataBackend in the context
func (ctx *Context) SetDataBackend(dataBackend data.Backend) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.dataBackend = dataBackend
}

// GetStreamBackend get streamBackend from the context.
func (ctx *Context) GetStreamBackend() data.Backend {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	if ctx.streamBackend == nil {
		panic("missing streamBackend from context")
	}

	return ctx.streamBackend
}

// SetStreamBackend set streamBackend in the context
func (ctx *Context) SetStreamBackend(streamBackend data.Backend) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.streamBackend = streamBackend
}

// GetAuthenticator get authenticator from the context.
func (ctx *Context) GetAuthenticator() *common.SessionAuthenticator {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	if ctx.authenticator == nil {
		panic("missing authenticator from context")
	}

	return ctx.authenticator
}

// SetAuthenticator set authenticator in the context
func (ctx *Context) SetAuthenticator(authenticator *common.SessionAuthenticator) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.authenticator = authenticator
}

// GetPagingQuery get pagingQuery from the context.
func (ctx *Context) GetPagingQuery() *common.PagingQuery {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	if ctx.pagingQuery == nil {
		panic("missing pagingQuery from context")
	}

	return ctx.pagingQuery
}

// SetPagingQuery set pagingQuery in the context
func (ctx *Context) SetPagingQuery(pagingQuery *common.PagingQuery) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.pagingQuery = pagingQuery
}

// GetSourceIP get sourceIP from the context.
func (ctx *Context) GetSourceIP() net.IP {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.sourceIP
}

// SetSourceIP set sourceIP in the context
func (ctx *Context) SetSourceIP(sourceIP net.IP) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.sourceIP = sourceIP
}

// GetUpload get upload from the context.
func (ctx *Context) GetUpload() *common.Upload {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.upload
}

// SetUpload set upload in the context
func (ctx *Context) SetUpload(upload *common.Upload) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.upload = upload
}

// GetFile get file from the context.
func (ctx *Context) GetFile() *common.File {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.file
}

// SetFile set file in the context
func (ctx *Context) SetFile(file *common.File) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.file = file
}

// GetUser get user from the context.
func (ctx *Context) GetUser() *common.User {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.user
}

// SetUser set user in the context
func (ctx *Context) SetUser(user *common.User) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.user = user
}

// GetToken get token from the context.
func (ctx *Context) GetToken() *common.Token {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.token
}

// SetToken set token in the context
func (ctx *Context) SetToken(token *common.Token) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.token = token
}

// IsRedirectOnFailure get isRedirectOnFailure from the context.
func (ctx *Context) IsRedirectOnFailure() bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.isRedirectOnFailure
}

// SetRedirectOnFailure set isRedirectOnFailure in the context
func (ctx *Context) SetRedirectOnFailure(isRedirectOnFailure bool) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.isRedirectOnFailure = isRedirectOnFailure
}

// IsQuick get isQuick from the context.
func (ctx *Context) IsQuick() bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.isQuick
}

// SetQuick set isQuick in the context
func (ctx *Context) SetQuick(isQuick bool) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.isQuick = isQuick
}

// GetReq get req from the context.
func (ctx *Context) GetReq() *http.Request {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.req
}

// SetReq set req in the context
func (ctx *Context) SetReq(req *http.Request) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.req = req
}

// GetResp get resp from the context.
func (ctx *Context) GetResp() http.ResponseWriter {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.resp
}

// SetResp set resp in the context
func (ctx *Context) SetResp(resp http.ResponseWriter) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.resp = resp
}
