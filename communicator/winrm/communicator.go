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
	timeout  time.Duration
}

type elevatedShellOptions struct {
	Command  string
	User     string
	Password string
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
		timeout:  timeout,
	}, nil
}

func (c *Communicator) Start(rc *packer.RemoteCmd) (err error) {
	return c.runCommand(rc, rc.Command)
}

func (c *Communicator) StartElevated(cmd *packer.RemoteCmd) (err error) {
	panic("not implemented")
}

func (c *Communicator) runCommand(rc *packer.RemoteCmd, command string, arguments ...string) (err error) {
	log.Printf("starting remote command: %s", rc.Command)

	// Create a new shell process on the guest
	params := winrm.DefaultParameters()
	params.Timeout = iso8601.FormatDuration(time.Minute * 120)
	client := winrm.NewClientWithParameters(c.endpoint, c.user, c.password, params)
	shell, err := client.CreateShell()
	if err != nil {
		return err
	}

	cmd, err := shell.Execute(command, arguments...)
	if err != nil {
		return err
	}

	go func(shell *winrm.Shell, cmd *winrm.Command, rc *packer.RemoteCmd) {
		defer shell.Close()

		go io.Copy(rc.Stdout, cmd.Stdout)
		go io.Copy(rc.Stderr, cmd.Stderr)

		cmd.Wait()
		rc.SetExited(cmd.ExitCode())
	}(shell, cmd, rc)

	return nil
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
		MaxOperationsPerShell: 15, // lowest common denominator
	})
}
