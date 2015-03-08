package main

import (
	"flag"
	plugin "github.com/packer-community/packer-windows-plugins/communicator/winrm"
	"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/packer/packer"
	rpc "github.com/mitchellh/packer/packer/plugin"
	"github.com/rakyll/command"
	"log"
	"os"
	"time"
)

var host = flag.String("host", "localhost", "host machine")
var port = flag.Int("port", 5985, "host port")
var user = flag.String("user", "vagrant", "user to run as")
var pass = flag.String("pass", "vagrant", "user's password")
var timeout = flag.Duration("timeout", 60*time.Second, "connection timeout")

func main() {
	args := os.Args[1:]
	if len(args) != 0 {
		standalone()
	} else {
		server, err := rpc.Server()
		if err != nil {
			panic(err)
		}
		server.RegisterCommunicator(new(plugin.Communicator))
		server.Serve()
	}
}

func standalone() {
	command.On("cmd", "run a command", &RunCommand{}, []string{})
	command.On("file", "copy a file", &FileCommand{}, []string{})
	command.On("dir", "copy a dir", &DirCommand{}, []string{})
	command.Parse()
	command.Run()
}

type RunCommand struct{}

func (r *RunCommand) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (r *RunCommand) Run(args []string) {
	command := args[0]

	endpoint := &winrm.Endpoint{Host: *host, Port: *port}
	communicator, err := plugin.New(endpoint, *user, *pass, *timeout)
	rc := &packer.RemoteCmd{
		Command: command,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
	if err != nil {
		log.Printf("unable to run command: %s", err)
		return
	}

	err = communicator.Start(rc)
	if err != nil {
		log.Printf("unable to run command: %s", err)
		return
	}

	rc.Wait()
}

type FileCommand struct {
	to   *string
	from *string
}

func (f *FileCommand) Flags(fs *flag.FlagSet) *flag.FlagSet {
	f.to = fs.String("to", "", "destination file path")
	f.from = fs.String("from", "", "source file path")
	return fs
}

func (f *FileCommand) Run(args []string) {
	endpoint := &winrm.Endpoint{Host: *host, Port: *port}
	communicator, err := plugin.New(endpoint, *user, *pass, *timeout)

	info, err := os.Stat(*f.from)
	if err != nil {
		log.Panicln("unable to stat file", err.Error())
	}

	file, err := os.Open(*f.from)
	if err != nil {
		log.Panicln("unable to open file", err.Error())
	}

	err = communicator.Upload(*f.to, file, &info)
	if err != nil {
		log.Printf("unable to copy file: %s", err)
	}
}

type DirCommand struct {
	to   *string
	from *string
}

func (f *DirCommand) Flags(fs *flag.FlagSet) *flag.FlagSet {
	f.to = fs.String("to", "", "destination file path")
	f.from = fs.String("from", "", "source file path")
	return fs
}

func (f *DirCommand) Run(args []string) {
	endpoint := &winrm.Endpoint{Host: *host, Port: *port}
	communicator, _ := plugin.New(endpoint, *user, *pass, *timeout)

	_, err := os.Stat(*f.from)
	if err != nil {
		log.Panicln("unable to stat dir", err.Error())
	}

	err = communicator.UploadDir(*f.to, *f.from, nil)
	if err != nil {
		log.Printf("unable to copy dir: %s", err)
	}
}
