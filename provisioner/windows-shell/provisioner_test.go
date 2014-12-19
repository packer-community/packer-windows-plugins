package shell

import (
	"bytes"
	"github.com/mitchellh/packer/packer"
	"io/ioutil"
	"os"
	"testing"
)

func testConfig() map[string]interface{} {
	return map[string]interface{}{
		"inline": []interface{}{"foo", "bar"},
	}
}

func TestProvisioner_Impl(t *testing.T) {
	var raw interface{}
	raw = &Provisioner{}
	if _, ok := raw.(packer.Provisioner); !ok {
		t.Fatalf("must be a Provisioner")
	}
}

func TestProvisionerPrepare_Defaults(t *testing.T) {
	var p Provisioner
	config := testConfig()

	err := p.Prepare(config)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if p.config.RemotePath != DefaultRemotePath {
		t.Errorf("unexpected remote path: %s", p.config.RemotePath)
	}

	if p.config.ExecuteCommand != "powershell \"& { {{.Vars}}{{.Path}} }\"" {
		t.Fatalf("Default command should be powershell \"& { {{.Vars}}{{.Path}} }\", but got %s", p.config.ExecuteCommand)
	}
}

func TestProvisionerPrepare_Config(t *testing.T) {

}

func TestProvisionerPrepare_InvalidKey(t *testing.T) {
	var p Provisioner
	config := testConfig()

	// Add a random key
	config["i_should_not_be_valid"] = true
	err := p.Prepare(config)
	if err == nil {
		t.Fatal("should have error")
	}
}

func TestProvisionerPrepare_Script(t *testing.T) {
	config := testConfig()
	delete(config, "inline")

	config["script"] = "/this/should/not/exist"
	p := new(Provisioner)
	err := p.Prepare(config)
	if err == nil {
		t.Fatal("should have error")
	}

	// Test with a good one
	tf, err := ioutil.TempFile("", "packer")
	if err != nil {
		t.Fatalf("error tempfile: %s", err)
	}
	defer os.Remove(tf.Name())

	config["script"] = tf.Name()
	p = new(Provisioner)
	err = p.Prepare(config)
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}

func TestProvisionerPrepare_ScriptAndInline(t *testing.T) {
	var p Provisioner
	config := testConfig()

	delete(config, "inline")
	delete(config, "script")
	err := p.Prepare(config)
	if err == nil {
		t.Fatal("should have error")
	}

	// Test with both
	tf, err := ioutil.TempFile("", "packer")
	if err != nil {
		t.Fatalf("error tempfile: %s", err)
	}
	defer os.Remove(tf.Name())

	config["inline"] = []interface{}{"foo"}
	config["script"] = tf.Name()
	err = p.Prepare(config)
	if err == nil {
		t.Fatal("should have error")
	}
}

func TestProvisionerPrepare_ScriptAndScripts(t *testing.T) {
	var p Provisioner
	config := testConfig()

	// Test with both
	tf, err := ioutil.TempFile("", "packer")
	if err != nil {
		t.Fatalf("error tempfile: %s", err)
	}
	defer os.Remove(tf.Name())

	config["inline"] = []interface{}{"foo"}
	config["scripts"] = []string{tf.Name()}
	err = p.Prepare(config)
	if err == nil {
		t.Fatal("should have error")
	}
}

func TestProvisionerPrepare_Scripts(t *testing.T) {
	config := testConfig()
	delete(config, "inline")

	config["scripts"] = []string{}
	p := new(Provisioner)
	err := p.Prepare(config)
	if err == nil {
		t.Fatal("should have error")
	}

	// Test with a good one
	tf, err := ioutil.TempFile("", "packer")
	if err != nil {
		t.Fatalf("error tempfile: %s", err)
	}
	defer os.Remove(tf.Name())

	config["scripts"] = []string{tf.Name()}
	p = new(Provisioner)
	err = p.Prepare(config)
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}

func TestProvisionerPrepare_EnvironmentVars(t *testing.T) {
	config := testConfig()

	// Test with a bad case
	config["environment_vars"] = []string{"badvar", "good=var"}
	p := new(Provisioner)
	err := p.Prepare(config)
	if err == nil {
		t.Fatal("should have error")
	}

	// Test with a trickier case
	config["environment_vars"] = []string{"=bad"}
	p = new(Provisioner)
	err = p.Prepare(config)
	if err == nil {
		t.Fatal("should have error")
	}

	// Test with a good case
	// Note: baz= is a real env variable, just empty
	config["environment_vars"] = []string{"FOO=bar", "baz="}
	p = new(Provisioner)
	err = p.Prepare(config)
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}

func TestProvisionerQuote_EnvironmentVars(t *testing.T) {
	config := testConfig()

	config["environment_vars"] = []string{"keyone=valueone", "keytwo=value\ntwo", "keythree='valuethree'", "keyfour='value\nfour'"}
	p := new(Provisioner)
	p.Prepare(config)

	expectedValue := "keyone=valueone"
	if p.config.Vars[0] != expectedValue {
		t.Fatalf("%s should be equal to %s", p.config.Vars[0], expectedValue)
	}

	expectedValue = "keytwo=value\ntwo"
	if p.config.Vars[1] != expectedValue {
		t.Fatalf("%s should be equal to %s", p.config.Vars[1], expectedValue)
	}

	expectedValue = "keythree='valuethree'"
	if p.config.Vars[2] != expectedValue {
		t.Fatalf("%s should be equal to %s", p.config.Vars[2], expectedValue)
	}

	expectedValue = "keyfour='value\nfour'"
	if p.config.Vars[3] != expectedValue {
		t.Fatalf("%s should be equal to %s", p.config.Vars[3], expectedValue)
	}
}

func testUi() *packer.BasicUi {
	return &packer.BasicUi{
		Reader:      new(bytes.Buffer),
		Writer:      new(bytes.Buffer),
		ErrorWriter: new(bytes.Buffer),
	}
}

func testObjects() (packer.Ui, packer.Communicator) {
	ui := testUi()
	return ui, new(packer.MockCommunicator)
}

func TestProvisionerProvision_Inline(t *testing.T) {
	config := testConfig()
	delete(config, "inline")
	config["scripts"] = []string{}
	ui := testUi()

	p := new(Provisioner)
	comm := new(packer.MockCommunicator)
	p.Prepare(config)
	err := p.Provision(ui, comm)

	// Should create a remote command from

	if err != nil {
		t.Fatal("should not have error")
	}

}

func TestProvisionerProvision_Scripts(t *testing.T) {

	// Create n scripts

	// Should execute them in order from 1-n

	// Should invoke 'runcommand' / 'Start'
}

func TestProvisionerProvision_UISlurp(t *testing.T) {
	// UI should be called n times

	// UI should receive following messages / output
}

func TestProvisioner_createFlattenedEnvVars_windows(t *testing.T) {
	config := testConfig()

	p := new(Provisioner)
	err := p.Prepare(config)
	if err != nil {
		t.Fatalf("should not have error preparing config: %s", err)
	}

	// Defaults provided by Packer
	p.config.PackerBuildName = "vmware"
	p.config.PackerBuilderType = "iso"

	// no user env var
	flattenedEnvVars, err := p.createFlattenedEnvVars()
	if err != nil {
		t.Fatalf("should not have error creating flattened env vars: %s", err)
	}
	if flattenedEnvVars != "$env:PACKER_BUILDER_TYPE=\\\"iso\\\"; $env:PACKER_BUILD_NAME=\\\"vmware\\\"; " {
		t.Fatalf("unexpected flattened env vars: %s", flattenedEnvVars)
	}

	// single user env var
	p.config.Vars = []string{"FOO=bar"}

	flattenedEnvVars, err = p.createFlattenedEnvVars()
	if err != nil {
		t.Fatalf("should not have error creating flattened env vars: %s", err)
	}
	if flattenedEnvVars != "$env:FOO=\\\"bar\\\"; $env:PACKER_BUILDER_TYPE=\\\"iso\\\"; $env:PACKER_BUILD_NAME=\\\"vmware\\\"; " {
		t.Fatalf("unexpected flattened env vars: %s", flattenedEnvVars)
	}

	// multiple user env vars
	p.config.Vars = []string{"FOO=bar", "BAZ=qux"}

	flattenedEnvVars, err = p.createFlattenedEnvVars()
	if err != nil {
		t.Fatalf("should not have error creating flattened env vars: %s", err)
	}
	if flattenedEnvVars != "$env:BAZ=\\\"qux\\\"; $env:FOO=\\\"bar\\\"; $env:PACKER_BUILDER_TYPE=\\\"iso\\\"; $env:PACKER_BUILD_NAME=\\\"vmware\\\"; " {
		t.Fatalf("unexpected flattened env vars: %s", flattenedEnvVars)
	}
}
