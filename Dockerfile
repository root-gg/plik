##################################################################################
FROM node:12.15-alpine AS plik-frontend-builder

# Install needed binaries
RUN apk add --no-cache git make bash

# Add the source code
ADD Makefile .
ADD webapp /webapp

##################################################################################
FROM plik-frontend-builder AS plik-frontend

RUN make clean-frontend frontend

##################################################################################
FROM golang:1.15.3-buster AS plik-builder

# Install needed binaries
RUN apt-get update && apt-get install -y build-essential crossbuild-essential-armhf crossbuild-essential-armel crossbuild-essential-arm64 crossbuild-essential-i386

# Prepare the source location
RUN mkdir -p /go/src/github.com/root-gg/plik
WORKDIR /go/src/github.com/root-gg/plik

# Add the source code ( see .dockerignore )
ADD . .

# Copy webapp build from previous stage
COPY --from=plik-frontend /webapp/dist webapp/dist

##################################################################################
FROM plik-builder AS plik-releases

ARG CLIENT_TARGETS=""
ENV CLIENT_TARGETS=$CLIENT_TARGETS

ARG TARGETS=""
ENV TARGETS=$TARGETS

RUN releaser/releaser.sh

##################################################################################
FROM alpine:3.12 AS plik-base

RUN apk add --no-cache ca-certificates

# Create plik user
ENV USER=plik
ENV UID=1000

# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/home/plik" \
    --shell "/bin/false" \
    --uid "${UID}" \
    "${USER}"

EXPOSE 8080
USER plik
WORKDIR /home/plik/server
CMD ./plikd

##################################################################################
FROM plik-base AS plik-amd64

COPY --from=plik-releases --chown=1000:1000 /go/src/github.com/root-gg/plik/releases/plik-*-linux-amd64 /home/plik/

##################################################################################
FROM plik-base AS plik-386

COPY --from=plik-releases --chown=1000:1000 /go/src/github.com/root-gg/plik/releases/plik-*-linux-386 /home/plik/

##################################################################################
FROM plik-base AS plik-arm

COPY --from=plik-releases --chown=1000:1000 /go/src/github.com/root-gg/plik/releases/plik-*-linux-arm /home/plik/

##################################################################################
FROM plik-base AS plik-arm64

COPY --from=plik-releases --chown=1000:1000 /go/src/github.com/root-gg/plik/releases/plik-*-linux-arm64 /home/plik/