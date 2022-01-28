SHELL = bash

BUILD_INFO = $(shell server/gen_build_info.sh base64)
BUILD_FLAG = -ldflags="-X github.com/root-gg/plik/server/common.buildInfoString=$(BUILD_INFO) -w -s -extldflags=-static"
BUILD_TAGS = -tags osusergo,netgo,sqlite_omit_load_extension

GO_BUILD = go build $(BUILD_FLAG) $(BUILD_TAGS)

COVER_FILE = /tmp/plik.coverprofile
GO_TEST = GORACE="halt_on_error=1" go test $(BUILD_FLAG) $(BUILD_TAGS) -race -cover -coverprofile=$(COVER_FILE) -p 1

ifdef ENABLE_RACE_DETECTOR
	GO_BUILD := GORACE="halt_on_error=1" $(GO_BUILD) -race
endif

all: clean clean-frontend frontend clients server

###
# Build frontend ressources
###
frontend:
	@if [ ! -d webapp/node_modules ]; then cd webapp && npm install ; fi
	@if [ ! -d webapp/bower_components ]; then cd webapp && node_modules/bower/bin/bower install --allow-root ; fi
	@cd webapp && node_modules/grunt-cli/bin/grunt

###
# Build plik server for the current architecture
###
server:
	@server/gen_build_info.sh info
	@echo "Building Plik server"
	@cd server && $(GO_BUILD) -o plikd

###
# Build plik client for the current architecture
###
client:
	@server/gen_build_info.sh info
	@echo "Building Plik client"
	@cd client && $(GO_BUILD) -o plik ./

###
# Build clients for all architectures
###
clients:
	# Only build clients
	@TARGETS="skip" releaser/releaser.sh

###
# Display build info
###
build-info:
	@server/gen_build_info.sh info

###
# Display version
###
version:
	@server/gen_build_info.sh version

###
# Run linters
###
lint:
	@FAIL=0 ;echo -n " - go fmt :" ; OUT=`gofmt -l . 2>&1 | grep -v ^vendor` ; \
	if [[ -z "$$OUT" ]]; then echo " OK" ; else echo " FAIL"; echo "$$OUT"; FAIL=1 ; fi ;\
	echo -n " - go vet :" ; OUT=`go vet ./... 2>&1` ; \
	if [[ -z "$$OUT" ]]; then echo " OK" ; else echo " FAIL"; echo "$$OUT"; FAIL=1 ; fi ;\
	test $$FAIL -eq 0

###
# Run fmt
###
fmt:
	@gofmt -w -s $(shell find . -type f -name '*.go' -not -path "./vendor/*" )

###
# Run tests
###
test:
	@if curl -s 127.0.0.1:8080 > /dev/null ; then echo "Plik server probably already running" ; exit 1 ; fi
	@$(GO_TEST) ./... 2>&1 | grep -v "no test files"; test $${PIPESTATUS[0]} -eq 0
	@echo "cli client integration tests :" && cd client && ./test.sh

###
# Open last cover profile in web browser
###
cover:
	@if [[ ! -f $(COVER_FILE) ]]; then echo "Please run \"make test\" first to generate a cover profile" ; exit 1; fi
	@go tool cover -html=$(COVER_FILE)
	@echo "Check your web browser to see the cover profile"

###
# Run integration tests for all available backends
###
test-backends:
	@testing/test_backends.sh

###
# Build a docker image locally
###
docker:
	@docker buildx build --progress=plain --load -t rootgg/plik:dev .

###
# Create release archives
###
release:
	@releaser/release.sh

###
# Create release archives, build a multiarch Docker image and push to Docker Hub
###
release-and-push-to-docker-hub:
	@PUSH_TO_DOCKER_HUB=true releaser/release.sh

###
# Remove server build files
###
clean:
	@rm -rf server/plikd
	@rm -rf client/plik
	@rm -rf clients
	@rm -rf servers
	@rm -rf release
	@rm -rf releases

###
# Remove frontend build files
###
clean-frontend:
	@rm -rf webapp/bower_components
	@rm -rf webapp/dist

###
# Remove all build files and node modules
###
clean-all: clean clean-frontend
	@rm -rf webapp/node_modules

###
# Since the client/server/version directories are not generated
# by make, we must declare these targets as phony to avoid :
# "make: `client' is up to date" cases at compile time
###
.PHONY: client clients server release