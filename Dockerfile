#####
####
### Plik - Docker file
##
#

ARG ALPINE_VERSION=3.9
ARG GOLANG_VERSION=1.12.3

# Let's setup the build environment

FROM golang:${GOLANG_VERSION}-alpine${ALPINE_VERSION} AS buildenv

ENV GIT_BRANCH=dev

RUN apk add --update --no-cache \
	git bash make nodejs-npm curl

WORKDIR /go/src/github.com/root-gg/plik/

# Get tools for testing
RUN go get golang.org/x/lint/golint

# Fetch code and use a nasty hack to make docker build ignore "go get" ignore
# the "undefined: common.GetBuildInfo" error from misc.go
RUN git clone https://github.com/root-gg/plik . --branch $GIT_BRANCH

# Build all the binaries
RUN make

FROM alpine:${ALPINE_VERSION}

# Prepare base image
RUN apk add --update --no-cache shadow

RUN useradd -d /opt/plik -m plik

# Get the binaries from the builder image
WORKDIR /opt/plik

# Add clients and server blobs (you can uncomment the clients line to shrink your image even further)
COPY --from=buildenv /go/src/github.com/root-gg/plik/clients       /opt/plik/clients
COPY --from=buildenv /go/src/github.com/root-gg/plik/server/public /opt/plik/public
COPY --from=buildenv /go/src/github.com/root-gg/plik/server/plikd  /opt/plik/server/plikd

# Add configuration and fix permissions
ADD server/plikd.cfg /opt/plik/plikd.cfg
RUN chown -R plik:plik /opt/plik

EXPOSE 8080

USER plik

CMD ["/opt/plik/server/plikd"]
