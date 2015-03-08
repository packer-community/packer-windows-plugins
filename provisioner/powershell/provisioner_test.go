package powershell

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/packer/packer"
)

func testConfig() map[string]interface{} {
	return map[string]interface{}{
		"inline": []interface{}{"foo", "bar"},
	}
}

func TestProvisionerPrepare_extractScript(t *testing.T) {
	config := testConfig()
	p := new(Provisioner)
	_ = p.Prepare(config)
	file, err := extractScript(p)
	if err != nil {
		t.Fatalf("Should not be error: %s", err)
	}
	t.Logf("File: %s", file)
	if strings.Index(file, os.TempDir()) != 0 {
		t.Fatalf("Temp file should reside in %s. File location: %s", os.TempDir(), file)
	}

	// File contents should contain 2 lines concatenated by newlines: foo\nbar
	readFile, err := ioutil.ReadFile(file)
	expectedContents := "foo\nbar\n"
	s := string(readFile[:])
	if s != expectedContents {
		t.Fatalf("Expected generated inlineScript to equal '%s', got '%s'", expectedContents, s)
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

	if p.config.ElevatedUser != "" {
		t.Error("expected elevated_user to be empty")
	}
	if p.config.ElevatedPassword != "" {
		t.Error("expected elevated_password to be empty")
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

	// Defaults provided by Packer
	config["remote_path"] = "c:/Windows/Temp/inlineScript.bat"
	config["inline"] = []string{"whoami"}
	ui := testUi()
	p := new(Provisioner)

	// Defaults provided by Packer
	p.config.PackerBuildName = "vmware"
	p.config.PackerBuilderType = "iso"
	comm := new(packer.MockCommunicator)
	p.Prepare(config)
	err := p.Provision(ui, comm)
	if err != nil {
		t.Fatal("should not have error")
	}

	expectedCommand := `powershell "& { $env:PACKER_BUILDER_TYPE=\"iso\"; $env:PACKER_BUILD_NAME=\"vmware\"; c:/Windows/Temp/inlineScript.bat }"`

	// Should run the command without alteration
	if comm.StartCmd.Command != expectedCommand {
		t.Fatalf("Expect command to be: %s, got %s", expectedCommand, comm.StartCmd.Command)
	}

	envVars := make([]string, 2)
	envVars[0] = "FOO=BAR"
	envVars[1] = "BAR=BAZ"
	config["environment_vars"] = envVars
	config["remote_path"] = "c:/Windows/Temp/inlineScript.bat"

	p.Prepare(config)
	err = p.Provision(ui, comm)
	if err != nil {
		t.Fatal("should not have error")
	}

	expectedCommand = `powershell "& { $env:BAR=\"BAZ\"; $env:FOO=\"BAR\"; $env:PACKER_BUILDER_TYPE=\"iso\"; $env:PACKER_BUILD_NAME=\"vmware\"; c:/Windows/Temp/inlineScript.bat }"`

	// Should run the command without alteration
	if comm.StartCmd.Command != expectedCommand {
		t.Fatalf("Expect command to be: %s, got: %s", expectedCommand, comm.StartCmd.Command)
	}
}
func xTestProvisionerProvision_Inline(t *testing.T) {
	config := testConfig()
	delete(config, "inline")
	config["scripts"] = []string{}
	config["inline"] = "powershell -command Foo-Command"
	ui := testUi()

	p := new(Provisioner)
	comm := new(packer.MockCommunicator)
	p.Prepare(config)
	err := p.Provision(ui, comm)
	if err != nil {
		t.Fatal("should not have error")
	}

	// Should run the command without alteration
	if comm.StartCmd.Command != config["inline"] {
		t.Fatalf("Expect command not to be: %s", comm.StartCmd.Command)
	}

	// Env vars - currently should not effect them
	envVars := make([]string, 2)
	envVars[0] = "FOO=BAR"
	envVars[1] = "BAR=BAZ"
	config["Vars"] = envVars

	p.Prepare(config)
	err = p.Provision(ui, comm)
	if err != nil {
		t.Fatal("should not have error")
	}

	// Should run the command without alteration
	if comm.StartCmd.Command != config["inline"] {
		t.Fatalf("Expect command not to be: %s", comm.StartCmd.Command)
	}
}

func TestProvisionerProvision_Scripts(t *testing.T) {
	tempFile, _ := ioutil.TempFile("", "packer")
	defer os.Remove(tempFile.Name())
	config := testConfig()
	delete(config, "inline")
	config["scripts"] = []string{tempFile.Name()}
	config["packer_build_name"] = "foobuild"
	config["packer_builder_type"] = "footype"
	ui := testUi()

	p := new(Provisioner)
	comm := new(packer.MockCommunicator)
	p.Prepare(config)
	err := p.Provision(ui, comm)
	if err != nil {
		t.Fatal("should not have error")
	}

	//powershell -Command "$env:PACKER_BUILDER_TYPE=''"; powershell -Command "$env:PACKER_BUILD_NAME='foobuild'";  powershell -Command c:/Windows/Temp/script.ps1
	expectedCommand := `powershell "& { $env:PACKER_BUILDER_TYPE=\"footype\"; $env:PACKER_BUILD_NAME=\"foobuild\"; c:/Windows/Temp/script.ps1 }"`

	// Should run the command without alteration
	if comm.StartCmd.Command != expectedCommand {
		t.Fatalf("Expect command to be %s NOT %s", expectedCommand, comm.StartCmd.Command)
	}
}

func TestProvisionerProvision_ScriptsWithEnvVars(t *testing.T) {
	tempFile, _ := ioutil.TempFile("", "packer")
	config := testConfig()
	ui := testUi()
	defer os.Remove(tempFile.Name())
	delete(config, "inline")

	config["scripts"] = []string{tempFile.Name()}
	config["packer_build_name"] = "foobuild"
	config["packer_builder_type"] = "footype"

	// Env vars - currently should not effect them
	envVars := make([]string, 2)
	envVars[0] = "FOO=BAR"
	envVars[1] = "BAR=BAZ"
	config["environment_vars"] = envVars

	p := new(Provisioner)
	comm := new(packer.MockCommunicator)
	p.Prepare(config)
	err := p.Provision(ui, comm)
	if err != nil {
		t.Fatal("should not have error")
	}

	expectedCommand := `powershell "& { $env:BAR=\"BAZ\"; $env:FOO=\"BAR\"; $env:PACKER_BUILDER_TYPE=\"footype\"; $env:PACKER_BUILD_NAME=\"foobuild\"; c:/Windows/Temp/script.ps1 }"`

	// Should run the command without alteration
	if comm.StartCmd.Command != expectedCommand {
		t.Fatalf("Expect command to be %s NOT %s", expectedCommand, comm.StartCmd.Command)
	}
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

func TestRetryable(t *testing.T) {
	config := testConfig()

	count := 0
	retryMe := func() error {
		t.Logf("RetryMe, attempt number %d", count)
		if count == 2 {
			return nil
		}
		count++
		return errors.New(fmt.Sprintf("Still waiting %d more times...", 2-count))
	}
	retryableSleep = 50 * time.Millisecond
	p := new(Provisioner)
	p.config.RawStartRetryTimeout = "155ms"
	err := p.Prepare(config)
	err = p.retryable(retryMe)
	if err != nil {
		t.Fatalf("should not have error retrying funuction")
	}

	count = 0
	p.config.RawStartRetryTimeout = "10ms"
	err = p.Prepare(config)
	err = p.retryable(retryMe)
	if err == nil {
		t.Fatalf("should have error retrying funuction")
	}
}

func TestCancel(t *testing.T) {
	// Don't actually call Cancel() as it performs an os.Exit(0)
	// which kills the 'go test' tool
}
