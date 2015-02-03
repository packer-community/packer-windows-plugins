package common

import (
	"errors"
	"testing"

	"github.com/mitchellh/multistep"
	parallelscommon "github.com/mitchellh/packer/builder/parallels/common"
	wincommon "github.com/packer-community/packer-windows-plugins/common"
)

func TestWinRMAddressFunc_UsesPortForwardingFail(t *testing.T) {
	config := wincommon.WinRMConfig{
		WinRMHost: "localhost",
		WinRMPort: 456,
	}

	state := new(multistep.BasicStateBag)
	state.Put("winrmHostPort", uint(123))
	state.Put("driver", &parallelscommon.DriverMock{IpAddressError: errors.New("Invalid machine state"), MacReturn: "01cd123"})
	state.Put("vmName", "myvmname")

	f := WinRMAddressFunc(config)
	_, err := f(state)

	if err == nil {
		t.Fatalf("Expected err %v", err)
	}
}

func TestWinRMAddressFunc_UsesPortForwarding(t *testing.T) {
	config := wincommon.WinRMConfig{
		WinRMHost: "localhost",
		WinRMPort: 456,
	}

	state := new(multistep.BasicStateBag)
	state.Put("winrmHostPort", uint(123))
	state.Put("driver", &parallelscommon.DriverMock{IpAddressReturn: "172.17.4.13", MacReturn: "01cd123"})
	state.Put("vmName", "myvmname")

	f := WinRMAddressFunc(config)
	address, err := f(state)

	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if address != "172.17.4.13:123" {
		t.Errorf("should have forwarded to port 123, but was %s", address)
	}
}
