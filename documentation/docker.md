### Docker
Plik comes with a simple Dockerfile that allows you to run it inside a docker container.

##### Getting image from docker registry

```sh
$ docker pull rootgg/plik:latest
```

##### Building the docker image

First, you need to pull the plik source :   
```sh
$ git clone https://github.com/root-gg/plik
```
Then, switch to the directory and build the Docker image:
```sh
$ cd plik
$ docker build -t plik .
```

##### Configuration

Then you can run an instance and map the local port 8080 to the plik port :   
```sh
$ docker run -t -d -p 8080:8080 rootgg/plik
ab9b2c99da1f3e309cd3b12392b9084b5cafcca0325d7d47ff76f5b1e475d1b9
```

To use a different config file, you can map a single file to the container at runtime :   
Here, we map local folder plikd.cfg to the /opt/plik/plikd.cfg which is the default config file location in the container :   
```sh
$ docker run -t -d -p 8080:8080 -v plikd.cfg:/opt/plik/plikd.cfg rootgg/plik
ab9b2c99da1f3e309cd3b12392b9084b5cafcca0325d7d47ff76f5b1e475d1b9
```

You can also use a volume to store uploads outside the container :   
Here, we map local folder /data to the /opt/plik/files folder of the container which is the default upload directory :   
```sh
$ docker run -t -d -p 8080:8080 -v /data:/opt/plik/files rootgg/plik
ab9b2c99da1f3e309cd3b12392b9084b5cafcca0325d7d47ff76f5b1e475d1b9
```

If you want to use a volume for the files, you most probably also want to use a volume for the database:
```sh
$ docker run -t -d -p 8080:8080 -v ./plik.db:/opt/plik/plik.db -v /data:/opt/plik/files rootgg/plik
ab9b2c99da1f3e309cd3b12392b9084b5cafcca0325d7d47ff76f5b1e475d1b9
```
