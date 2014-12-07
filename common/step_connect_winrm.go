package common

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	plugin "github.com/dylanmei/packer-communicator-winrm/communicator/winrm"
	"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
)

// StepConnectWinRM is a multistep Step implementation that waits for WinRM
// to become available. It gets the connection information from a single
// configuration when creating the step.
//
// Uses:
//   ui packer.Ui
//
// Produces:
//   communicator packer.Communicator
type StepConnectWinRM struct {
	// WinRMAddress is a function that returns the TCP address to connect to
	// for WinRM. This is a function so that you can query information
	// if necessary for this address.
	WinRMAddress func(multistep.StateBag) (string, error)

	// The user name to connect to WinRM as
	WinRMUser string

	// The user password
	WinRMPassword string

	// WinRMWaitTimeout is the total timeout to wait for WinRM to become available.
	WinRMWaitTimeout time.Duration

	comm packer.Communicator
}

func (s *StepConnectWinRM) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)

	var comm packer.Communicator
	var err error

	cancel := make(chan struct{})
	waitDone := make(chan bool, 1)
	go func() {
		ui.Say("Waiting for WinRM to become available...")
		comm, err = s.waitForWinRM(state, cancel)
		waitDone <- true
	}()

	log.Printf("Waiting for WinRM, up to timeout: %s", s.WinRMWaitTimeout)
	timeout := time.After(s.WinRMWaitTimeout)
WaitLoop:
	for {
		// Wait for either WinRM to become available, a timeout to occur,
		// or an interrupt to come through.
		select {
		case <-waitDone:
			if err != nil {
				ui.Error(fmt.Sprintf("Error waiting for WinRM: %s", err))
				return multistep.ActionHalt
			}

			ui.Say("Connected to WinRM!")
			s.comm = comm
			state.Put("communicator", comm)
			break WaitLoop
		case <-timeout:
			err := fmt.Errorf("Timeout waiting for WinRM.")
			state.Put("error", err)
			ui.Error(err.Error())
			close(cancel)
			return multistep.ActionHalt
		case <-time.After(1 * time.Second):
			if _, ok := state.GetOk(multistep.StateCancelled); ok {
				// The step sequence was cancelled, so cancel waiting for WinRM
				// and just start the halting process.
				close(cancel)
				log.Println("Interrupt detected, quitting waiting for WinRM.")
				return multistep.ActionHalt
			}
		}
	}

	return multistep.ActionContinue
}

func (s *StepConnectWinRM) Cleanup(multistep.StateBag) {
}

func (s *StepConnectWinRM) waitForWinRM(state multistep.StateBag, cancel <-chan struct{}) (packer.Communicator, error) {
	var comm packer.Communicator
	for {
		select {
		case <-cancel:
			log.Println("WinRM wait cancelled. Exiting loop.")
			return nil, errors.New("WinRM wait cancelled")
		case <-time.After(5 * time.Second):
		}

		address, err := s.WinRMAddress(state)
		if err != nil {
			log.Printf("Error getting WinRM address: %s", err)
			continue
		}

		endpoint, err := newEndpoint(address)
		if err != nil {
			log.Printf("Incorrect format for WinRM address: %s", err)
			continue
		}

		log.Println("Attempting WinRM connection (timeout: %s)", s.WinRMWaitTimeout)

		comm, err = plugin.New(endpoint, s.WinRMUser, s.WinRMPassword, s.WinRMWaitTimeout)
		if err != nil {
			log.Printf("WinRM connection err: %s", err)
			continue
		}

		break
	}

	return comm, nil
}

func newEndpoint(address string) (*winrm.Endpoint, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	iport, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}
	return &winrm.Endpoint{host, iport}, nil
}
