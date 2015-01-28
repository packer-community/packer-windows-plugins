package common

import (
	"fmt"
	"log"

	"github.com/mitchellh/multistep"
	wincommon "github.com/packer-community/packer-windows-plugins/common"
)

func WinRMAddressFunc(config wincommon.WinRMConfig) func(state multistep.StateBag) (string, error) {
	if config.WinRMHost == "" {
		log.Printf("No WinRM Host provided, using default host 127.0.0.1")
		config.WinRMHost = "127.0.0.1"
	}

	log.Printf("Have address from config: %s:%d", config.WinRMHost, config.WinRMPort)

	return func(state multistep.StateBag) (string, error) {
		winrmPort := config.WinRMPort
		if forwardedPort, ok := state.GetOk("winrmHostPort"); ok {
			winrmPort = forwardedPort.(uint)
		}

		return fmt.Sprintf("%s:%d", config.WinRMHost, winrmPort), nil
	}
}

// Creates a generic WinRM connect step from a Parallels builder config
func NewConnectStep(winrmConfig wincommon.WinRMConfig) multistep.Step {
	return &wincommon.StepConnectWinRM{
		WinRMAddress:     WinRMAddressFunc(winrmConfig),
		WinRMUser:        winrmConfig.WinRMUser,
		WinRMPassword:    winrmConfig.WinRMPassword,
		WinRMWaitTimeout: winrmConfig.WinRMWaitTimeout,
	}
}
