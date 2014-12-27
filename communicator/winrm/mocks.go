package winrm

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	//	"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/packer/packer"
)

type MockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *MockFileInfo) ModTime() time.Time { return m.modTime }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() interface{}   { return nil }

//
// Communicator Mocks to test behaviours
//

// Mock Default WinRM Communicator - throws errors when remote commands called
type MockWinRMCommunicatorWithErrors struct {
	hostUploadDir  string
	guestUploadDir string
}

func (c *MockWinRMCommunicatorWithErrors) StartElevated(cmd *packer.RemoteCmd) (err error) { return nil }
func (c *MockWinRMCommunicatorWithErrors) Start(cmd *packer.RemoteCmd) (err error) {
	log.Printf("Starting remote command: %s", cmd.Command)
	cmd.ExitStatus = 1
	cmd.SetExited(1)
	return errors.New("Remote command failed for test purposes")
}
func (c *MockWinRMCommunicatorWithErrors) runCommand(commandText string, cmd *packer.RemoteCmd) (err error) {
	log.Printf("Running command: %s", commandText)
	cmd.ExitStatus = 1
	cmd.SetExited(1)
	return errors.New("Remote command failed for test purposes")
}
func (c *MockWinRMCommunicatorWithErrors) Upload(string, io.Reader, *os.FileInfo) error { return nil }
func (c *MockWinRMCommunicatorWithErrors) UploadDir(dst string, src string, exclude []string) error {
	return nil
}
func (c *MockWinRMCommunicatorWithErrors) Download(string, io.Writer) error { return nil }

// Mock Default WinRM Communicator - does nothing at all
type MockWinRMCommunicator struct {
	hostUploadDir  string
	guestUploadDir string
	// Set this expectation before calling Start or (re)set to ""
	// if you don't want to check anything
	expectedCommand string
}

func (c *MockWinRMCommunicator) StartElevated(cmd *packer.RemoteCmd) (err error) { return nil }
func (c *MockWinRMCommunicator) Start(cmd *packer.RemoteCmd) (err error) {
	log.Printf("Starting remote command: %s", cmd.Command)
	if c.expectedCommand != "" && cmd.Command != c.expectedCommand {
		cmd.ExitStatus = 1
		cmd.SetExited(1)
		return errors.New(fmt.Sprintf("Expected command to be '%s' but got '%s'", c.expectedCommand, cmd.Command))
	}
	cmd.SetExited(0)
	return nil
}
func (c *MockWinRMCommunicator) runCommand(commandText string, cmd *packer.RemoteCmd) (err error) {
	log.Printf("Running command: %s", commandText)
	return nil
}
func (c *MockWinRMCommunicator) Upload(string, io.Reader, *os.FileInfo) error             { return nil }
func (c *MockWinRMCommunicator) UploadDir(dst string, src string, exclude []string) error { return nil }
func (c *MockWinRMCommunicator) Download(string, io.Writer) error                         { return nil }
