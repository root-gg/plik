##################################################################################
FROM --platform=$BUILDPLATFORM node:24-alpine AS plik-frontend-builder

# Install needed binaries
RUN apk add --no-cache git make bash

# Add the source code
COPY Makefile .
COPY webapp /webapp

RUN make clean-frontend frontend

##################################################################################
FROM --platform=$BUILDPLATFORM golang:1-bookworm AS plik-client-builder

# Prepare the source location
RUN mkdir -p /go/src/github.com/root-gg/plik
WORKDIR /go/src/github.com/root-gg/plik

ARG CLIENT_TARGETS=""
ENV CLIENT_TARGETS=$CLIENT_TARGETS

ARG VERSION=""
ENV VERSION=$VERSION

# Add the source code ( see .dockerignore )
COPY . .

# Build all clients once ( pure Go cross-compilation, no CGO needed )
RUN releaser/build_clients.sh

##################################################################################
FROM --platform=$BUILDPLATFORM golang:1-bookworm AS plik-builder

# Install needed binaries for server cross-compilation
RUN apt-get update && apt-get install -y build-essential crossbuild-essential-armhf crossbuild-essential-armel crossbuild-essential-arm64 crossbuild-essential-i386

# Prepare the source location
RUN mkdir -p /go/src/github.com/root-gg/plik
WORKDIR /go/src/github.com/root-gg/plik

# Copy webapp build from previous stage
COPY --from=plik-frontend-builder /webapp/dist webapp/dist

# Copy pre-built clients from previous stage
COPY --from=plik-client-builder /go/src/github.com/root-gg/plik/clients clients

ARG TARGETOS TARGETARCH TARGETVARIANT CC
ENV TARGETOS=$TARGETOS
ENV TARGETARCH=$TARGETARCH
ENV TARGETVARIANT=$TARGETVARIANT
ENV CC=$CC

ARG VERSION=""
ENV VERSION=$VERSION

# Add the source code ( see .dockerignore )
COPY . .

RUN releaser/build_server_release.sh

##################################################################################
FROM scratch AS plik-clients-archive

COPY --from=plik-client-builder --chown=1000:1000 /go/src/github.com/root-gg/plik/clients /

##################################################################################
FROM scratch AS plik-release-archive

COPY --from=plik-builder --chown=1000:1000 /go/src/github.com/root-gg/plik/plik-server-*.tar.gz /

##################################################################################
FROM alpine:3.24 AS plik-image

RUN apk add --no-cache ca-certificates

# Create plik user
ENV USER=plik
ENV UID=1000

# See https://stackoverflow.com/a/55757473/12429735
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/home/plik" \
    --shell "/bin/false" \
    --uid "${UID}" \
    "${USER}"

COPY --from=plik-builder --chown=1000:1000 /go/src/github.com/root-gg/plik/release /home/plik/

EXPOSE 8080
USER plik
WORKDIR /home/plik/server
CMD ["./plikd"]
