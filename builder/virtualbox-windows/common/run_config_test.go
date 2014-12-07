package common

import (
	"testing"
)

func TestRunConfigPrepare_BootWait(t *testing.T) {
	var c *RunConfig
	var errs []error

	// Test a default boot_wait
	c = new(RunConfig)
	errs = c.Prepare(testConfigTemplate(t))
	if len(errs) > 0 {
		t.Fatalf("should not have error: %s", errs)
	}

	if c.RawBootWait != "10s" {
		t.Fatalf("bad value: %s", c.RawBootWait)
	}

	// Test with a bad boot_wait
	c = new(RunConfig)
	c.RawBootWait = "this is not good"
	errs = c.Prepare(testConfigTemplate(t))
	if len(errs) == 0 {
		t.Fatalf("bad: %#v", errs)
	}

	// Test with a good one
	c = new(RunConfig)
	c.RawBootWait = "5s"
	errs = c.Prepare(testConfigTemplate(t))
	if len(errs) > 0 {
		t.Fatalf("should not have error: %s", errs)
	}
}

func TestRunConfigPrepare_WinRMHostPortMin(t *testing.T) {
	var c *RunConfig
	var errs []error

	c = new(RunConfig)
	errs = c.Prepare(testConfigTemplate(t))
	if len(errs) > 0 {
		t.Fatalf("should not have error: %s", errs)
	}
	if c.WinRMHostPortMax != 6985 {
		t.Fatalf("Default max port should be 6985: %#v", errs)
	}
	if c.WinRMHostPortMin != 4985 {
		t.Fatalf("Default min port should be 4985: %#v", errs)
	}

	c = new(RunConfig)
	c.WinRMHostPortMax = 10
	c.WinRMHostPortMin = 10

	errs = c.Prepare(testConfigTemplate(t))
	if len(errs) == 0 {
		t.Fatalf("Should have error - range is 0: %#v", errs)
	}

	c = new(RunConfig)
	c.WinRMHostPortMax = 1
	c.WinRMHostPortMin = 10

	errs = c.Prepare(testConfigTemplate(t))
	if len(errs) == 0 {
		t.Fatalf("Should have error - min is greater than max: %#v", errs)
	}
}
