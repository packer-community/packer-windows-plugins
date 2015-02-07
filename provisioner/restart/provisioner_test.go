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

	if p.config.RawStartRetryTimeout != "5m" {
		t.Errorf("unexpected remote path: %s", p.config.RawStartRetryTimeout)
	}

	if p.config.RestartCommand != "shutdown /r /c \"packer restart\" /t 5 && net stop winrm" {
		t.Errorf("unexpected remote path: %s", p.config.RestartCommand)
	}

	if p.config.startRetryTimeout != 5*time.Minute {
		t.Errorf("Expected default startRetryTimeout to be 5 minutes")
	}
}

func TestProvisionerPrepare_ConfigRetryTimeout(t *testing.T) {
	var p Provisioner
	config := testConfig()
	config["start_retry_timeout"] = "1m"

	err := p.Prepare(config)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if p.config.RawStartRetryTimeout != "1m" {
		t.Errorf("unexpected remote path: %s", p.config.RawStartRetryTimeout)
	}

	if p.config.startRetryTimeout != 1*time.Minute {
		t.Errorf("Expected default startRetryTimeout to be 5 minutes")
	}
}

func TestProvisionerPrepare_ConfigErrors(t *testing.T) {
	var p Provisioner
	config := testConfig()
	config["start_retry_timeout"] = "m"

	err := p.Prepare(config)
	if err == nil {
		t.Fatal("Expected error parsing start_retry_timeout but did not receive one.")
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
	err := p.Provision(ui, comm)
	if err != nil {
		t.Fatal("should not have error")
	}

	expectedCommand := DefaultRestartCommand

	// Should run the command without alteration
	if comm.StartCmd.Command != expectedCommand {
		t.Fatalf("Expect command to be: %s, got %s", expectedCommand, comm.StartCmd.Command)
	}
}
func TestProvisionerProvision_Fail(t *testing.T) {
	config := testConfig()

	// Defaults provided by Packer
	ui := testUi()
	p := new(Provisioner)

	// Defaults provided by Packer
	comm := new(packer.MockCommunicator)
	comm.StartStderr = "WinRM terminated"
	comm.StartExitStatus = 1
	p.Prepare(config)
	err := p.Provision(ui, comm)
	if err == nil {
		t.Fatal("should have error")
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
	p.config.RawStartRetryTimeout = "155ms"
	err := p.Prepare(config)
	err = p.retryable(retryMe)
	if err != nil {
		t.Fatalf("should not have error retrying funuction")
	}

	count = 0
	p.config.RawStartRetryTimeout = "10ms"
	err = p.Prepare(config)
	err = p.retryable(retryMe)
	if err == nil {
		t.Fatalf("should have error retrying funuction")
	}
}

func TestCancel(t *testing.T) {
	// Don't actually call Cancel() as it performs an os.Exit(0)
	// which kills the 'go test' tool
}
