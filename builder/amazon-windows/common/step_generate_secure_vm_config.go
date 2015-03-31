package common

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"

	wincommon "github.com/packer-community/packer-windows-plugins/common"
)

type StepGenerateSecureWinRMUserData struct {
	WinRMConfig          *wincommon.WinRMConfig
	WinRMCertificateFile string
	RunConfig            *RunConfig
}

func (s *StepGenerateSecureWinRMUserData) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)

	if !s.RunConfig.ConfigureSecureWinRM {
		return multistep.ActionContinue
	}

	ui.Say("Generating user data for configuring WinRM over TLS...")

	certBytes, err := ioutil.ReadFile(s.WinRMCertificateFile)
	if err != nil {
		ui.Error(fmt.Sprintf("Error reading WinRM certificate file: %s", err))
		return multistep.ActionHalt
	}

	encodedCert := base64.StdEncoding.EncodeToString(certBytes)

	var adminPasswordBuffer bytes.Buffer
	if s.RunConfig.NewAdministratorPassword != "" {
		ui.Say("Configuring user data to change Administrator password...")
		err = changeAdministratorPasswordTemplate.Execute(&adminPasswordBuffer, changeAdministratorPasswordOptions{
			NewAdministratorPassword: s.RunConfig.NewAdministratorPassword,
		})
		if err != nil {
			ui.Error(fmt.Sprintf("Error executing Change Administrator Password template: %s", err))
			return multistep.ActionHalt
		}
	}

	var buffer bytes.Buffer
	err = configureSecureWinRMTemplate.Execute(&buffer, configureSecureWinRMOptions{
		CertificatePfxBase64Encoded:        encodedCert,
		InstallListenerCommand:             installListenerCommand,
		AllowBasicCommand:                  allowBasicCommand,
		AllowUnencryptedCommand:            allowUnencryptedCommand,
		AllowCredSSPCommand:                allowCredSSPCommand,
		MaxMemoryPerShellCommand:           maxMemoryPerShellCommand,
		MaxTimeoutMsCommand:                maxTimeoutMsCommand,
		ChangeAdministratorPasswordCommand: adminPasswordBuffer.String(),
	})
	if err != nil {
		ui.Error(fmt.Sprintf("Error executing Secure WinRM User Data template: %s", err))
		return multistep.ActionHalt
	}

	s.RunConfig.UserData = buffer.String()
	return multistep.ActionContinue
}

func (s *StepGenerateSecureWinRMUserData) Cleanup(multistep.StateBag) {
	// No cleanup...
}

type changeAdministratorPasswordOptions struct {
	NewAdministratorPassword string
}

var changeAdministratorPasswordTemplate = template.Must(template.New("ChangeAdministratorPassword").Parse(`$user = [adsi]"WinNT://localhost/Administrator,user"
$user.SetPassword("{{.NewAdministratorPassword}}")
$user.SetInfo()`))

type configureSecureWinRMOptions struct {
	CertificatePfxBase64Encoded        string
	InstallListenerCommand             string
	AllowBasicCommand                  string
	AllowUnencryptedCommand            string
	AllowCredSSPCommand                string
	MaxMemoryPerShellCommand           string
	MaxTimeoutMsCommand                string
	ChangeAdministratorPasswordCommand string
}

//This is needed to because Powershell uses ` for escapes and there's no straightforward way of constructing
// the necessary escaping in the hash otherwise.
const (
	installListenerCommand = "Start-Process -FilePath winrm -ArgumentList \"create winrm/config/listener?Address=*+Transport=HTTPS @{Hostname=`\"$certSubjectName`\";CertificateThumbprint=`\"$certThumbprint`\";Port=`\"5986`\"}\" -NoNewWindow -Wait"

	allowBasicCommand        = "Start-Process -FilePath winrm -ArgumentList \"set winrm/config/service/auth @{Basic=`\"true`\"}\" -NoNewWindow -Wait"
	allowUnencryptedCommand  = "Start-Process -FilePath winrm -ArgumentList \"set winrm/config/service @{AllowUnencrypted=`\"false`\"}\" -NoNewWindow -Wait"
	allowCredSSPCommand      = "Start-Process -FilePath winrm -ArgumentList \"set winrm/config/service/auth @{CredSSP=`\"true`\"}\" -NoNewWindow -Wait"
	maxMemoryPerShellCommand = "Start-Process -FilePath winrm -ArgumentList \"set winrm/config/winrs @{MaxMemoryPerShellMB=`\"1024`\"}\" -NoNewWindow -Wait"
	maxTimeoutMsCommand      = "Start-Process -FilePath winrm -ArgumentList \"set winrm/config @{MaxTimeoutms=`\"1800000`\"}\" -NoNewWindow -Wait"
)

var configureSecureWinRMTemplate = template.Must(template.New("ConfigureSecureWinRM").Parse(`<powershell>
Write-Host "Setting execution policy to RemoteSigned..."
Set-ExecutionPolicy RemoteSigned

Write-Host "Disabling WinRM over HTTP..."
Disable-NetFirewallRule -Name "WINRM-HTTP-In-TCP"
Disable-NetFirewallRule -Name "WINRM-HTTP-In-TCP-PUBLIC"

Start-Process -FilePath winrm -ArgumentList "delete winrm/config/listener?Address=*+Transport=HTTP" -NoNewWindow -Wait

Write-Host "Configuring WinRM for HTTPS..."

{{.MaxTimeoutMsCommand}}

{{.MaxMemoryPerShellCommand}}

{{.AllowUnencryptedCommand}}

{{.AllowBasicCommand}}

{{.AllowCredSSPCommand}}

New-NetFirewallRule -Name "WINRM-HTTPS-In-TCP" -DisplayName "Windows Remote Management (HTTPS-In)" -Description "Inbound rule for Windows Remote Management via WS-Management. [TCP 5986]" -Group "Windows Remote Management" -Program "System" -Protocol TCP -LocalPort "5986" -Action Allow -Profile Domain,Private

New-NetFirewallRule -Name "WINRM-HTTPS-In-TCP-PUBLIC" -DisplayName "Windows Remote Management (HTTPS-In)" -Description "Inbound rule for Windows Remote Management via WS-Management. [TCP 5986]" -Group "Windows Remote Management" -Program "System" -Protocol TCP -LocalPort "5986" -Action Allow -Profile Public

$certContent = "{{.CertificatePfxBase64Encoded}}"

$certBytes = [System.Convert]::FromBase64String($certContent)
$pfx = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2
$pfx.Import($certBytes, "", "Exportable,PersistKeySet,MachineKeySet")
$certThumbprint = $pfx.Thumbprint
$certSubjectName = $pfx.SubjectName.Name.TrimStart("CN = ").Trim()

$store = new-object System.Security.Cryptography.X509Certificates.X509Store("My", "LocalMachine")
try {
    $store.Open("ReadWrite,MaxAllowed")
    $store.Add($pfx)

} finally {
    $store.Close()
}

{{.InstallListenerCommand}}

{{.ChangeAdministratorPasswordCommand}}

Write-Host "Restarting WinRM Service..."
Stop-Service winrm
Set-Service winrm -StartupType "Automatic"
Start-Service winrm
</powershell>`))
