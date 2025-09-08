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

//go:build test || windows
// +build test windows

package tools

import (
	"os"
	"testing"
)

func setupTempFile() (string, error) {
	// Create a temporary file to simulate a device
	tempFile, err := os.CreateTemp("", "testdevice")
	if err != nil {
		return "", err
	}
	tempFileName := tempFile.Name()
	tempFile.Close()
	return tempFileName, nil
}

func teardownTempFile(fileName string) {
	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		// Clean up by removing the file
		os.Remove(fileName)
	}
}

// Ensure the test runs only on Windows (or with the test build tag)
func TestUnmount(t *testing.T) {
	tempFileName, err := setupTempFile()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer teardownTempFile(tempFileName)

	// Attempt to unmount the simulated device (temporary file)
	if err := Unmount(tempFileName, 0); err != nil {
		t.Fatalf("Unmount() failed: %v", err)
	}

	// Verify that the file has been removed
	if _, err := os.Stat(tempFileName); !os.IsNotExist(err) {
		t.Fatalf("File %s was not removed", tempFileName)
	}
}

// Ensure the test runs only on Windows (or with the test build tag)
func TestCreat(t *testing.T) {
	tempDir := os.TempDir()
	testFilePath := tempDir + "/testcreatefile.txt"

	// Ensure the file does not already exist
	os.Remove(testFilePath)
	defer os.Remove(testFilePath)

	// Attempt to create the file
	fd, err := Creat(testFilePath, 0)
	if err != nil {
		t.Fatalf("Creat() failed: %v", err)
	} else if fd < 0 {
		t.Fatalf("Creat() returned invalid file descriptor: %d", fd)
	}

	// Verify that the file has been created
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Fatalf("File %s was not created", testFilePath)
	}
}
