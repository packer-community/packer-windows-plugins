package main

import (
	"github.com/mitchellh/packer/packer/plugin"
	"github.com/packer-community/packer-windows-plugins/builder/amazon-windows/instance"
)

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(instance.Builder))
	server.Serve()
}
