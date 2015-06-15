package common

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/mitchellh/packer/template/interpolate"
)

type WinRMConfig struct {
	WinRMUser           string `mapstructure:"winrm_username"`
	WinRMPassword       string `mapstructure:"winrm_password"`
	WinRMHost           string `mapstructure:"winrm_host"`
	WinRMPort           uint   `mapstructure:"winrm_port"`
	RawWinRMWaitTimeout string `mapstructure:"winrm_wait_timeout"`

	WinRMWaitTimeout time.Duration
}

func (c *WinRMConfig) Prepare(ctx *interpolate.Context) []error {
	if c.WinRMPort == 0 {
		c.WinRMPort = 5985
	}

	if c.RawWinRMWaitTimeout == "" {
		c.RawWinRMWaitTimeout = "20m"
	}

	var errs []error
	if c.WinRMHost != "" {
		if ip := net.ParseIP(c.WinRMHost); ip == nil {
			if _, err := net.LookupHost(c.WinRMHost); err != nil {
				errs = append(errs, errors.New("winrm_host is an invalid IP or hostname"))
			}
		}
	}

	if c.WinRMUser == "" {
		errs = append(errs, errors.New("winrm_username must be specified."))
	}

	var err error
	c.WinRMWaitTimeout, err = time.ParseDuration(c.RawWinRMWaitTimeout)
	if err != nil {
		errs = append(errs, fmt.Errorf("Failed parsing winrm_wait_timeout: %s", err))
	}

	if c.WinRMWaitTimeout <= 0*time.Second {
		errs = append(errs, fmt.Errorf("Failed parsing winrm_wait_timeout: %s, raw timeout: %s", c.WinRMWaitTimeout, c.RawWinRMWaitTimeout))
	}

	return errs
}
