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

func WinRMAddressFunc(config *wincommon.WinRMConfig) func(state multistep.StateBag) (string, error) {
	if config.WinRMHost == "" {
		log.Printf("No WinRM Host provided, using default host 127.0.0.1")
		config.WinRMHost = "127.0.0.1"
	}
	log.Printf("Have address from config: %s:%d", config.WinRMHost, config.WinRMPort)
	return func(state multistep.StateBag) (string, error) {
		log.Printf("Returning address from config: %s:%d", config.WinRMHost, config.WinRMPort)
		return fmt.Sprintf("%s:%d", config.WinRMHost, config.WinRMPort), nil
	}
}

// Returns an Endpoint suitable for the WinRM communicator
func WinRMAddress(e *ec2.EC2, port int, private bool) func(multistep.StateBag) (string, error) {
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

// Creates a generic SSH or WinRM connect step from a VMWare builder config
func NewConnectStep(ec2 *ec2.EC2, port int, private bool, winrmConfig wincommon.WinRMConfig) multistep.Step {
	return &wincommon.StepConnectWinRM{
		WinRMAddress:     WinRMAddress(ec2, port, private),
		WinRMUser:        winrmConfig.WinRMUser,
		WinRMPassword:    winrmConfig.WinRMPassword,
		WinRMWaitTimeout: winrmConfig.WinRMWaitTimeout,
	}
}
