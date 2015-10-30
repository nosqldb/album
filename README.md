#G

##Requirements

- Go1.5+
- MongoDB
- github.com/gorilla/mux
- github.com/gorilla/sessions
- github.com/qiniu/bytes
- github.com/qiniu/rpc
- github.com/qiniu/api.v6
- labix.org/v2/mgo
- github.com/pborman/uuid
- github.com/jimmykuu/wtforms
- github.com/deferpanic/deferclient/deferclient

##Install

    $ go get github.com/nosqldb/G/server


copy *etc/config.json.default* to  *etc/config.json* as the configure file

start MongoDB

generater private key and certification `key.pem` and `cert.pem`

	go run $GOROOT/src/crypto/tls/generate_cert.go --host domain

Linux/Unix/OS X:

    $ $GOPATH/bin/server

Windows:

    > $GOPATH\bin\server.exe

or:

	$ go build -o binary github.com/nosqldb/G/server
	$ ./binary

##Contributors

- [Contributors](https://github.com/nosqldb/G/graphs/contributors)


##License

Copyright (c) 2012-2015

Released under the MIT license:

- [www.opensource.org/licenses/MIT](http://www.opensource.org/licenses/MIT)
