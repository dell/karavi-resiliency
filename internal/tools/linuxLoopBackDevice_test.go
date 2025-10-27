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

// File: linuxLoopBackDevice_test.go
//go:build test || linux
// +build test linux

package tools

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Implement the MockCommander to satisfy the Commander interface
type MockCommander struct {
	output    []byte
	outputErr error
	stdin     io.Reader
}

func (m *MockCommander) Output() ([]byte, error) {
	return m.output, m.outputErr
}

func (m *MockCommander) SetStdin(stdin io.Reader) {
	m.stdin = stdin
}

func TestGetLoopBackDevice(t *testing.T) {
	tests := []struct {
		name          string
		pvname        string
		losetupOutput string
		losetupErr    error
		grepOutput    string
		grepErr       error
		want          string
		expectErr     bool
	}{
		{
			name:          "Valid loopback device",
			pvname:        "test.img",
			losetupOutput: "/dev/loop0: 0 2048 /var/lib/libvirt/images/test.img\n/dev/loop1: 0 2048 /var/lib/libvirt/images/alpine.iso",
			losetupErr:    nil,
			grepOutput:    "/dev/loop0: 0 2048 /var/lib/libvirt/images/test.img",
			grepErr:       nil,
			want:          "/dev/loop0",
			expectErr:     false,
		},
		{
			name:          "Invalid Case",
			pvname:        "test.img",
			losetupOutput: "",
			losetupErr:    nil,
			grepOutput:    "/dev/loop0: 0 2048 /var/lib/libvirt/images/test.img",
			grepErr:       nil,
			want:          "",
			expectErr:     false,
		},
		{
			name:          "Invalid loopback device",
			pvname:        "nonexistent.img",
			losetupOutput: "/dev/loop0: 0 2048 /var/lib/libvirt/images/test.img\n/dev/loop1: 0 2048 /var/lib/libvirt/images/alpine.iso",
			losetupErr:    nil,
			grepOutput:    "",
			grepErr:       errors.New("not found"),
			want:          "",
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCommand = func(name string, _ ...string) Commander {
				switch name {
				case "/usr/sbin/losetup":
					return &MockCommander{
						output:    []byte(tt.losetupOutput),
						outputErr: tt.losetupErr,
					}
				case "grep":
					return &MockCommander{
						output:    []byte(tt.grepOutput),
						outputErr: tt.grepErr,
					}
				default:
					return &MockCommander{}
				}
			}
			got, err := GetLoopBackDevice(tt.pvname)
			if tt.expectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}

	// Reset execCommand to its original setting after tests
	resetExecCommand()
}

func TestDeleteLoopBackDevice(t *testing.T) {
	tests := []struct {
		name      string
		loopDev   string
		output    []byte
		outputErr error
		expectErr bool
	}{
		{
			name:      "Successful deletion of loopback device",
			loopDev:   "/dev/loop0",
			output:    []byte(""),
			outputErr: nil,
			expectErr: false,
		},
		{
			name:      "Error during deletion of loopback device",
			loopDev:   "/dev/loop1",
			output:    nil,
			outputErr: errors.New("error deleting loopback device"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCommand = func(name string, arg ...string) Commander {
				if name == "/usr/sbin/losetup" && len(arg) > 0 && arg[0] == "-d" {
					return &MockCommander{
						output:    tt.output,
						outputErr: tt.outputErr,
					}
				}
				return &MockCommander{}
			}

			got, err := DeleteLoopBackDevice(tt.loopDev)
			if tt.expectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.output, got)
			}
		})
	}

	// Reset execCommand to its original setting after tests
	resetExecCommand()
}

func resetExecCommand() {
	execCommand = func(name string, arg ...string) Commander {
		return &RealCommander{cmd: exec.Command(name, arg...)}
	}
}

func TestRealCommander_Output(t *testing.T) {
	cmd := exec.Command("echo", "test output")
	c := &RealCommander{cmd: cmd}

	want := []byte("test output\n")

	got, err := c.Output()
	if err != nil {
		t.Errorf("RealCommander.Output() error = %v, wantErr nil", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RealCommander.Output() = %v, want %v", got, want)
	}
}

func TestRealCommander_SetStdin(t *testing.T) {
	type fields struct {
		cmd *exec.Cmd
	}
	type args struct {
		stdin io.Reader
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   io.Reader
	}{
		{
			name:   "set stdin to a byte buffer",
			fields: fields{cmd: &exec.Cmd{}},
			args:   args{stdin: bytes.NewBuffer([]byte("test stdin"))},
			want:   bytes.NewBuffer([]byte("test stdin")),
		},
		{
			name:   "set stdin to a string reader",
			fields: fields{cmd: &exec.Cmd{}},
			args:   args{stdin: strings.NewReader("test stdin")},
			want:   strings.NewReader("test stdin"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RealCommander{
				cmd: tt.fields.cmd,
			}
			c.SetStdin(tt.args.stdin)
			got := c.cmd.Stdin
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RealCommander.SetStdin() = %v, want %v", got, tt.want)
			}
		})
	}
}
