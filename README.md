## Packer Windows Plugins

A suite of [Packer](http://www.packer.io/) plugins for provisioning Windows machines using Windows Remote Management.

### Status

This is super experimental. *These are not complete Packer plugins yet*.

[![wercker status](https://app.wercker.com/status/900b58d8e99fca90bcfcd599a5e5219e/m "wercker status")](https://app.wercker.com/project/bykey/900b58d8e99fca90
bcfcd599a5e5219e)

### Getting Started

*Instructions for using these plugins without [Go](http://golang.org) is forthcoming.*

With [Go 1.2+](http://golang.org) installed, follow these steps to use these community plugins for Windows:

1. Install packer
1. Clone this repo
1. Run `make dev`
1. Copy the output of `./bin` to `~/.packer.d/plugins` on Unix systems or `%APPDATA%/packer.d` on Windows.
1. Change to a directory where you have packer templates, and run.

The simplest packer template for Windows, which utilizes the `virtualbox-windows-iso` builder and `winrm` communicator plugins, will look something like

<pre>
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
          "./winrm.bat",
        ],
        "vboxmanage": [
            [ "modifyvm", "{{.Name}}", "--memory", "2048" ],
        ]
      }],
    }
</pre>


We recommend these projects for more information about, and examples of Windows-centric Packer templates:

- [joefitzgerald/packer-windows](https://github.com/joefitzgerald/packer-windows)
- [mwrock/boxstarter](https://github.com/mwrock/boxstarter)
- [dylanmei/packer-windows-templates](https://github.com/dylanmei/packer-windows-templates)

### Community

- *IRC*: `#packer-tool` / `#packer-windows` on Freenode.
- *Slack*: packer.slack.com
