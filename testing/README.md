Testing Plik backends
=====================

Here you'll find some scripts to run testing instances of Plik backends using docker.

First start the docker instance you want to test :

```
$ testing/backend/run.sh start 
```

Then start a plik instance using this backend :

```
$ make
$ server/plikd --config testing/backend/plikd.cfg
```

To terminate the docker instance run :

```
$ testing/backend/run.sh stop 
```

To run tests for a specific backend
```
$ testing/test-backends.sh backend
```

To run a specific test
```
$ testing/test-backends.sh backend test_name
```

To target a specific version/tag for the docker image
```
DOCKER_VERSION="XXX" testing/test_backends.sh backend
```