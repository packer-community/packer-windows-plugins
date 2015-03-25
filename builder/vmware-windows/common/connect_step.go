package common

import (
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/common"
	wincommon "github.com/packer-community/packer-windows-plugins/common"
)

// Creates a generic SSH or WinRM connect step from a VMWare builder config
func NewConnectStep(communicatorType string, driver Driver, sshConfig *SSHConfig, winrmConfig *wincommon.WinRMConfig) multistep.Step {
	//if communicatorType == packer.WinRMCommunicatorType {
	if communicatorType == "winrm" {
		return &wincommon.StepConnectWinRM{
			WinRMAddress:     WinRMAddressFunc(winrmConfig, driver),
			WinRMConfig:      winrmConfig,
			WinRMWaitTimeout: winrmConfig.WinRMWaitTimeout,
		}
	} else {
		return &common.StepConnectSSH{
			SSHAddress:     SSHAddressFunc(sshConfig, driver),
			SSHConfig:      SSHConfigFunc(sshConfig),
			SSHWaitTimeout: sshConfig.SSHWaitTimeout,
			NoPty:          sshConfig.SSHSkipRequestPty,
		}
	}
}
