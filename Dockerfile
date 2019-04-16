#####
####
### Plik - Docker file
##
#

ARG ALPINE_VERSION=3.9
ARG GOLANG_VERSION=1.12.3

# Let's setup the build environment

#FROM golang:${GOLANG_VERSION}-alpine${ALPINE_VERSION} AS buildenv
FROM golang:1.12.3-stretch AS buildenv

WORKDIR /go/src/github.com/root-gg/plik/

RUN apt update && apt install -y \
	   git-core build-essential bash curl zip gpg file openssl \
	&& curl -o- https://raw.githubusercontent.com/creationix/nvm/v0.34.0/install.sh | bash \
	&& curl -sL https://deb.nodesource.com/setup_11.x | bash - && apt-get install -y nodejs

RUN go get golang.org/x/lint/golint

# Add the plik code
ADD . .

# Build all the binaries
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 make test 
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make

# Last stage, let's only save, what we actually need
FROM alpine:${ALPINE_VERSION}

RUN apk add --update --no-cache shadow ca-certificates \
	&& useradd -d /opt/plik -m plik

WORKDIR /opt/plik

RUN apk add --update --no-cache ca-certificates shadow

USER plik

WORKDIR /opt/plik

# Add clients and server blobs (you can uncomment the clients line to shrink your image even further)
COPY --from=buildenv --chown=plik /go/src/github.com/root-gg/plik/clients       /opt/plik/clients
COPY --from=buildenv --chown=plik /go/src/github.com/root-gg/plik/server/public /opt/plik/public
COPY --from=buildenv --chown=plik /go/src/github.com/root-gg/plik/server/plikd  /opt/plik/server/plikd

# Add configuration
ADD server/plikd.cfg /opt/plik/plikd.cfg

EXPOSE 8080

CMD ["/opt/plik/server/plikd"]
