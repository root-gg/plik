SHELL = bash

BUILD_INFO = $(shell server/gen_build_info.sh base64)
VERSION = $(shell server/gen_build_info.sh version)

# External ldflags: default to -static for static builds on linux.
# macOS (Darwin) does not support full static linking; avoid -static there.
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	EXTLDFLAGS :=
else
	EXTLDFLAGS := -static
endif

ifeq ($(strip $(EXTLDFLAGS)),)
	BUILD_FLAG = -ldflags="-X github.com/root-gg/plik/server/common.buildInfoString=$(BUILD_INFO) -w -s"
else
	BUILD_FLAG = -ldflags="-X github.com/root-gg/plik/server/common.buildInfoString=$(BUILD_INFO) -w -s -extldflags=$(EXTLDFLAGS)"
endif

BUILD_TAGS = -tags osusergo,netgo,sqlite_omit_load_extension

GO_BUILD = go build $(BUILD_FLAG) $(BUILD_TAGS)

COVER_FILE = /tmp/plik.coverprofile
RACE_FLAGS := $(if $(NO_RACE),,-race)
RACE_ENV   := $(if $(NO_RACE),,GORACE="halt_on_error=1")
GO_TEST     = $(RACE_ENV) go test $(BUILD_FLAG) $(BUILD_TAGS) $(RACE_FLAGS) -cover -coverprofile=$(COVER_FILE) -p 1

ifdef ENABLE_RACE_DETECTOR
	GO_BUILD := GORACE="halt_on_error=1" $(GO_BUILD) -race
endif

all: clean clean-frontend frontend clients server

###
# Build frontend ressources
###
frontend:
	@cd webapp && npm ci && npm run build

###
# Build plik server for the current architecture
###
server:
	@server/gen_build_info.sh info
	@echo "Building Plik server → ./server/plikd"
	@cd server && $(GO_BUILD) -o plikd

###
# Build plik client for the current architecture
###
client:
	@server/gen_build_info.sh info
	@echo "Building Plik client → ./client/plik"
	@cd client && $(GO_BUILD) -o plik ./

###
# Build clients for all architectures
###
clients:
	@releaser/build_clients.sh

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
	@FAIL=0 ;echo -n " - go fmt :" ; OUT=`gofmt -l client server plik` ; \
	if [[ -z "$$OUT" ]]; then echo " OK" ; else echo " FAIL"; echo "$$OUT"; FAIL=1 ; fi ;\
	echo -n " - go vet :" ; OUT=`go vet ./... 2>&1` ; RC=$$? ; \
	if [[ $$RC -eq 0 ]]; then echo " OK" ; else echo " FAIL"; echo "$$OUT"; FAIL=1 ; fi ;\
	echo -n " - go fix :" ; BEFORE=`git diff --name-only` ; go fix ./... 2>&1 ; AFTER=`git diff --name-only` ; \
	if [[ "$$BEFORE" == "$$AFTER" ]]; then echo " OK" ; else echo " FAIL"; diff <(echo "$$BEFORE") <(echo "$$AFTER") | grep '^>' | sed 's/^> /  /'; FAIL=1 ; fi ;\
	test $$FAIL -eq 0

###
# Run vulnerability check (requires: go install golang.org/x/vuln/cmd/govulncheck@latest)
###
vuln:
	@echo "Running govulncheck..."
	@govulncheck ./... || true
	@echo ""
	@echo "Running npm audit..."
	@cd webapp && npm audit || true

###
# Run fmt
###
fmt:
	@gofmt -w -s client server plik

###
# Run go fix
###
gofix:
	@go fix -v ./...

###
# Run tests
###
test:
	@if curl -s 127.0.0.1:8080 > /dev/null ; then echo "Plik server probably already running" ; exit 1 ; fi
	@$(GO_TEST) ./... 2>&1 | grep -v "no test files"; test $${PIPESTATUS[0]} -eq 0

###
# Run webapp unit tests (vitest)
###
test-frontend:
	@cd webapp && npm ci && npm test

###
# Run webapp e2e tests (playwright — builds frontend+server, starts fresh plikd)
###
test-frontend-e2e: frontend server
	@cd webapp && npm ci && npx playwright install chromium
	@cd webapp && npx playwright test $(if $(HEADED),--headed)

###
# Run bash client tests (unit + e2e against a temporary server)
###
test-bash-client: server
	@echo "Running bash client unit tests..."
	@bash client/plik_bash_test.sh --unit
	@echo ""
	@echo "Starting plik server for integration tests..."
	@cd server && ./plikd > /dev/null 2>&1 &
	@sleep 2
	@echo "Running bash client integration tests..."
	@bash client/plik_bash_test.sh; EXIT=$$?; \
		kill $$(lsof -ti:8080) 2>/dev/null || true; \
		exit $$EXIT

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
# Run integration tests for a single backend
# Usage: make test-backend mariadb
###
ifeq (test-backend,$(firstword $(MAKECMDGOALS)))
  BACKEND_ARG := $(wordlist 2,2,$(MAKECMDGOALS))
  $(eval $(BACKEND_ARG):;@:)
endif

test-backend:
	@testing/test_backends.sh $(BACKEND_ARG)

###
# Build documentation
###
docs: client
	@cd docs && npm ci && bash inject_version.sh && bash copy_architecture.sh && bash inject_plikrc.sh && bash inject_help.sh && npm run build

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
# Regenerate Helm chart README from values.yaml annotations
# Auto-installs helm-docs via go install if not found
###
helm-docs:
	@command -v helm-docs >/dev/null 2>&1 || { echo "Installing helm-docs..." && go install github.com/norwoodj/helm-docs/cmd/helm-docs@latest; }
	@helm-docs --chart-search-root ./charts

###
# Package Helm chart locally (auto-detects version from git tags)
###
helm: helm-docs
	@DRY_RUN=true releaser/helm_release.sh $(VERSION)

###
# Package and install Helm chart locally
###
helm-install: helm
	@helm install plik releases/plik-helm-$(VERSION).tgz

###
# Build Debian packages locally for all Linux architectures
# Requires: nfpm (go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest)
# Requires: cross-compilers (apt install crossbuild-essential-{armhf,arm64,i386})
# Set DEB_TARGETS to override (e.g. DEB_TARGETS=amd64)
###
DEB_TARGETS ?= amd64 386 arm64 arm

deb: frontend clients
	@echo ""
	@echo " Building Debian packages for $(VERSION)"
	@echo ""
	@mkdir -p releases
	@for GOARCH in $(DEB_TARGETS); do \
		DEB_ARCH=$$GOARCH ; \
		CROSS_CC="" ; \
		case "$$GOARCH" in \
			amd64) CROSS_CC="" ;; \
			386)   DEB_ARCH="i386"  ; CROSS_CC="i686-linux-gnu-gcc" ;; \
			arm64) CROSS_CC="aarch64-linux-gnu-gcc" ;; \
			arm)   DEB_ARCH="armhf" ; CROSS_CC="arm-linux-gnueabihf-gcc" ;; \
		*)     echo " Error: unknown GOARCH $$GOARCH, update the case statement in the deb target" ; exit 1 ;; \
		esac ; \
		if [ -n "$$CROSS_CC" ] && ! command -v "$$CROSS_CC" >/dev/null 2>&1; then \
			echo " Skipping $$DEB_ARCH ($$CROSS_CC not found)" ; \
			continue ; \
		fi ; \
		echo "" ; \
		echo " Building plik server ($$DEB_ARCH)" ; \
		GOOS=linux GOARCH=$$GOARCH CGO_ENABLED=1 CC=$$CROSS_CC $(GO_BUILD) -o server/plikd ./server ; \
		rm -rf release && mkdir -p release/server release/webapp ; \
		cp server/plikd release/server/plikd ; \
		cp server/plikd.cfg release/server/plikd.cfg ; \
		cp -r webapp/dist release/webapp/dist ; \
		cp -r clients release/clients ; \
		cp -r changelog release/changelog ; \
		echo " Packaging plik-server ($$DEB_ARCH)" ; \
		VERSION=$(VERSION) DEB_ARCH=$$DEB_ARCH \
			nfpm pkg --config releaser/nfpm-server.yaml --packager deb --target releases/ ; \
		if [ -f "release/clients/linux-$$GOARCH/plik" ]; then \
			echo " Packaging plik-client ($$DEB_ARCH)" ; \
			mkdir -p release/client ; \
			cp "release/clients/linux-$$GOARCH/plik" release/client/plik ; \
			VERSION=$(VERSION) DEB_ARCH=$$DEB_ARCH \
				nfpm pkg --config releaser/nfpm-client.yaml --packager deb --target releases/ ; \
		fi ; \
	done
	@rm -rf release
	@echo ""
	@echo " Packages built:"
	@ls -l releases/*.deb

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
	@rm -rf webapp/dist

###
# Remove all build files and node modules
###
clean-all: clean clean-frontend
	@rm -rf webapp/node_modules

###
# Publish existing .deb packages to the gh-pages APT repository
# Run after "make deb" to push packages without a full release
###
deb-publish:
	@SKIP_BUILD=true releaser/apt_release.sh $(VERSION)

###
# Since the client/server/version directories are not generated
# by make, we must declare these targets as phony to avoid :
# "make: `client' is up to date" cases at compile time
###
.PHONY: client clients server release helm helm-docs helm-install deb deb-publish docs test-backend test-bash-client test-frontend test-frontend-e2e
