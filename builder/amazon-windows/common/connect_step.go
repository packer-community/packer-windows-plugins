package common

import (
	"fmt"
	"github.com/mitchellh/goamz/ec2"
	"github.com/mitchellh/multistep"
	//"github.com/mitchellh/packer/common"
	"errors"
	wincommon "github.com/packer-community/packer-windows-plugins/common"
	"log"
	"time"
)

// Returns an Endpoint suitable for the WinRM communicator
func WinRMAddress(e *ec2.EC2, port uint, private bool) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		for j := 0; j < 2; j++ {
			var host string
			i := state.Get("instance").(*ec2.Instance)
			if i.DNSName != "" {
				host = i.DNSName
			} else if i.VpcId != "" {
				if i.PublicIpAddress != "" && !private {
					host = i.PublicIpAddress
				} else {
					host = i.PrivateIpAddress
				}
			}

			if host != "" {
				log.Printf("Configured remote WinRM address to be %s:%d", host, port)
				return fmt.Sprintf("%s:%d", host, port), nil
			}

			r, err := e.Instances([]string{i.InstanceId}, ec2.NewFilter())
			if err != nil {
				return "", err
			}

			if len(r.Reservations) == 0 || len(r.Reservations[0].Instances) == 0 {
				return "", fmt.Errorf("instance not found: %s", i.InstanceId)
			}

			state.Put("instance", &r.Reservations[0].Instances[0])
			time.Sleep(1 * time.Second)
		}

		return "", errors.New("couldn't determine IP address for instance")
	}
}

// Creates a WinRM connect step for an EC2 instance
func NewConnectStep(ec2 *ec2.EC2, private bool, winrmConfig *wincommon.WinRMConfig) multistep.Step {
	return &wincommon.StepConnectWinRM{
		WinRMAddress:     WinRMAddress(ec2, winrmConfig.WinRMPort, private),
		WinRMConfig:      winrmConfig,
		WinRMWaitTimeout: winrmConfig.WinRMWaitTimeout,
	}
}
