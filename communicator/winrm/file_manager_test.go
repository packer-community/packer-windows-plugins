package winrm

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestTempFile(t *testing.T) {
	comm := defaultCommunicator()
	fm := &fileManager{comm: comm}
	tempString := "Temp for packer"
	var output *os.File
	var input *os.File
	defer func() {
		// Close and delete tmp files
		input.Close()
		output.Close()
		os.Remove(input.Name())
		os.Remove(output.Name())
	}()

	input, err := ioutil.TempFile("/tmp", "packer-test-tmp")
	fmt.Printf("Input name: %s", input.Name())
	input.WriteString(tempString)
	if err != nil {
		t.Fatalf("Unable to create tmp file for test: %s", err)
	}
	f, err := os.Open(input.Name())
	output, err = fm.TempFile(f)
	fmt.Printf("Output name: %s", output.Name())

	if err != nil {
		t.Fatalf("Unable to create tmp file for test: %s", err)
	}

	data, err := ioutil.ReadFile(output.Name())
	dataString := string(data[0:15])
	if dataString != tempString {
		t.Fatalf("File contents should equal \"%s\". Actual: \"%s\"", tempString, dataString)
	}
}

func TestWinFriendlyPath(t *testing.T) {
	in := "c:/foo/bar/baz"
	out := winFriendlyPath(in)
	if out != "c:\\foo\\bar\\baz" {
		t.Fatalf("Path should be %s", out)
	}
	log.Printf("Winfriendly: %s", out)

	in = "C:/Windows/Temp/DSC\\Index_files"
	out = winFriendlyPath(in)
	if out != "C:\\Windows\\Temp\\DSC\\Index_files" {
		t.Fatalf("Path should be %s but is %s", "C:\\Windows\\Temp\\DSC\\Index_files", out)
	}
	log.Printf("Winfriendly: %s", out)
}

func TestPrepareFileDirectory(t *testing.T) {
	comm := new(MockWinRMCommunicator)
	fm, err := NewFileManager(comm)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}
	comm.expectedCommand = `powershell -Command "& { md -Force $([System.IO.Path]::GetFullPath('/foo')) }"`
	err = fm.prepareFileDirectory("/foo")
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}

}

// TODO: Test relative paths

func TestUploadDir_Error(t *testing.T) {
	comm := new(MockWinRMCommunicatorWithErrors)

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))

	fm, err := NewFileManager(comm)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}

	err = fm.UploadDir("c:\\windows\\temp", dir)
	if err == nil {
		t.Fatalf("Should have error")
	}
}

func TestUploadDir_Success(t *testing.T) {
	comm := new(MockWinRMCommunicator)

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))

	fm, err := NewFileManager(comm)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}

	// Upload
	err = fm.UploadDir("c:\\windows\\temp", dir)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}
}

func TestUpload_Success(t *testing.T) {
	comm := new(MockWinRMCommunicator)

	fileName, _ := filepath.Abs(os.Args[0])
	file, err := os.Open(fileName)

	fm, err := NewFileManager(comm)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}

	// Upload
	err = fm.Upload("c:\\windows\\temp\\filename", file)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}
}
func TestUpload_Fail(t *testing.T) {
	comm := new(MockWinRMCommunicatorWithErrors)

	fileName, _ := filepath.Abs(os.Args[0])
	file, err := os.Open(fileName)

	fm, err := NewFileManager(comm)
	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}

	// Upload
	err = fm.Upload("c:\\windows\\temp\\filename", file)
	if err == nil {
		t.Fatalf("Should have error: %s", err)
	}
}

func TestShouldUploadFile(t *testing.T) {
	hostFileInfo := new(MockFileInfo)
	hostFileInfo.isDir = true
	if shouldUploadFile(hostFileInfo) != false {
		t.Fatalf("Expected sholudUploadFile to be false when uploading a dir")
	}
	hostFileInfo.isDir = false
	hostFileInfo.name = "foo"
	if shouldUploadFile(hostFileInfo) == false {
		t.Fatalf("Expected sholudUploadFile to be true when uploading a file")
	}
}

func TestUploadFileWalker_Fail(t *testing.T) {
	comm := new(MockWinRMCommunicator)
	fm := &fileManager{comm: comm}
	tempString := "foobar"
	input, err := ioutil.TempFile("/tmp", "packer-test-tmp")
	fmt.Printf("Input name: %s", input.Name())
	input.WriteString(tempString)
	if err != nil {
		t.Fatalf("Unable to create tmp file for test: %s", err)
	}
	hostFileInfo := new(MockFileInfo)
	hostFileInfo.name = "isurelymustnotexistforthisisasillyname"

	// Upload a file that should not exist
	err = fm.uploadFileWalker(hostFileInfo.Name(), hostFileInfo, nil)
	if err == nil {
		t.Fatalf("Should have error. %s should not exist", hostFileInfo.Name())
	}
}
func TestUploadFileWalker(t *testing.T) {
	comm := new(MockWinRMCommunicator)
	fm := &fileManager{comm: comm}
	tempString := "foobar"
	input, err := ioutil.TempFile("/tmp", "packer-test-tmp")
	fmt.Printf("Input name: %s", input.Name())
	input.WriteString(tempString)
	if err != nil {
		t.Fatalf("Unable to create tmp file for test: %s", err)
	}
	hostFileInfo := new(MockFileInfo)
	hostFileInfo.isDir = true

	// Upload a temporary file
	err = fm.uploadFileWalker(input.Name(), hostFileInfo, nil)

	if err != nil {
		t.Fatalf("Should not have error: %s", err)
	}
}
