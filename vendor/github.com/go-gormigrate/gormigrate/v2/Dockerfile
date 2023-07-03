FROM golang:1.20-alpine

RUN apk --no-cache add alpine-sdk

WORKDIR /gormigrate
COPY . .

RUN go mod download
