# Packer WinRM Shell Provisioner

[![Build Status](https://travis-ci.org/mefellows/packer-winrm-shell.svg?branch=master)](https://travis-ci.org/mefellows/packer-winrm-shell)
[![Coverage Status](https://coveralls.io/repos/mefellows/packer-winrm-shell/badge.png)](https://coveralls.io/r/mefellows/packer-winrm-shell)

The [WinRM]() Shell Provisioner for Packer allows you to provision a Windows Guest 
via shell commands or scripts that are executed via WinRM instead of the default SSH
behaviour.

WinRM is a native Windows remote management capability that can provide a more reliable
experience interacting with a remote Windows machine.

## Installation

Download the binary for your platform and place on your execution `$PATH` environment variable, 
or alongside the Packer.io installation.

### Manual Installation

TBA

## Usage

The WinRM shell is very similar to the default Packer [Shell](https://www.packer.io/docs/provisioners/shell.html) 
provisioner, the main difference being the name of the provisioner is changed to `winrm-shell`:

```
"provisioners": [
   {
     "type": "winrm-shell",
     "inline": ["whoami"]
   }
]
```

### Configuration Options

Aside from the default parameters from the [Shell](https://www.packer.io/docs/provisioners/shell.html) Provisioner, you can use Modified the below:

*Authentication*
* Username: The WinRM username. Defaults to 'vagrant'
* Password: The WinRM password. Defaults to 'vagrant'
* Hostname: The WinRM hostname. Defaults to 'localhost'
* Port: The WinRM connection port. Defaults to '5985'

## Contributing
