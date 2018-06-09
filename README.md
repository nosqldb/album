#G

##Requirements

- Go1.9+
- MongoDB
- github.com/gorilla/mux
- github.com/gorilla/sessions
- github.com/qiniu/bytes
- github.com/qiniu/rpc
- github.com/qiniu/api.v6
- labix.org/v2/mgo
- github.com/pborman/uuid
- github.com/jimmykuu/wtforms

##Install

    $ go get github.com/nosqldb/album/server


copy *etc/config.json.default* to  *etc/config.json* as the configure file

start MongoDB

generater private key and certification `key.pem` and `cert.pem`

	go run $GOROOT/src/crypto/tls/generate_cert.go --host domain

Linux/Unix/OS X:

    $ $GOPATH/bin/server

Windows:

    > $GOPATH\bin\server.exe

or:

	$ go build -o G github.com/nosqldb/album/server
	$ ./G
