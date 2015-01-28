package common

import (
	"testing"

	"github.com/mitchellh/multistep"
	wincommon "github.com/packer-community/packer-windows-plugins/common"
)

func TestWinRMAddressFunc_UsesPortForwarding(t *testing.T) {
	config := wincommon.WinRMConfig{
		WinRMHost: "localhost",
		WinRMPort: 456,
	}

	state := new(multistep.BasicStateBag)
	state.Put("winrmHostPort", uint(123))

	f := WinRMAddressFunc(config)
	address, err := f(state)

	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if address != "localhost:123" {
		t.Errorf("should have forwarded to port 123, but was %s", address)
	}
}
