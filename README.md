#G

This repo is initially clone from [gopher](https://github.com/jimmykuu/gopher)

##Requirements

- Go1.2+
- MongoDB
- github.com/gorilla/mux
- github.com/gorilla/sessions
- github.com/qiniu/bytes
- github.com/qiniu/rpc
- github.com/qiniu/api.v6
- labix.org/v2/mgo
- code.google.com/p/go-uuid/uuid
- github.com/jimmykuu/webhelpers
- github.com/jimmykuu/wtforms
- github.com/deferpanic/deferclient/deferclient

##Install

    $ go get github.com/jimmykuu/gopher/server


copy *etc/config.json.default* to  *etc/config.json* as the configure file



e.g.:

    {
        "host": "http://localhost:8888",
        "port": 8888,
        "db": "localhost:27017",
        "cookie_secret": "05e0ba2eca9411e18155109add4b8aac",
        "smtp_username": "username@example.com",
        "smtp_password": "password",
        "smtp_host": "smtp.example.com",
        "smtp_addr": "smtp.example.com:25",
        "from_email": "who@example.com",
        "superusers": "jimmykuu,another",
        "analytics_file": "",
        "time_zone_offset": 8,
        "static_file_version": 1,
        "go_get_path": "/tmp/download",
        "packages_download_path": "/var/go/gopher/static/download/packages",
        "public_salt": "",
		"github_auth_client_id": "example",
		"github_auth_client_secret": "example",
		"github_login_redirect": "/",
		"github_login_success_redirect": "/auth/signup",
		"deferpanic_api_key": ""
    }

start MongoDB

generater private key and certification `key.pem` and `cert.pem`

	go run $GOROOT/src/crypto/tls/generate_cert.go --host domain

Linux/Unix/OS X:

    $ $GOPATH/bin/server

Windows:

    > $GOPATH\bin\server.exe

or:

	$ go build -o binary github.com/jimmykuu/gopher/server
	$ ./binary

##Contributors

- [Contributors](https://github.com/nosqldb/G/graphs/contributors)


##License

Copyright (c) 2012-2015

Released under the MIT license:

- [www.opensource.org/licenses/MIT](http://www.opensource.org/licenses/MIT)
