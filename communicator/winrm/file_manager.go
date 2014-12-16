package winrm

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/packer/packer"
)

type fileManager struct {
	comm               packer.Communicator
	server             *http.Server
	guestUploadDir     string
	hostUploadDir      string
	webServerIpAddress string
}

const DEFAULT_HOST_IP_ADDRESS = "10.0.2.2"

func NewFileManager(comm packer.Communicator) (*fileManager, error) {
	return &fileManager{comm: comm, webServerIpAddress: DEFAULT_HOST_IP_ADDRESS}, nil
}

// Get a WebServer to serve the given file in the host.
//
// Provides:
//
//	int	The listen port of the server
//  http.server A reference to the Server to run
func (f *fileManager) getHttpServer(uploadFile os.File) *http.Server {
	// Find an available TCP port for our HTTP server
	var httpAddr string
	portRange := 1000
	var httpPort uint

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
		l, err := net.Listen("tcp", httpAddr)
		if err == nil {
			// Free port. TODO: Maybe pass the listener around instead
			l.Close()
			break
		}
	}

	log.Printf("Returning HTTP server on port %d hosting files in dir %s", httpPort, path.Dir(uploadFile.Name()))
	fileServer := http.FileServer(http.Dir(path.Dir(uploadFile.Name())))
	server := &http.Server{Addr: httpAddr, Handler: fileServer}

	return server
}

func (f *fileManager) UploadFile(dst string, src *os.File, server *http.Server) error {
	winDest := winFriendlyPath(dst)
	log.Printf("Uploading: %s -> %s", src.Name(), winDest)

	// Start the HTTP server and run it in the background
	parts := strings.SplitN(server.Addr, ":", 2)
	port := parts[1]
	go server.ListenAndServe()

	// Pull down file via remote command
	log.Printf("Uploading \"%s\"with the HTTP Server on ip %s and port %s with path %s", src.Name(), f.webServerIpAddress, port, winDest)
	//downloadCommand := fmt.Sprintf("powershell -Command \"iex ((new-object net.webclient).DownloadFile('http://%s:%d/%s', '%s'))\"", ipAddress, httpPort, path.Base(tmp.Name()), winDest)
	//downloadCommand := fmt.Sprintf("powershell \"iex ((new-object net.webclient).DownloadFile('http://%s:%d/%s', '%s'))\"", ipAddress, httpPort, path.Base(tmp.Name()), winDest)
	downloadCommand := fmt.Sprintf("powershell Invoke-WebRequest 'http://%s:%s/%s' -OutFile %s", f.webServerIpAddress, port, path.Base(src.Name()), winDest)
	log.Printf("Executing download command: %s", downloadCommand)

	cmd := &packer.RemoteCmd{
		Command: downloadCommand,
	}
	err := f.comm.Start(cmd)
	return err
}

func (f *fileManager) Upload(dst string, input io.Reader) error {

	// Copy file to local temp file
	tmp, err := f.TempFile(input)
	if err != nil {
		log.Print("Error creating temporary upload of file: %s", err)
		return err
	}

	server := f.getHttpServer(*tmp)
	f.UploadFile(dst, tmp, server)

	return err
}

var uploadDir = func(f *fileManager, dst string, src string) error {
	log.Printf("uploadDir to %s from %s", dst, src)

	// We need these dirs later when walking files
	f.guestUploadDir = dst
	f.hostUploadDir = src

	// Walk all files in the src directory on the host system
	return filepath.Walk(src, f.uploadFileWalker)
}

func (f *fileManager) UploadDir(dst string, src string) error {
	return uploadDir(f, dst, src)
}

//
// /tmp/foo/
//   - bar.txt
//   - bat/
//   	- bat.txt
//   	- baz.txt
//   - bro.txt
func (f *fileManager) uploadFileWalker(hostPath string, hostFileInfo os.FileInfo, err error) error {
	log.Printf("uploadFileWalker hostUploadDir: %s, hostpath: %s, hostFileInfo.name(): %s", f.hostUploadDir, hostPath, hostFileInfo.Name())
	if err == nil && shouldUploadFile(hostFileInfo) {
		relPath := filepath.Dir(hostPath[len(f.hostUploadDir):len(hostPath)])
		guestPath := filepath.Join(f.guestUploadDir, relPath, hostFileInfo.Name())
		hostFile, err := os.Open(hostPath)
		defer hostFile.Close()
		if err != nil {
			log.Printf("Unable to open source file %s for upload: %s", hostPath, err)
			return err
		}
		server := f.getHttpServer(*hostFile)
		err = f.UploadFile(guestPath, hostFile, server)
	} else if hostFileInfo.IsDir() {
		relPath := filepath.Dir(hostPath[len(f.hostUploadDir):len(hostPath)])
		log.Printf("Found a directory, preparing it: %s", relPath)
		f.prepareFileDirectory(relPath)
	}
	return err
}

func (f *fileManager) runCommand(cmd string) error {
	remoteCmd := &packer.RemoteCmd{
		Command: cmd,
	}

	err := f.comm.Start(remoteCmd)
	if err != nil {
		return err
	}
	remoteCmd.Wait()

	if remoteCmd.ExitStatus != 0 {
		return errors.New("A file upload command failed with a non-zero exit code")
	}

	return nil
}

func winFriendlyPath(path string) string {
	return strings.Replace(path, "/", "\\", -1)
}

func (f *fileManager) prepareFileDirectory(dst string) error {
	log.Printf("Preparing directory for upload: %s", dst)

	command := fmt.Sprintf(`
$dest_file_path = [System.IO.Path]::GetFullPath("%s")
if (-not (Test-Path $dest_file_path) ) {
  rm $dest_file_path
  Write-Output "Creating directory: $dest_file_path"
  md $dest_file_path -Force
}`, dst)

	cmd := &packer.RemoteCmd{
		Command: command,
	}

	err := f.comm.Start(cmd)

	return err
}

func shouldUploadFile(hostFile os.FileInfo) bool {
	// Ignore dir entries and OS X special hidden file
	return !hostFile.IsDir() && ".DS_Store" != hostFile.Name()
}

func encodeChunks(bytes []byte, chunkSize int) []string {
	text := base64.StdEncoding.EncodeToString(bytes)
	reader := strings.NewReader(text)

	var chunks []string
	chunk := make([]byte, chunkSize)

	for {
		n, _ := reader.Read(chunk)
		if n == 0 {
			break
		}

		chunks = append(chunks, string(chunk[:n]))
	}

	return chunks
}

func (f *fileManager) TempFile(input io.Reader) (*os.File, error) {
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
