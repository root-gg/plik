#####
####
### Plik - Docker file
##
#

FROM debian:latest

ADD plikd.cfg /home/plik/server/plikd.cfg

RUN apt update && apt install -y ca-certificates curl && \
        useradd plik && \
        curl -s https://api.github.com/repos/root-gg/plik/releases/latest | grep -o "https://.*linux-64bits.tar.gz" | xargs curl -L --output /tmp/plik.tar.gz && \
        cd /home/plik && tar --strip 1 -xvzf /tmp/plik.tar.gz && rm -f /tmp/plik.tar.gz && \
        chown -R plik:plik /home/plik && \
        chmod +x /home/plik/server/plikd && \
        apt -y purge curl && rm -rf /var/lib/apt/lists/* && rm -rf /var/cache

EXPOSE 8080

USER plik
WORKDIR /home/plik/server
CMD ["/home/plik/server/plikd"]
