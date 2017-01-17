Testing Plik backends
=====================

Here you'll find some scripts to run testing instances of plik backends using docker.

First start the docker instance you want to test ( mongodb/swift/weedfs ) :

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