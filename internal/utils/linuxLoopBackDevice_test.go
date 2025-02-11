// File: linuxLoopBackDevice_test.go
//go:build test || linux
// +build test linux

package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Mock exec.Command for testing purposes
var (
	execCommandBackup = execCommand
)

func testHelperProcessSuccess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	cmdArgs := strings.Join(os.Args[3:], " ")
	switch {
	case strings.Contains(cmdArgs, "/usr/sbin/losetup -a"):
		fmt.Fprint(os.Stdout, "/dev/loop1: 0 /dev/sda")
	case strings.Contains(cmdArgs, "grep"):
		textBytes, _ := ioutil.ReadAll(os.Stdin)
		if bytes.Contains(textBytes, []byte("mypv")) {
			fmt.Fprint(os.Stdout, "/dev/loop1")
		}
	case strings.Contains(cmdArgs, "/usr/sbin/losetup -d"):
		fmt.Fprint(os.Stdout, "deleted\n")
	default:
		fmt.Fprint(os.Stderr, "unexpected command")
	}
	os.Exit(0)
}

func testHelperProcessFailure(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	cmdArgs := strings.Join(os.Args[3:], " ")
	switch {
	case strings.Contains(cmdArgs, "/usr/sbin/losetup -a"):
		fmt.Fprint(os.Stdout, "")
	case strings.Contains(cmdArgs, "grep"):
		textBytes, _ := ioutil.ReadAll(os.Stdin)
		if bytes.Contains(textBytes, []byte("mypv")) {
			fmt.Fprint(os.Stderr, "error identifying loopback device")
			os.Exit(1)
		}
	case strings.Contains(cmdArgs, "/usr/sbin/losetup -d"):
		fmt.Fprint(os.Stderr, "failed to delete loopback device")
		os.Exit(1)
	default:
		fmt.Fprint(os.Stderr, "unexpected command")
		os.Exit(1)
	}
	os.Exit(1)
}

func TestHelperProcessSuccess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	testHelperProcessSuccess(nil)
}

func TestHelperProcessFailure(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	testHelperProcessFailure(nil)
}

func TestGetLoopBackDevice_Success(t *testing.T) {
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcessSuccess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}
	defer func() { execCommand = execCommandBackup }()

	_, err := GetLoopBackDevice("mypv")
	if err != nil {
		t.Fatalf("GetLoopBackDevice() failed unexpectedly: %v", err)
	}
}

func TestGetLoopBackDevice_Failure(t *testing.T) {
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcessFailure", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}
	defer func() { execCommand = execCommandBackup }()

	_, err := GetLoopBackDevice("mypv")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteLoopBackDevice_Success(t *testing.T) {
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcessSuccess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}
	defer func() { execCommand = execCommandBackup }()

	out, err := DeleteLoopBackDevice("/dev/loop1")
	if err != nil {
		t.Fatalf("DeleteLoopBackDevice() failed unexpectedly: %v", err)
	}
	expectedOutput := "deleted\n"
	if string(out) != expectedOutput {
		t.Fatalf("expected %s, got %s", expectedOutput, string(out))
	}
}

func TestDeleteLoopBackDevice_Failure(t *testing.T) {
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcessFailure", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}
	defer func() { execCommand = execCommandBackup }()

	_, err := DeleteLoopBackDevice("/dev/loop1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
