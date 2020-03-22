##################################################################################
# Builder 1: make frontend
FROM node:12.15-alpine AS builder-frontend

# Install needed binaries
RUN apk add --no-cache git make bash

# Add the source code
ADD Makefile .
ADD webapp /webapp

RUN make frontend

##################################################################################
# Builder 2: make server and clients
FROM golang:1.14-alpine AS builder-go

# Install needed binaries
RUN apk add --no-cache git make bash gcc g++

# Prepare the source location
RUN mkdir -p /go/src/github.com/root-gg/plik
WORKDIR /go/src/github.com/root-gg/plik

# Add the source code ( see .dockerignore )
ADD . .

# Build everything
RUN make clients
RUN make server

##################################################################################
# Builder 3: we need ca-certificates in the final container
FROM alpine:3.11 AS builder-env

# Install needed binaries
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

COPY --from=builder-frontend --chown=1000:1000 /webapp/dist /home/plik/webapp/dist

COPY --from=builder-go --chown=1000:1000 /go/src/github.com/root-gg/plik/server/plikd /home/plik/server/plikd
COPY --from=builder-go --chown=1000:1000 /go/src/github.com/root-gg/plik/server/plikd.cfg /home/plik/server/plikd.cfg
COPY --from=builder-go --chown=1000:1000 /go/src/github.com/root-gg/plik/clients /home/plik/clients
COPY --from=builder-go --chown=1000:1000 /go/src/github.com/root-gg/plik/changelog /home/plik/changelog

EXPOSE 8080
USER plik
WORKDIR /home/plik/server
CMD ./plikd