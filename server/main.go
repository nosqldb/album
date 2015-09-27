package main

import (
	"github.com/nosqldb/G"
)

func main() {
	go g.RssRefresh()
	g.StartServer()
}
