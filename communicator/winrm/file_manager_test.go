package winrm

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

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

func testWinFriendlyPath(t *testing.T) {
	in := "/foo/bar/baz"
	out := winFriendlyPath(in)
	if out != "\\foo\\bar\\baz" {
		t.Fatalf("Path should be %s", out)
	}
}

func TestPrepareFileDirectory(t *testing.T) {
	comm := new(MockWinRMCommunicator)
	fm, err := NewFileManager(comm)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}
	comm.expectedCommand = `
$dest_file_path = [System.IO.Path]::GetFullPath("/foo")
if (-not (Test-Path $dest_file_path) ) {
  rm $dest_file_path
  Write-Output "Creating directory: $dest_file_path"
  md $dest_file_path -Force
}`
	err = fm.prepareFileDirectory("/foo")
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}

}
func TestUploadDir(t *testing.T) {
	comm := new(MockWinRMCommunicator)
	comm.expectedCommand = "ppowershell Invoke-WebRequest 'http://10.0.2.2:8081/tmp' -OutFile c:\\windows\\temp\\tmp"

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))

	fm, err := NewFileManager(comm)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}
	err = fm.UploadDir("c:\\windows\\temp", dir)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}

	// Error out the dir upload
	var called bool
	uploadDir = func(f *fileManager, dst string, src string) error {
		called = true
		return errors.New("Upload failed for test purposes")
	}
	err = fm.UploadDir("c:\\windows\\temp", dir)
	if called == false {
		t.Fatalf("Expected uploadDir to have been called")
	}
	if err == nil {
		t.Fatalf("Should have error")
	}
}

// Mock Default WinRM Communicator - does nothing at all
type MockWinRMCommunicator struct {
	hostUploadDir  string
	guestUploadDir string

	// Set this expectation before calling Start
	//
	expectedCommand string
}

func (c *MockWinRMCommunicator) StartElevated(cmd *packer.RemoteCmd) (err error) { return nil }
func (c *MockWinRMCommunicator) Start(cmd *packer.RemoteCmd) (err error) {
	log.Printf("Starting remote command: %s", cmd.Command)
	if cmd.Command != c.expectedCommand {
		return errors.New(fmt.Sprintf("Expected command to be '%s' but got '%s'", c.expectedCommand, cmd.Command))
	}
	return nil
}
func (c *MockWinRMCommunicator) runCommand(commandText string, cmd *packer.RemoteCmd) (err error) {
	log.Printf("Running command: %s", commandText)
	return nil
}
func (c *MockWinRMCommunicator) Upload(string, io.Reader, *os.FileInfo) error             { return nil }
func (c *MockWinRMCommunicator) UploadDir(dst string, src string, exclude []string) error { return nil }
func (c *MockWinRMCommunicator) Download(string, io.Writer) error                         { return nil }
