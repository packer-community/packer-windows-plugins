## Packer Windows Plugins

A suite of [Packer](http://www.packer.io/) plugins for provisioning Windows machines using Windows Remote Management.

### Status

The plugins are generally available and actively maintained.

[![wercker status](https://app.wercker.com/status/900b58d8e99fca90bcfcd599a5e5219e/m/master "wercker status")](https://app.wercker.com/project/bykey/900b58d8e99fca90bcfcd599a5e5219e)
[![Coverage Status](https://coveralls.io/repos/packer-community/packer-windows-plugins/badge.png?branch=HEAD)](https://coveralls.io/r/packer-community/packer-windows-plugins)

### The Plugins

We have created the following Windows-specific plugins:

#### Builders

* VirtualBox ISO and OVF (`virtualbox-windows-iso`, `virtualbox-windows-ovf`)
* VMWare ISO and VMX (`vmware-windows-iso`, `vmware-windows-vmx`)
* Parallels ISO and PVM (`parallels-windows-iso`, `parallels-windows-pvm`)
* Amazon EBS (`amazon-windows-ebs`)

#### Provisioners

* Powershell (`powershell`)
* Windows Shell (`windows-shell`)
* Restart Windows (`restart-windows`)

### Getting Started

The plugins can be used by downloading pre-built binaries, or by building the project locally and ensuring the binaries are in the correct location.

#### Using pre-built binaries

1. Install Packer
1. Download the latest release for your host environment: [packer-windows-plugins/releases](https://github.com/packer-community/packer-windows-plugins/releases)
1. Unzip the plugin binaries to [a location where Packer will detect them at run-time](https://packer.io/docs/extend/plugins.html), such as any of the following:
  - The directory where the packer binary is.
  - `~/.packer.d/plugins` on Unix systems or `%APPDATA%/packer.d/plugins` on Windows.
  - The current working directory.
1. Change to a directory where you have packer templates, and run as usual.

#### Using Chocolatey

```
choco install packer-windows-plugins -pre # Only until the package is moderated
```

#### Using a local build

With [Go 1.2+](http://golang.org) installed, follow these steps to use these community plugins for Windows:

1. Install packer
1. Clone this repo
1. Run `make dev`
1. Copy the plugin binaries located in `./bin` to [a location where Packer will detect them at run-time](https://packer.io/docs/extend/plugins.html), such as any of the following:
  - The directory where the packer binary is. If you've built Packer locally, then Packer and the new plugins are already in `$GOPATH/bin` together.
  - `~/.packer.d/plugins` on Unix systems or `%APPDATA%/packer.d` on Windows.
  - The current working directory.
1. Change to a directory where you have packer templates, and run as usual.

#### A simple Packer template

A simple Packer template for Windows, which utilizes the `virtualbox-windows-iso` builder and `winrm` communicator plugins, will look something like

```json
  {
    "builders": [{
      "type": "virtualbox-windows-iso",
      "vm_name": "windows_2012_r2",
      "iso_url": "iso/en_windows_server_2012_r2_with_update_x64_dvd_4065220.iso",
      "iso_checksum_type": "md5",
      "iso_checksum": "af6a7f966b4c4e31da5bc3cdc3afcbec",
      "guest_os_type": "Windows2012_64",
      "boot_wait": "2m",
      "winrm_username": "packer",
      "winrm_password": "packer",
      "winrm_wait_timeout": "10m",
      "shutdown_timeout": "1h",
      "shutdown_command": "shutdown /s /t 10 /f /d p:4:1 /c \"Packer Shutdown\"",
      "disk_size": 61440,
      "format": "ova",
      "floppy_files": [
        "./Autounattend.xml",
        "./enable-winrm.bat",
      ]
    }],
    "provisioners": [{
      "type": "powershell",
      "scripts": [
        "scripts/chocolatey.ps1"
      ]
    },{
      "type": "powershell",
      "inline": [
        "choco install 7zip",
        "choco install dotnet4.5.2"
      ]
    },{
      "type": "windows-shell",
      "scripts": [
        "scripts/netsh.bat"
      ]
    }],
    "post-processors": [{
      "type": "vagrant",
      "output": "windows_2012_r2_virtualbox.box",
      "vagrantfile_template": "Vagrantfile.template"
    }]
  }
```

Check out these projects for more detailed examples of Windows-centric Packer templates:
- [dylanmei/packer-windows-templates](https://github.com/dylanmei/packer-windows-templates)
- [joefitzgerald/packer-windows](https://github.com/joefitzgerald/packer-windows)
- [box-cutter/windows-vm](https://github.com/box-cutter/windows-vm)

### Running commands with Elevated privileges

In certain situations, you may need to run commands with elevated privileges even if your `winrm_username` user is an Administrator, for example upgrading system packages like the .NET Framework.
In these cases there are 2 additional keys provided on the `powershell` provisioner that you can supply that will enable this mode: `elevated_user` and `elevated_password`:

```
{
  "type": "powershell",
  "elevated_user":"vagrant",
  "elevated_password":"vagrant",
  "inline": [
    "choco install netfx-4.5.2-devpack"
  ]
}
```

### Community
- **IRC**: `#packer-community` on Freenode.
- **Slack**: packer.slack.com
