#####
####
### Plik - Docker file
##
#

# Let's start with a fresh debian jessie
FROM debian:jessie

# Some generic information
MAINTAINER Charles-Antoine Mathieu
MAINTAINER Mathieu Bodjikian

RUN apt-get update && apt-get install -y ca-certificates

# Create user
RUN useradd -U -d /home/plik -m -s /bin/false plik 

# Expose the plik port
EXPOSE 8080

# Copy plik
ADD server /home/plik/server/
ADD clients /home/plik/clients/
RUN chown -R plik:plik /home/plik
RUN chmod +x /home/plik/server/plikd

# Launch it
USER plik
WORKDIR /home/plik/server
CMD ./plikd

