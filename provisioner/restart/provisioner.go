package restart

import (
	"fmt"
	//"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/packer/common"
	"github.com/mitchellh/packer/packer"
	"log"
	"os"
	"time"
)

var DefaultRestartCommand = "shutdown /r /c \"packer restart\" /t 5 && net stop winrm"

var retryableSleep = 2 * time.Second

type config struct {
	common.PackerConfig `mapstructure:",squash"`

	// The command used to execute the script. The '{{ .Path }}' variable
	// should be used to specify where the script goes, {{ .Vars }}
	// can be used to inject the environment_vars into the environment.
	RestartCommand string `mapstructure:"restart_command"`

	// The timeout for retrying to start the process. Until this timeout
	// is reached, if the provisioner can't start a process, it retries.
	// This can be set high to allow for reboots.
	RawStartRetryTimeout string `mapstructure:"start_retry_timeout"`
	startRetryTimeout    time.Duration
	tpl                  *packer.ConfigTemplate
}

type Provisioner struct {
	config config
}

func (p *Provisioner) Prepare(raws ...interface{}) error {
	md, err := common.DecodeConfig(&p.config, raws...)
	if err != nil {
		return err
	}

	p.config.tpl, err = packer.NewConfigTemplate()
	if err != nil {
		return err
	}
	p.config.tpl.UserVars = p.config.PackerUserVars

	// Accumulate any errors
	errs := common.CheckUnusedConfig(md)

	if p.config.RestartCommand == "" {
		p.config.RestartCommand = DefaultRestartCommand
	}

	if p.config.RawStartRetryTimeout == "" {
		p.config.RawStartRetryTimeout = "5m"
	}

	if p.config.RawStartRetryTimeout != "" {
		p.config.startRetryTimeout, err = time.ParseDuration(p.config.RawStartRetryTimeout)
		if err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("Failed parsing start_retry_timeout: %s", err))
		}
	}

	if errs != nil && len(errs.Errors) > 0 {
		return errs
	}

	return nil
}

func (p *Provisioner) Provision(ui packer.Ui, comm packer.Communicator) error {
	ui.Say("Restarting Windows Machine")

	var cmd *packer.RemoteCmd
	command := DefaultRestartCommand
	err := p.retryable(func() error {
		cmd = &packer.RemoteCmd{Command: command}
		return cmd.StartWithUi(comm, ui)
	})

	if err != nil {
		return err
	}

	if cmd.ExitStatus != 0 {
		return fmt.Errorf("Restart script exited with non-zero exit status: %d", cmd.ExitStatus)
	}

	return nil
}

func (p *Provisioner) Cancel() {
	os.Exit(0)
}

// retryable will retry the given function over and over until a
// non-error is returned.
func (p *Provisioner) retryable(f func() error) error {
	startTimeout := time.After(p.config.startRetryTimeout)
	for {
		var err error
		if err = f(); err == nil {
			return nil
		}

		// Create an error and log it
		err = fmt.Errorf("Retryable error: %s", err)
		log.Printf(err.Error())

		// Check if we timed out, otherwise we retry. It is safe to
		// retry since the only error case above is if the command
		// failed to START.
		select {
		case <-startTimeout:
			return err
		default:
			time.Sleep(retryableSleep)
		}
	}
}
