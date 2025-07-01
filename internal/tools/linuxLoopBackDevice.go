//go:build test || linux
// +build test linux

/*
* Copyright (c) 2022 Dell Inc., or its subsidiaries. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package tools

import (
	"bytes"
	"io"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Define the Commander interface
type Commander interface {
	Output() ([]byte, error)
	SetStdin(io.Reader)
}

type RealCommander struct {
	cmd *exec.Cmd
}

func (c *RealCommander) Output() ([]byte, error) {
	return c.cmd.Output()
}

func (c *RealCommander) SetStdin(stdin io.Reader) {
	c.cmd.Stdin = stdin
}

// Create an execCommand function to return the wrapped exec.Cmd
var execCommand = func(name string, arg ...string) Commander {
	return &RealCommander{cmd: exec.Command(name, arg...)}
}

// GetLoopBackDevice get the loopbackdevice for given pv
func GetLoopBackDevice(pvname string) (string, error) {
	textBytes, err := execCommand("/usr/sbin/losetup", "-a").Output()
	if err != nil || string(textBytes) == "" {
		return "", err
	}

	cmd := execCommand("grep", pvname)
	cmd.SetStdin(bytes.NewBuffer(textBytes))
	textBytes, err = cmd.Output()
	if err != nil || string(textBytes) == "" {
		return "", err
	}
	log.Debugf("losetup output: %s", string(textBytes))
	loopDevices := strings.Split(string(textBytes), ":")
	return loopDevices[0], nil
}

// DeleteLoopBackDevice deletes the loopbackdevice from the system
func DeleteLoopBackDevice(loopDev string) ([]byte, error) {
	cmd := execCommand("/usr/sbin/losetup", "-d", loopDev)
	return cmd.Output()
}
