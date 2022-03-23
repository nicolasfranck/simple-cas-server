Simple CAS Server
=================

CAS Server to test your CAS Single Sign On setup

No user or attribute management

Login with username and password equal to each other

Requirements
------------

* go 1.17

Build
-----

```
$ go build
$ ./simple-cas-server -bind :4000
```

Usage
-----

* set the cas base url to http://localhost:4000, depending on how you started the cas server
