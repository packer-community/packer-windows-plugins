package winrm

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/packer/packer"
)

type Communicator struct {
	client   *winrm.Client
	endpoint *winrm.Endpoint
	user     string
	password string
	timeout  time.Duration
}

type elevatedShellOptions struct {
	Command  string
	User     string
	Password string
}

// Creates a new packer.Communicator implementation over WinRM.
// Called when Packer tries to connect to WinRM
func New(endpoint *winrm.Endpoint, user string, password string, timeout time.Duration) (*Communicator, error) {

	// Create the WinRM client we use internally
	params := winrm.DefaultParameters()
	params.Timeout = ISO8601DurationString(timeout)
	client := winrm.NewClientWithParameters(endpoint, user, password, params)

	// Attempt to connect to the WinRM service
	shell, err := client.CreateShell()
	if err != nil {
		return nil, err
	}

	err = shell.Close()
	if err != nil {
		return nil, err
	}

	return &Communicator{
		endpoint: endpoint,
		user:     user,
		password: password,
		timeout:  timeout,
		client:   client,
	}, nil
}

func (c *Communicator) Start(cmd *packer.RemoteCmd) (err error) {
	// TODO: Can we only run as Elevated if specified in config/setting.
	// It's fairly slow. It also doesn't work see Issue #1
	//return c.StartElevated(cmd)
	return c.StartUnelevated(cmd)
}

func (c *Communicator) StartElevated(cmd *packer.RemoteCmd) (err error) {
	// Wrap the command in scheduled task
	tpl, err := packer.NewConfigTemplate()
	if err != nil {
		return err
	}

	// The command gets put into an interpolated string in the PS script,
	// so we need to escape any embedded quotes.
	escapedCmd := strings.Replace(cmd.Command, "\"", "`\"", -1)

	elevatedScript, err := tpl.Process(ElevatedShellTemplate, &elevatedShellOptions{
		Command:  escapedCmd,
		User:     c.user,
		Password: c.password,
	})
	if err != nil {
		return err
	}

	// Upload the script which creates and manages the scheduled task
	log.Printf("uploading elevated command: %s", cmd.Command)
	err = c.Upload("$env:TEMP/packer-elevated-shell.ps1", strings.NewReader(elevatedScript), nil)
	if err != nil {
		return err
	}

	// Run the script that was uploaded
	path := "%TEMP%/packer-elevated-shell.ps1"
	log.Printf("executing elevated command: %s", path)
	command := fmt.Sprintf("powershell -executionpolicy bypass -file \"%s\"", path)
	return c.runCommand(command, cmd)
}

func (c *Communicator) StartUnelevated(cmd *packer.RemoteCmd) (err error) {
	log.Printf("starting remote command: %s", cmd.Command)
	return c.runCommand(cmd.Command, cmd)
}

func (c *Communicator) runCommand(commandText string, cmd *packer.RemoteCmd) (err error) {
	// Create a new shell process on the guest
	err = c.client.RunWithInput(commandText, os.Stdout, os.Stderr, os.Stdin)
	if err != nil {
		fmt.Println(err)
		cmd.SetExited(1)
		return err
	}
	cmd.SetExited(0)
	return
}

func (c *Communicator) TempFile(input io.Reader) (*os.File, error) {
	httpDir := "/tmp"
	tmp, err := ioutil.TempFile(httpDir, "packer-tmp")

	defer func() {
		tmp.Close()
	}()

	r := bufio.NewReader(input)
	b := make([]byte, 1024)
	for {
		_, err := r.Read(b)
		if err == io.EOF {
			break
		}
		tmp.Write(b)
	}

	return tmp, err
}

// func (c *Communicator) CreateWebServer(port string)

// func (c *Communicator) xUploadDir(dst string, src string, excl []string) error {
func (c *Communicator) Upload(dst string, input io.Reader, ignored *os.FileInfo) error {
	log.Println("Uploading with the HTTP Server")

	// Copy file to local temp file
	tmp, err := c.TempFile(input)
	if err != nil {
		log.Print("Error creating temporary upload of file: %s", err)
		return err
	}

	// Find an available TCP port for our HTTP server
	var httpPort uint
	var httpAddr string
	ipAddress := "10.0.2.2"
	portRange := 1000
	var l net.Listener

	log.Print("Looking for an available port...")
	for {
		var err error
		var offset uint = 0

		if portRange > 0 {
			// Intn will panic if portRange == 0, so we do a check.
			offset = uint(rand.Intn(portRange))
		}

		httpPort = offset + 8000
		httpAddr = fmt.Sprintf(":%d", httpPort)
		log.Printf("Trying port: %d", httpPort)
		l, err = net.Listen("tcp", httpAddr)
		if err == nil {
			break
		}
	}

	log.Printf("Starting HTTP server on port %d hosting files in dir %s", httpPort, path.Dir(tmp.Name()))

	// Start the HTTP server and run it in the background
	fileServer := http.FileServer(http.Dir(path.Dir(tmp.Name())))
	server := &http.Server{Addr: httpAddr, Handler: fileServer}
	go server.Serve(l)
	//defer l.Close()

	// Pull down file via remote command
	log.Printf("Uploading \"%s\"with the HTTP Server on ip %s and port %d with path %s", tmp.Name(), ipAddress, httpPort, dst)
	//downloadCommand := fmt.Sprintf("powershell -Command \"iex ((new-object net.webclient).DownloadFile('http://%s:%d/%s', '%s'))\"", ipAddress, httpPort, path.Base(tmp.Name()), dst)
	//downloadCommand := fmt.Sprintf("powershell \"iex ((new-object net.webclient).DownloadFile('http://%s:%d/%s', '%s'))\"", ipAddress, httpPort, path.Base(tmp.Name()), dst)
	downloadCommand := fmt.Sprintf("powershell Invoke-WebRequest 'http://%s:%d/%s' -OutFile %s", ipAddress, httpPort, path.Base(tmp.Name()), dst)
	log.Printf("Executing download command: %s", downloadCommand)

	cmd := &packer.RemoteCmd{
		Command: downloadCommand,
	}
	err = c.runCommand(downloadCommand, cmd)

	// Save the address into the state so it can be accessed in the future
	// 	state.Put("http_port", httpPort)
	return err
}

func (c *Communicator) xUpload(dst string, input io.Reader, ignored *os.FileInfo) error {
	fm := &fileManager{
		comm: c,
	}

	return fm.Upload(dst, input)
}

func (c *Communicator) UploadDir(dst string, src string, excl []string) error {
	fm := &fileManager{
		comm: c,
	}
	return fm.UploadDir(dst, src)
}

func (c *Communicator) Download(string, io.Writer) error {
	panic("Download not implemented yet")
}

const ElevatedShellTemplate = `
$command = "{{.Command}}" + '; exit $LASTEXITCODE'
$user = '{{.User}}'
$password = '{{.Password}}'

$task_name = "packer-elevated-shell"
$out_file = "$env:TEMP\packer-elevated-shell.log"

if (Test-Path $out_file) {
  del $out_file
}

$task_xml = @'
<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <Principals>
    <Principal id="Author">
      <UserId>{user}</UserId>
      <LogonType>Password</LogonType>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>false</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <IdleSettings>
      <StopOnIdleEnd>true</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT2H</ExecutionTimeLimit>
    <Priority>4</Priority>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>cmd</Command>
      <Arguments>{arguments}</Arguments>
    </Exec>
  </Actions>
</Task>
'@

$bytes = [System.Text.Encoding]::Unicode.GetBytes($command)
$encoded_command = [Convert]::ToBase64String($bytes)
$arguments = "/c powershell.exe -EncodedCommand $encoded_command &gt; $out_file 2&gt;&amp;1"

$task_xml = $task_xml.Replace("{arguments}", $arguments)
$task_xml = $task_xml.Replace("{user}", $user)

$schedule = New-Object -ComObject "Schedule.Service"
$schedule.Connect()
$task = $schedule.NewTask($null)
$task.XmlText = $task_xml
$folder = $schedule.GetFolder("\")
$folder.RegisterTaskDefinition($task_name, $task, 6, $user, $password, 1, $null) | Out-Null

$registered_task = $folder.GetTask("\$task_name")
$registered_task.Run($null) | Out-Null

$timeout = 10
$sec = 0
while ( (!($registered_task.state -eq 4)) -and ($sec -lt $timeout) ) {
  Start-Sleep -s 1
  $sec++
}

function SlurpOutput($out_file, $cur_line) {
  if (Test-Path $out_file) {
    get-content $out_file | select -skip $cur_line | ForEach {
      $cur_line += 1
      Write-Host "$_" 
    }
  }
  return $cur_line
}

$cur_line = 0
do {
  Start-Sleep -m 100
  $cur_line = SlurpOutput $out_file $cur_line
} while (!($registered_task.state -eq 3))

$exit_code = $registered_task.LastTaskResult
[System.Runtime.Interopservices.Marshal]::ReleaseComObject($schedule) | Out-Null

exit $exit_code
`

func ISO8601DurationString(d time.Duration) string {
	// We're not supporting negative durations
	if d.Seconds() <= 0 {
		return "PT0S"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) - (hours * 60)
	seconds := int(d.Seconds()) - (hours*3600 + minutes*60)

	s := "PT"
	if hours > 0 {
		s = fmt.Sprintf("%s%dH", s, hours)
	}
	if minutes > 0 {
		s = fmt.Sprintf("%s%dM", s, minutes)
	}
	if seconds > 0 {
		s = fmt.Sprintf("%s%dS", s, seconds)
	}

	return s
}
