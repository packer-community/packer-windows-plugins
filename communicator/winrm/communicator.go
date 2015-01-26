package winrm

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/dylanmei/iso8601"
	"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/packer/packer"
	"github.com/packer-community/winrmcp/winrmcp"
)

type Communicator struct {
	client   *winrm.Client
	endpoint *winrm.Endpoint
	user     string
	password string
}

// Creates a new packer.Communicator implementation over WinRM.
// Called when Packer tries to connect to WinRM
func New(endpoint *winrm.Endpoint, user string, password string, timeout time.Duration) (*Communicator, error) {
	// Create the WinRM client we use internally
	params := winrm.DefaultParameters()
	params.Timeout = iso8601.FormatDuration(timeout)
	client := winrm.NewClientWithParameters(endpoint, user, password, params)

	// Attempt to connect to the WinRM service
	shell, err := client.CreateShell()
	if err != nil {
		return nil, err
	}

	err = shell.Close()
	if err != nil {
		return nil, err
	}

	return &Communicator{
		endpoint: endpoint,
		user:     user,
		password: password,
	}, nil
}

func (c *Communicator) Start(rc *packer.RemoteCmd) error {
	log.Printf("starting remote command: %s", rc.Command)

	// Create a new shell process on the guest
	client := winrm.NewClient(c.endpoint, c.user, c.password)
	shell, err := client.CreateShell()
	if err != nil {
		return err
	}

	cmd, err := shell.Execute(rc.Command)
	if err != nil {
		return err
	}

	go runCommand(shell, cmd, rc)
	return nil
}

func runCommand(shell *winrm.Shell, cmd *winrm.Command, rc *packer.RemoteCmd) {
	defer shell.Close()

	go io.Copy(rc.Stdout, cmd.Stdout)
	go io.Copy(rc.Stderr, cmd.Stderr)

	cmd.Wait()
	rc.SetExited(cmd.ExitCode())
}

func (c *Communicator) Upload(dst string, input io.Reader, ignored *os.FileInfo) error {
	wcp, err := c.newCopyClient()
	if err != nil {
		return err
	}
	return wcp.Write(dst, input)
}

func (c *Communicator) UploadDir(dst string, src string, TODO []string) error {
	wcp, err := c.newCopyClient()
	if err != nil {
		return err
	}
	return wcp.Copy(src, dst)
}

func (c *Communicator) Download(string, io.Writer) error {
	panic("Download not implemented yet")
}

func (c *Communicator) newCopyClient() (*winrmcp.Winrmcp, error) {
	addr := fmt.Sprintf("%s:%d", c.endpoint.Host, c.endpoint.Port)
	return winrmcp.New(addr, &winrmcp.Config{
		Auth: winrmcp.Auth{
			User:     c.user,
			Password: c.password,
		},
		OperationTimeout:      time.Minute * 5,
		MaxOperationsPerShell: 15, // lowest common denominator
	})
}
