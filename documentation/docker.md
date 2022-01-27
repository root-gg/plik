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

Use this example file to set up your instance with all persistent data/metadata. All files, accounts and tokens will be persistent in this configuration.
Adjust directories to your like.

```
$ cd ~
$ mkdir plik
$ curl https://raw.githubusercontent.com/root-gg/plik/master/server/plikd.cfg # copy server configuration
$ plik mkdir data # create directory to save files and/or metadata outside of the docker image
$ plik chown 1000:1000 data # match UIDs with docker
$ plik chown 1000:1000 plikd.cfg # match UIDs with docker
```

Edit plikd.cfg to point the metadata and/or data to a mountpoint that you can match in docker-compose (/data in this example)
```
DataBackend = "file"
[DataBackendConfig]
    Directory = "/data/files" # <===

[MetadataBackendConfig]
    Driver = "sqlite3"
    ConnectionString = "/data/plik.db" # <===
```

Create a docker-compose.yml file with the following content
```yaml
version: "2"
services:
  plik:
    image: rootgg/plik:1.3.5
    container_name: plik
    volumes:
      - /home/{user}/plik/plikd.cfg:/home/plik/server/plikd.cfg
      - /home/{user}/plik/data:/data
    ports:
      - 8080:8080   
    restart: "unless-stopped"
```

```
$ docker-compose up
Starting plik ... done
Attaching to plik
plik    | [01/27/2022 10:48:26][INFO    ] Starting plikd server v...
plik    | [01/27/2022 10:48:26][INFO    ] Starting server at http://0.0.0.0:8080
```

Reach out on telegram or this thread if you need help or have any suggestion :
https://github.com/root-gg/plik/issues/326
