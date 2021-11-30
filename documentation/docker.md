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


### Usage with docker-compose

Use this example file to set up your instance with all needed volumes. All files, accounts and tokens will be persistent in this configuration.
Ensure that all volumes belong to user/group `1000`.

```yaml
version: "3.5"
services:
  plik:
    image: rootgg/plik:latest
    container_name: plik
    ports:
      - 8091:8080
    volumes:
      - ./plikd.cfg:/home/plik/server/plikd.cfg
      - ./plik.db:/home/plik/server/plik.db
      - ./plik.db-wal:/home/plik/server/plik.db-wal
      - ./data:/home/plik/server/files
      - ./metadata:/home/plik/server/metadata    
    restart: "unless-stopped"
```
