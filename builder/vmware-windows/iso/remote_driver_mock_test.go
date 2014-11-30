package iso

import (
	"testing"

	vmwcommon "github.com/packer-community/packer-windows-plugins/builder/vmware-windows/common"
)

func TestRemoteDriverMock_impl(t *testing.T) {
	var _ vmwcommon.Driver = new(RemoteDriverMock)
	var _ RemoteDriver = new(RemoteDriverMock)
}
