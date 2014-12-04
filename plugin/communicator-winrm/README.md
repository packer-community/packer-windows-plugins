## Packer WinRM Plugin

A [Packer](http://www.packer.io/) communicator plugin for interacting with machines using Windows Remote Management. For more information on WinRM, visit [Microsoft's WinRM site](http://msdn.microsoft.com/en-us/library/aa384426\(v=VS.85\).aspx).

### Status

This is a work in progress. *It is not a usable Packer plugin yet*. However, while the kinks are being worked out it is also a stand-alone command-line application.

### Usage

A Packer *communicator* plugin supports the following functionality: Execute a shell command, upload a file, download a file, and upload a directory.

#### Help

    alias pcw=`pwd`/communicator-winrm
    pcw help

#### Executing a shell command

    pcw cmd "powershell Write-Host 'Hello' (Get-WmiObject -class Win32_OperatingSystem).Caption"

#### Uploading a file

    pcw file -from=./README.md -to=C:\\Windows\\Temp\\README.md
    pcw cmd "type C:\\Windows\\Temp\\README.md"

#### Uploading a directory

	pcw dir -from="~/cookbooks/" -to="c:\\Windows\\Temp\\cookbooks"
	pcw cmd "dir c:\\Windows\\Temp\\cookbooks"

#### Downloading a file

*not started*

