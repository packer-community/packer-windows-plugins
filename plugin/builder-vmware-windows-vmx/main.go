package main

import (
	"github.com/packer-community/packer-windows-plugins/builder/vmware-windows/vmx"
	"github.com/mitchellh/packer/packer/plugin"
)

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(vmx.Builder))
	server.Serve()
}
