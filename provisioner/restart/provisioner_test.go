package restart

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/mitchellh/packer/packer"
	"testing"
	"time"
)

func testConfig() map[string]interface{} {
	return map[string]interface{}{}
}

func TestProvisioner_Impl(t *testing.T) {
	var raw interface{}
	raw = &Provisioner{}
	if _, ok := raw.(packer.Provisioner); !ok {
		t.Fatalf("must be a Provisioner")
	}
}

func TestProvisionerPrepare_Defaults(t *testing.T) {
	var p Provisioner
	config := testConfig()

	err := p.Prepare(config)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if p.config.RawRestartTimeout != "5m" {
		t.Errorf("unexpected remote path: %s", p.config.RawRestartTimeout)
	}

	if p.config.RestartCommand != "shutdown /r /c \"packer restart\" /t 5 && net stop winrm" {
		t.Errorf("unexpected remote path: %s", p.config.RestartCommand)
	}

	if p.config.restartTimeout != 5*time.Minute {
		t.Errorf("Expected default restartTimeout to be 5 minutes")
	}
}

func TestProvisionerPrepare_ConfigRetryTimeout(t *testing.T) {
	var p Provisioner
	config := testConfig()
	config["restart_timeout"] = "1m"

	err := p.Prepare(config)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if p.config.RawRestartTimeout != "1m" {
		t.Errorf("unexpected remote path: %s", p.config.RawRestartTimeout)
	}

	if p.config.restartTimeout != 1*time.Minute {
		t.Errorf("Expected default restartTimeout to be 5 minutes")
	}
}

func TestProvisionerPrepare_ConfigErrors(t *testing.T) {
	var p Provisioner
	config := testConfig()
	config["restart_timeout"] = "m"

	err := p.Prepare(config)
	if err == nil {
		t.Fatal("Expected error parsing restart_timeout but did not receive one.")
	}
}

func TestProvisionerPrepare_InvalidKey(t *testing.T) {
	var p Provisioner
	config := testConfig()

	// Add a random key
	config["i_should_not_be_valid"] = true
	err := p.Prepare(config)
	if err == nil {
		t.Fatal("should have error")
	}
}

func testUi() *packer.BasicUi {
	return &packer.BasicUi{
		Reader:      new(bytes.Buffer),
		Writer:      new(bytes.Buffer),
		ErrorWriter: new(bytes.Buffer),
	}
}

func TestProvisionerProvision_Success(t *testing.T) {
	config := testConfig()

	// Defaults provided by Packer
	ui := testUi()
	p := new(Provisioner)

	// Defaults provided by Packer
	comm := new(packer.MockCommunicator)
	p.Prepare(config)
	//	waitForRestart = func(p *Provisioner) error {
	//		return nil
	//	}
	waitForCommunicatorOld := waitForCommunicator
	waitForCommunicator = func(p *Provisioner) error {
		return nil
	}
	err := p.Provision(ui, comm)
	if err != nil {
		t.Fatal("should not have error")
	}

	expectedCommand := DefaultRestartCommand

	// Should run the command without alteration
	if comm.StartCmd.Command != expectedCommand {
		t.Fatalf("Expect command to be: %s, got %s", expectedCommand, comm.StartCmd.Command)
	}
	// Set this back!
	waitForCommunicator = waitForCommunicatorOld
}

func TestProvisionerProvision_RestartCommandFail(t *testing.T) {
	config := testConfig()
	ui := testUi()
	p := new(Provisioner)
	comm := new(packer.MockCommunicator)
	comm.StartStderr = "WinRM terminated"
	comm.StartExitStatus = 1

	p.Prepare(config)
	err := p.Provision(ui, comm)
	if err == nil {
		t.Fatal("should have error")
	}
}
func TestProvisionerProvision_WaitForRestartFail(t *testing.T) {
	config := testConfig()

	// Defaults provided by Packer
	ui := testUi()
	p := new(Provisioner)

	// Defaults provided by Packer
	comm := new(packer.MockCommunicator)
	p.Prepare(config)
	waitForCommunicatorOld := waitForCommunicator
	waitForCommunicator = func(p *Provisioner) error {
		return fmt.Errorf("Machine did not restart properly")
	}
	err := p.Provision(ui, comm)
	if err == nil {
		t.Fatal("should have error")
	}

	// Set this back!
	waitForCommunicator = waitForCommunicatorOld
}

func TestProvision_waitForRestartTimeout(t *testing.T) {
	config := testConfig()
	config["restart_timeout"] = "1ms"
	ui := testUi()
	p := new(Provisioner)
	comm := new(packer.MockCommunicator)
	var err error

	p.Prepare(config)
	waitForCommunicatorOld := waitForCommunicator
	waitDone := make(chan bool)

	// Block until cancel comes through
	waitForCommunicator = func(p *Provisioner) error {
		for {
			select {
			case <-waitDone:
			}
		}
	}

	go func() {
		err = p.Provision(ui, comm)
		waitDone <- true
	}()
	<-waitDone

	if err == nil {
		t.Fatal("should not have error")
	}

	// Set this back!
	waitForCommunicator = waitForCommunicatorOld

}

func TestProvision_waitForCommunitactor(t *testing.T) {
	config := testConfig()

	// Defaults provided by Packer
	ui := testUi()
	p := new(Provisioner)

	// Defaults provided by Packer
	comm := new(packer.MockCommunicator)
	p.comm = comm
	p.ui = ui
	comm.StartStderr = "WinRM terminated"
	comm.StartExitStatus = 1
	p.Prepare(config)
	err := waitForCommunicator(p)

	if err != nil {
		t.Fatal("should not have error, got: %s", err.Error())
	}

	expectedCommand := DefaultRestartCheckCommand

	// Should run the command without alteration
	if comm.StartCmd.Command != expectedCommand {
		t.Fatalf("Expect command to be: %s, got %s", expectedCommand, comm.StartCmd.Command)
	}
}

func TestRetryable(t *testing.T) {
	config := testConfig()

	count := 0
	retryMe := func() error {
		t.Logf("RetryMe, attempt number %d", count)
		if count == 2 {
			return nil
		}
		count++
		return errors.New(fmt.Sprintf("Still waiting %d more times...", 2-count))
	}
	retryableSleep = 50 * time.Millisecond
	p := new(Provisioner)
	p.config.RawRestartTimeout = "155ms"
	err := p.Prepare(config)
	err = p.retryable(retryMe)
	if err != nil {
		t.Fatalf("should not have error retrying funuction")
	}

	count = 0
	p.config.RawRestartTimeout = "10ms"
	err = p.Prepare(config)
	err = p.retryable(retryMe)
	if err == nil {
		t.Fatalf("should have error retrying funuction")
	}
}

func TestCancel(t *testing.T) {
	config := testConfig()

	// Defaults provided by Packer
	ui := testUi()
	p := new(Provisioner)

	var err error

	comm := new(packer.MockCommunicator)
	p.Prepare(config)
	waitDone := make(chan bool)

	// Block until cancel comes through
	waitForCommunicator = func(p *Provisioner) error {
		for {
			select {
			case <-waitDone:
			}
		}
	}

	// Create two go routines to provision and cancel in parallel
	// Provision will block until cancel happens
	go func() {
		err = p.Provision(ui, comm)
		waitDone <- true
	}()

	go func() {
		p.Cancel()
	}()
	<-waitDone

	// Expect interupt error
	if err == nil {
		t.Fatal("should have error")
	}
}
