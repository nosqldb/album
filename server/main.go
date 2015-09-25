package main

import (
	"github.com/nosqldb/G"
)

func main() {
	go gopher.RssRefresh()
	gopher.StartServer()
}
