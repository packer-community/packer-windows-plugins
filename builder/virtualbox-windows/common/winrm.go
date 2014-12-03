package common

// func WinRMAddress(state multistep.StateBag) (string, error) {
// 	sshHostPort := state.Get("sshHostPort").(uint)
// 	return fmt.Sprintf("127.0.0.1:%d", sshHostPort), nil
// }

/*

import (
	"fmt"

	"github.com/mitchellh/multistep"
	vboxcommon "github.com/mitchellh/packer/builder/virtualbox/common"
	common "github.com/packer-community/packer-windows-plugins/common"
)

func WinRMAddressFunc(config *common.WinRMConfig, driver vboxcommon.Driver) func(multistep.StateBag) (string, error) {
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
*/
