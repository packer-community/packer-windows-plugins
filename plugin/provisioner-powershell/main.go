package main

import (
	"github.com/mitchellh/packer/packer/plugin"
	powershell "github.com/packer-community/packer-windows-plugins/provisioner/powershell"
)

func main() {

	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterProvisioner(new(powershell.Provisioner))
	server.Serve()
}
