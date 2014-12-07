package common

import (
	"errors"
	"fmt"
	"time"

	"github.com/mitchellh/packer/packer"
)

type RunConfig struct {
	Headless    bool   `mapstructure:"headless"`
	RawBootWait string `mapstructure:"boot_wait"`

	HTTPDir          string `mapstructure:"http_directory"`
	HTTPPortMin      uint   `mapstructure:"http_port_min"`
	HTTPPortMax      uint   `mapstructure:"http_port_max"`
	WinRMHostPortMin uint   `mapstructure:"winrm_host_port_min"`
	WinRMHostPortMax uint   `mapstructure:"winrm_host_port_max"`

	BootWait time.Duration ``
}

func (c *RunConfig) Prepare(t *packer.ConfigTemplate) []error {
	if c.RawBootWait == "" {
		c.RawBootWait = "10s"
	}

	if c.HTTPPortMin == 0 {
		c.HTTPPortMin = 8000
	}

	if c.HTTPPortMax == 0 {
		c.HTTPPortMax = 9000
	}

	if c.WinRMHostPortMin == 0 {
		c.WinRMHostPortMin = 4985
	}

	if c.WinRMHostPortMax == 0 {
		c.WinRMHostPortMax = 6985
	}

	templates := map[string]*string{
		"boot_wait":      &c.RawBootWait,
		"http_directory": &c.HTTPDir,
	}

	errs := make([]error, 0)
	for n, ptr := range templates {
		var err error
		*ptr, err = t.Process(*ptr, nil)
		if err != nil {
			errs = append(errs, fmt.Errorf("Error processing %s: %s", n, err))
		}
	}

	var err error
	c.BootWait, err = time.ParseDuration(c.RawBootWait)

	if err != nil {
		errs = append(errs, fmt.Errorf("Failed parsing boot_wait: %s", err))
	}

	if c.HTTPPortMin == c.HTTPPortMax {
		errs = append(errs,
			errors.New("http_port_max must be greater than http_port_min"))
	}

	if c.HTTPPortMin > c.HTTPPortMax {
		errs = append(errs,
			errors.New("http_port_min must be less than http_port_max"))
	}

	if c.HTTPPortMin > c.HTTPPortMin {
		errs = append(errs, errors.New("http_port_min must be less than http_port_max"))
	}

	if c.WinRMHostPortMin == c.WinRMHostPortMax {
		errs = append(errs,
			errors.New("winrm_host_port_max must be greater than winrm_host_port_min"))

	}
	if c.WinRMHostPortMin > c.WinRMHostPortMax {
		errs = append(errs,
			errors.New("winrm_host_port_min must be less than winrm_host_port_max"))
	}

	if c.WinRMHostPortMin > c.WinRMHostPortMin {
		errs = append(errs, errors.New("winrm_host_port_min must be less than winrm_host_port_max"))
	}

	return errs
}
