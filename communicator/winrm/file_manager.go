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
	comm               *Communicator
	server             *http.Server
	guestUploadDir     string
	hostUploadDir      string
	webServerIpAddress string
	webServerPort      uint
}

func NewFileManager(comm *Communicator) (*fileManager, error) {
	//return &fileManager{comm: comm, server: f.defaultHttpServer()}, nil
	return &fileManager{comm: comm, webServerIpAddress: "10.0.2.2"}, nil
}

// Get a WebServer to serve the given file in the host.
//
// Provides:
//
//	int	The listen port of the server
//  http.server A reference to the Server to run
func (f *fileManager) getHttpServer(uploadFile os.File) (uint, *http.Server) {
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
		_, err = net.Listen("tcp", httpAddr)
		if err == nil {
			break
		}
	}

	log.Printf("Returning HTTP server on port %d hosting files in dir %s", httpPort, path.Dir(uploadFile.Name()))
	fileServer := http.FileServer(http.Dir(path.Dir(uploadFile.Name())))
	server := &http.Server{Addr: httpAddr, Handler: fileServer}

	return httpPort, server
}

func (f *fileManager) UploadFile(dst string, src *os.File) error {
	winDest := winFriendlyPath(dst)
	log.Printf("Uploading: %s ->%s", src.Name(), winDest)

	// Start the HTTP server and run it in the background
	port, server := f.getHttpServer(*src)
	go server.ListenAndServe()
	//l, err := net.Listen("tcp", ":8124")
	//port = 8124
	//go server.Serve(l)

	// Pull down file via remote command
	//log.Printf("Uploading \"%s\"with the HTTP Server on ip %s and port %d with path %s", tmp.Name(), ipAddress, httpPort, winDest)
	//downloadCommand := fmt.Sprintf("powershell -Command \"iex ((new-object net.webclient).DownloadFile('http://%s:%d/%s', '%s'))\"", ipAddress, httpPort, path.Base(tmp.Name()), winDest)
	//downloadCommand := fmt.Sprintf("powershell \"iex ((new-object net.webclient).DownloadFile('http://%s:%d/%s', '%s'))\"", ipAddress, httpPort, path.Base(tmp.Name()), winDest)
	downloadCommand := fmt.Sprintf("powershell Invoke-WebRequest 'http://%s:%d/%s' -OutFile %s", f.webServerIpAddress, port, path.Base(src.Name()), winDest)
	log.Printf("Executing download command: %s", downloadCommand)

	cmd := &packer.RemoteCmd{
		Command: downloadCommand,
	}
	err := f.comm.runCommand(downloadCommand, cmd)
	return err
}

func (f *fileManager) xUploadFile(dst string, src string) error {
	winDest := winFriendlyPath(dst)
	log.Printf("Uploading: %s ->%s", src, winDest)

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	return f.Upload(winDest, srcFile)
}

func (f *fileManager) Upload(dst string, input io.Reader) error {

	// Copy file to local temp file
	tmp, err := f.TempFile(input)
	if err != nil {
		log.Print("Error creating temporary upload of file: %s", err)
		return err
	}

	f.UploadFile(dst, tmp)

	return err
}

func (f *fileManager) UploadDir(dst string, src string) error {
	// We need these dirs later when walking files
	f.guestUploadDir = dst
	f.hostUploadDir = src

	// Walk all files in the src directory on the host system
	return filepath.Walk(src, f.walkFile)
}

func (f *fileManager) walkFile(hostPath string, hostFileInfo os.FileInfo, err error) error {
	if err == nil && shouldUploadFile(hostFileInfo) {
		relPath := filepath.Dir(hostPath[len(f.hostUploadDir):len(hostPath)])
		guestPath := filepath.Join(f.guestUploadDir, relPath, hostFileInfo.Name())
		hostFile, err := os.Open(hostPath)
		defer hostFile.Close()
		if err != nil {
			log.Printf("Unable to open source file %s for upload: %s", hostPath, err)
			return err
		}
		err = f.UploadFile(guestPath, hostFile)
	}
	return err
}

func (f *fileManager) runCommand(cmd string) error {
	remoteCmd := &packer.RemoteCmd{
		Command: cmd,
	}

	err := f.comm.runCommand(cmd, remoteCmd)
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

type Payload struct {
	Src  string
	Dest string
}
