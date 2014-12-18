package winrm

import (
	"bufio"
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

	err := f.runCommand(downloadCommand)
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
	err = f.UploadFile(dst, tmp, server)

	return err
}

func (f *fileManager) UploadDir(dst string, src string) error {
	log.Printf("Uploading dir to %s from %s", dst, src)

	// We need these dirs later when walking files
	f.guestUploadDir = dst
	f.hostUploadDir = src

	// Walk all files in the src directory on the host system
	return filepath.Walk(src, f.uploadFileWalker)
}

//
// /tmp/foo/
//   - bar.txt
//   - bat/
//   	- bat.txt
//   	- baz.txt
//   - bro.txt
func (f *fileManager) uploadFileWalker(hostPath string, hostFileInfo os.FileInfo, err error) error {

	// Game plan:
	//
	// 1. if it's a file and should be uploaded, upload file
	// 2. if it's a dir, create it first on the client
	// 3. ...repeat process recursively until all dirs have been created and files uploaded
	// NOTE: Re-use the HTTP server - this may require some form of state

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
		if err != nil {
			log.Printf("Unable to upload file %s to path %s, error: ", hostPath, err)
			return err
		}
	} else if hostFileInfo.IsDir() {
		relPath := filepath.Dir(hostPath[len(f.hostUploadDir):len(hostPath)])
		log.Printf("Found a directory, preparing it: %s", relPath)
		err = f.prepareFileDirectory(filepath.Join(f.guestUploadDir, relPath))
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

	command := fmt.Sprintf(`powershell -Command "& { md -Force $([System.IO.Path]::GetFullPath('%s')) }"`, dst)

	err := f.runCommand(command)

	return err
}

func shouldUploadFile(hostFile os.FileInfo) bool {
	// Ignore dir entries and OS X special hidden file
	return !hostFile.IsDir() && ".DS_Store" != hostFile.Name()
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
