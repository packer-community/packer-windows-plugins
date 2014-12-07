package common

import (
	"fmt"
	"github.com/mitchellh/multistep"
	vboxcommon "github.com/mitchellh/packer/builder/virtualbox/common"
	"github.com/mitchellh/packer/packer"
	"log"
	"math/rand"
	"net"
)

// This step adds a NAT port forwarding definition so that WinRM is available
// on the guest machine.
//
// Uses:
//   driver Driver
//   ui packer.Ui
//   vmName string
//
// Produces:
type StepForwardWinRM struct {
	GuestPort   uint
	HostPortMin uint
	HostPortMax uint
}

func (s *StepForwardWinRM) Run(state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(vboxcommon.Driver)
	ui := state.Get("ui").(packer.Ui)
	vmName := state.Get("vmName").(string)

	log.Printf("Looking for available WinRM port between %d and %d",
		s.HostPortMin, s.HostPortMax)
	var winrmHostPort uint
	var offset uint = 0

	portRange := int(s.HostPortMax - s.HostPortMin)
	if portRange > 0 {
		// Have to check if > 0 to avoid a panic
		offset = uint(rand.Intn(portRange))
	}

	for {
		winrmHostPort = offset + s.HostPortMin
		log.Printf("Trying port: %d", winrmHostPort)
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", winrmHostPort))
		if err == nil {
			defer l.Close()
			break
		}
	}

	// Create a forwarded port mapping to the VM
	ui.Say(fmt.Sprintf("Creating forwarded port mapping for WinRM (host port %d)", winrmHostPort))
	command := []string{
		"modifyvm", vmName,
		"--natpf1",
		fmt.Sprintf("packerwinrm,tcp,127.0.0.1,%d,,%d", winrmHostPort, s.GuestPort),
	}
	if err := driver.VBoxManage(command...); err != nil {
		err := fmt.Errorf("Error creating port forwarding rule: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Save the port we're using so that future steps can use it.
	// Alias to 'ssh' so that other steps can use it
	state.Put("winrmHostPort", winrmHostPort)
	state.Put("sshHostPort", winrmHostPort)

	return multistep.ActionContinue
}

func (s *StepForwardWinRM) Cleanup(state multistep.StateBag) {}
