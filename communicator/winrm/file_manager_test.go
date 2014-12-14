package winrm

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
	//	"time"

	//	"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/packer/packer"
)

func TestTempFile(t *testing.T) {
	comm := defaultCommunicator()
	fm := &fileManager{comm: comm}
	tempString := "Temp for packer"
	var output *os.File
	var input *os.File
	defer func() {
		// Close and delete tmp files
		input.Close()
		output.Close()
		os.Remove(input.Name())
		os.Remove(output.Name())
	}()

	input, err := ioutil.TempFile("/tmp", "packer-test-tmp")
	fmt.Printf("Input name: %s", input.Name())
	input.WriteString(tempString)
	if err != nil {
		t.Fatalf("Unable to create tmp file for test: %s", err)
	}
	f, err := os.Open(input.Name())
	output, err = fm.TempFile(f)
	fmt.Printf("Output name: %s", output.Name())

	if err != nil {
		t.Fatalf("Unable to create tmp file for test: %s", err)
	}

	data, err := ioutil.ReadFile(output.Name())
	dataString := string(data[0:15])
	if dataString != tempString {
		t.Fatalf("File contents should equal \"%s\". Actual: \"%s\"", tempString, dataString)
	}
}

func TestPrepareFileDirectory(t *testing.T) {
	//t.Skip()
	comm := new(MockWinRMCommunicator)
	comm.runCommand("foo", nil)

	fm := &fileManager{comm: comm}
	fm.UploadDir("c:\\windows\\temp", "/tmp")
}

type MockWinRMCommunicator struct {
	hostUploadDir  string
	guestUploadDir string
}

func (c *MockWinRMCommunicator) StartElevated(cmd *packer.RemoteCmd) (err error) {

	return nil
}

func (c *MockWinRMCommunicator) Start(cmd *packer.RemoteCmd) (err error) {
	return nil
}

func (c *MockWinRMCommunicator) runCommand(commandText string, cmd *packer.RemoteCmd) (err error) {
	log.Printf("Running command: %s", commandText)
	return nil
}

func (c *MockWinRMCommunicator) Upload(string, io.Reader, *os.FileInfo) error {
	return nil
}

// UploadDir uploads the contents of a directory recursively to
// the remote path. It also takes an optional slice of paths to
// ignore when uploading.
//
// The folder name of the source folder should be created unless there
// is a trailing slash on the source "/". For example: "/tmp/src" as
// the source will create a "src" directory in the destination unless
// a trailing slash is added. This is identical behavior to rsync(1).
func (c *MockWinRMCommunicator) UploadDir(dst string, src string, exclude []string) error {
	return nil
}

// Download downloads a file from the machine from the given remote path
// with the contents writing to the given writer. This method will
// block until it completes.
func (c *MockWinRMCommunicator) Download(string, io.Writer) error {
	return nil
}
