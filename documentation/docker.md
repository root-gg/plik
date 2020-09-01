### Docker
Plik comes with a simple Dockerfile that allows you to run it inside a docker container.

##### Getting image from docker registry

```sh
$ docker pull rootgg/plik:latest
```

##### Building the docker image

First, you need to build the docker image :   
```sh
$ make docker
```

##### Configuration

Then you can run an instance and map the local port 8080 to the plik port :   
```sh
$ docker run -t -d -p 8080:8080 rootgg/plik
ab9b2c99da1f3e309cd3b12392b9084b5cafcca0325d7d47ff76f5b1e475d1b9
```

To use a different config file, you can map a single file to the container at runtime :   
Here, we map local folder plikd.cfg to the home/plik/server/plikd.cfg which is the default config file location in the container :   
```sh
$ docker run -t -d -p 8080:8080 -v plikd.cfg:/home/plik/server/plikd.cfg rootgg/plik
ab9b2c99da1f3e309cd3b12392b9084b5cafcca0325d7d47ff76f5b1e475d1b9
```

You can also use a volume to store uploads outside the container :   
Here, we map local folder /data to the /home/plik/server/files folder of the container which is the default upload directory :   
```sh
$ docker run -t -d -p 8080:8080 -v /data:/home/plik/server/files rootgg/plik
ab9b2c99da1f3e309cd3b12392b9084b5cafcca0325d7d47ff76f5b1e475d1b9
```

#### Configuration (via docker-compose, persistent files)

A simple `docker-compose` file setting up a `plik` instance with persistent file storage could look like this:

```yml
version: "3.2"

volumes:
  files:

services:
  app:
    image: rootgg/plik
    restart: unless-stopped
    volumes:
      - ./plikd.cfg:/home/plik/server/plikd.cfg
      - files:/home/plik/server/files
```

Note, however, that the server is run by a user `plik` with a UID/GUI of 1000:1000 (compare the [Dockerfile](../Dockerfile)), which leads to problems accessing the volume. This can be easily mitigated by changing the access rights in the volume _after_ it has been created:

1. Trigger volume creation by starting the container
```sh
$ docker-compose up -d
```

2. Change the ownership in the `files` folder
```sh
$ docker-compose exec --user root app chown plik:plik /home/plik/server/files
```


#### Configuration (via docker-compose, persistent files and metadata)

A similar `volumes`-based approach could be used to make the sqlite3 metadata database persistent. Here as well, the ownership has to be transfered. As volumes work with folders, however, some extra effort is needed to set this up.

1. Update the `docker-compose` file, adding a new volume for the database:

```yml
version: "3.2"

volumes:
  files:
  db:

services:
  app:
    image: rootgg/plik
    restart: unless-stopped
    volumes:
      - ./plikd.cfg:/home/plik/server/plikd.cfg
      - files:/home/plik/server/files
      - db:/home/plik/server/db
```

2. Start the container:
```sh
$ docker-compose up -d
```

3. Change ownership of the new volume:
```sh
$ docker-compose exec --user root app chown plik:plik /home/plik/server/db
```

4. Update `plikd.cfg` in order to use the volume to hold the sqlite3 database:
```
[MetadataBackendConfig]
    Driver = "sqlite3"
    ConnectionString = "db/plik.db"
    Debug = false # Log SQL requests
```

5. Restart plik:
```sh
$ docker-compose stop
$ docker-compose up -d
```

6. Setup some local users: 
```sh
$ docker-compose exec app ./plikd user --help
$ docker-compose exec app ./plikd user create --admin --login mytestuser --name "My Test User" 
```
      
