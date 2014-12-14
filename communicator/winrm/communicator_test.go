package winrm

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/packer/packer"
)

func TestCommIsCommunicator(t *testing.T) {
	var raw interface{}
	raw = &Communicator{}
	if _, ok := raw.(packer.Communicator); !ok {
		t.Fatalf("comm must be a communicator")
	}
}

func TestStart(t *testing.T) {
	// This test hits an already running Windows VM
	// You can comment this line out temporarily during development
	t.Skip()

	comm, err := New(&winrm.Endpoint{"localhost", 5985}, "vagrant", "vagrant", time.Duration(1)*time.Minute)
	if err != nil {
		t.Fatalf("error connecting to WinRM: %s", err)
	}

	var cmd packer.RemoteCmd
	var outWriter, errWriter bytes.Buffer

	cmd.Command = "dir"
	cmd.Stdout = &outWriter
	cmd.Stderr = &errWriter

	err = comm.Start(&cmd)
	if err != nil {
		t.Fatalf("error starting cmd: %s", err)
	}
	cmd.Wait()

	fmt.Println(outWriter.String())
	fmt.Println(errWriter.String())

	if err != nil {
		t.Fatalf("error running cmd: %s", err)
	}

	if cmd.ExitStatus != 0 {
		t.Fatalf("exit status was non-zero: %d", cmd.ExitStatus)
	}
}

func TestStartElevated(t *testing.T) {
	// This test hits an already running Windows VM
	// You can comment this line out temporarily during development
	t.Skip()

	comm, err := New(&winrm.Endpoint{"localhost", 5985}, "vagrant", "vagrant", time.Duration(1)*time.Minute)
	if err != nil {
		t.Fatalf("error connecting to WinRM: %s", err)
	}

	var cmd packer.RemoteCmd
	var outWriter, errWriter bytes.Buffer

	cmd.Command = "dir"
	cmd.Stdout = &outWriter
	cmd.Stderr = &errWriter

	err = comm.StartElevated(&cmd)
	if err != nil {
		t.Fatalf("error starting cmd: %s", err)
	}
	cmd.Wait()

	fmt.Println(outWriter.String())
	fmt.Println(errWriter.String())

	if err != nil {
		t.Fatalf("error running cmd: %s", err)
	}

	if cmd.ExitStatus != 0 {
		t.Fatalf("exit status was non-zero: %d", cmd.ExitStatus)
	}
}

func TestUpload(t *testing.T) {
	// This test hits an already running Windows VM
	// You can comment this line out temporarily during development
	t.Skip()

	comm, err := New(&winrm.Endpoint{"localhost", 5985}, "vagrant", "vagrant", time.Duration(1)*time.Minute)
	if err != nil {
		t.Fatalf("error connecting to WinRM: %s", err)
	}

	f, err := os.Open("packer.jpg")
	if err != nil {
		t.Fatalf("error opening file: %s", err)
	}
	defer f.Close()

	err = comm.Upload("c:\\packer.jpg", f, nil)
	if err != nil {
		t.Fatalf("error uploading file: %s", err)
	}
}

func TestUploadDir(t *testing.T) {
	// This test hits an already running Windows VM
	// You can comment this line out temporarily during development
	t.Skip()

	comm, err := New(&winrm.Endpoint{"localhost", 5985}, "vagrant", "vagrant", time.Duration(1)*time.Minute)
	if err != nil {
		t.Fatalf("error connecting to WinRM: %s", err)
	}

	err = comm.UploadDir("c:\\src\\chef-repo", "~/src/chef-repo", nil)
	if err != nil {
		t.Fatalf("error uploading dir: %s", err)
	}
}

func TestISO8601DurationString(t *testing.T) {
	// Test complex duration with hours, minutes, seconds
	d := time.Duration(3701) * time.Second
	s := ISO8601DurationString(d)
	if s != "PT1H1M41S" {
		t.Fatalf("bad ISO 8601 duration string: %s", s)
	}

	// Test only minutes duration
	d = time.Duration(20) * time.Minute
	s = ISO8601DurationString(d)
	if s != "PT20M" {
		t.Fatalf("bad ISO 8601 duration string for 20M: %s", s)
	}

	// Test only seconds
	d = time.Duration(1) * time.Second
	s = ISO8601DurationString(d)
	if s != "PT1S" {
		t.Fatalf("bad ISO 8601 duration string for 1S: %s", s)
	}

	// Test negative duration (unsupported)
	d = time.Duration(-1) * time.Second
	s = ISO8601DurationString(d)
	if s != "PT0S" {
		t.Fatalf("bad ISO 8601 duration string for negative: %s", s)
	}
}

func defaultCommunicator() *Communicator {
	comm, err := New(&winrm.Endpoint{"localhost", 5985}, "vagrant", "vagrant", time.Duration(1)*time.Minute)
	if err != nil {
		fmt.Sprintf("Could not create Communicator: %s", err)
	}
	return comm
}
