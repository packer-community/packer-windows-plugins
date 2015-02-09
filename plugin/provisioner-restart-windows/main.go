package main

import (
	"github.com/mitchellh/packer/packer/plugin"
	restartwindows "github.com/packer-community/packer-windows-plugins/provisioner/restart"
)

func main() {

	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterProvisioner(new(restartwindows.Provisioner))
	server.Serve()
}
