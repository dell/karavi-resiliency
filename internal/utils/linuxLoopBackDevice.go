//go:build test || linux
// +build test linux

/*
 * Copyright (c) 2021. Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 */

package utils

import (
	"bytes"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	execCommand = exec.Command
)

func GetLoopBackDevice(pvname string) (string, error) {
	textBytes, err := execCommand("/usr/sbin/losetup", "-a").Output()
	if err != nil || string(textBytes) == "" {
		return "", err
	}

	cmd := execCommand("grep", pvname)
	cmd.Stdin = bytes.NewBuffer(textBytes)
	textBytes, err = cmd.Output()
	if err != nil || string(textBytes) == "" {
		return "", err
	}
	log.Debugf("losetup output: %s", string(textBytes))
	loopDevices := strings.Split(string(textBytes), ":")
	return loopDevices[0], nil
}

func DeleteLoopBackDevice(loopDev string) ([]byte, error) {
	cmd := execCommand("/usr/sbin/losetup", "-d", loopDev)
	return cmd.Output()
}
