package main

import (
	"github.com/bradenn/turnin-compute/config"
	"github.com/bradenn/turnin-compute/server"
)

func main() {
	config.Init()
	server.Init()
}
