// File: linuxUnmount_test.go
//go:build test || linux

package tools

import (
	"os"
	"syscall"
	"testing"
)

// setupTestFile creates a test file and returns its name and a cleanup function
func setupTestFile(t *testing.T) (string, func()) {
	t.Helper() // Marks this function as a test helper

	tempFile, err := os.CreateTemp("", "testdevice")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFileName := tempFile.Name()
	tempFile.Close()

	// Cleanup function to remove the test file
	cleanup := func() {
		os.Remove(tempFileName)
	}

	return tempFileName, cleanup
}

// TestUnmount verifies Unmount wrapper
func TestUnmount(t *testing.T) {
	tempFile, cleanup := setupTestFile(t)
	defer cleanup()

	// Attempt to unmount the temporary file
	err := Unmount(tempFile, 0)
	if err != nil {
		t.Log("Unmount failed with a known invalid argument for simply using a file as device-like")
	} else {
		t.Fatalf("Unmount expected to fail on regular file. Possible invalid test argument.")
	}
}

// TestCreat verifies Creat wrapper
func TestCreat(t *testing.T) {
	tempDir := os.TempDir()
	testFilePath := tempDir + "/testcreatefile.txt"

	// Ensure the file does not already exist
	os.Remove(testFilePath)
	defer os.Remove(testFilePath)

	// Use syscall.Creat as per syscall requirement
	fd, err := Creat(testFilePath, 0)
	if err != nil {
		t.Fatalf("Creat() failed: %v", err)
	}
	defer syscall.Close(fd) // Clean up file descriptor after the test

	if fd < 0 {
		t.Fatalf("Creat() returned invalid file descriptor: %d", fd)
	}

	// Verify that the file has been created
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Fatalf("File %s was not created", testFilePath)
	}
}
