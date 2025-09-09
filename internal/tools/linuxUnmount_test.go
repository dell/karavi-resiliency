//go:build test || linux
// +build test linux

/*
 *
 * Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// File: linuxUnmount_test.go

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
