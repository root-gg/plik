FROM golang:1.4-wheezy
MAINTAINER Loïc PORTE<bewiwi@gmail.com>
EXPOSE 8080
RUN curl -sL https://deb.nodesource.com/setup | bash -
RUN apt-get update && apt-get install nodejs npm  -y
COPY ./ /go/src/github.com/root-gg/plik
WORKDIR /go/src/github.com/root-gg/plik
RUN make install-devtools &&\
    make install DEST_DIR=/opt/plik && rm -rf /usr/src/go
WORKDIR /opt/plik/server/
CMD ./plikd
