package common

import (
	"fmt"

	"github.com/mitchellh/multistep"
	wincommon "github.com/packer-community/packer-windows-plugins/common"
)

func WinRMAddressFunc(config *wincommon.WinRMConfig, driver Driver) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		if config.WinRMHost != "" {
			return fmt.Sprintf("%s:%d", config.WinRMHost, config.WinRMPort), nil
		}

		ipAddress, err := driver.GuestAddress(state)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%s:%d", ipAddress, config.WinRMPort), nil
	}
}
