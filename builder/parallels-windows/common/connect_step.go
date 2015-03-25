package common

import (
	"fmt"
	"log"

	"github.com/mitchellh/multistep"
	parallelscommon "github.com/mitchellh/packer/builder/parallels/common"
	wincommon "github.com/packer-community/packer-windows-plugins/common"
)

func WinRMAddressFunc(config *wincommon.WinRMConfig) func(state multistep.StateBag) (string, error) {

	return func(state multistep.StateBag) (string, error) {
		log.Printf("Determining WinRM remote IP address...")
		vmName := state.Get("vmName").(string)
		driver := state.Get("driver").(parallelscommon.Driver)

		mac, err := driver.Mac(vmName)
		if err != nil {
			return "", err
		}

		ip, err := driver.IpAddress(mac)
		if err != nil {
			return "", err
		}

		winrmPort := config.WinRMPort
		if forwardedPort, ok := state.GetOk("winrmHostPort"); ok {
			winrmPort = forwardedPort.(uint)
		}

		log.Printf("Detected WinRM address to be: %s:%d", ip, winrmPort)
		return fmt.Sprintf("%s:%d", ip, winrmPort), nil
	}
}

// Creates a generic WinRM connect step from a Parallels builder config
func NewConnectStep(winrmConfig *wincommon.WinRMConfig) multistep.Step {
	return &wincommon.StepConnectWinRM{
		WinRMAddress:     WinRMAddressFunc(winrmConfig),
		WinRMConfig:      winrmConfig,
		WinRMWaitTimeout: winrmConfig.WinRMWaitTimeout,
	}
}
