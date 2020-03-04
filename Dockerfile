#####
####
### Plik - Docker file
##
#

# Let's start with a fresh debian image
FROM debian:buster

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
ADD webapp /home/plik/webapp/

# Set permission
RUN chown -R plik:plik /home/plik
RUN chmod +x /home/plik/server/plikd

# Launch Plik server
USER plik
WORKDIR /home/plik/server
CMD ./plikd