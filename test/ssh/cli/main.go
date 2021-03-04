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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"podmon/test/ssh"
	"time"
)

func main() {
	info := ssh.AccessInfo{
		Hostname: os.Getenv("HOST"),
		Port:     "22",
		Username: os.Getenv("USER"),
		Password: os.Getenv("PASSWORD"),
	}

	wrapper := ssh.NewWrapper(&info)

	client := ssh.CommandExecution{
		AccessInfo: &info,
		SSHWrapper: wrapper,
		Timeout:    4 * time.Second,
	}

	if err := client.Run("date; ls -ltr /tmp"); err == nil {
		for _, out := range client.GetOutput() {
			fmt.Printf("%s\n", out)
		}
	} else {
		fmt.Printf("ERROR %s\n", err)
	}

	err := client.Copy(filepath.Join("C:\\", "workspace", "karavi-resiliency", "test", "sh", "bounce.ip"), "/tmp/bounce.ip")
	if err != nil {
		fmt.Printf("ERROR %v", err)
	}

	if err := client.Run("date; ls -ltr /tmp; cat /tmp/bounce.ip; rm -f /tmp/bounce.ip"); err == nil {
		for _, out := range client.GetOutput() {
			fmt.Printf("%s\n", out)
		}
	} else {
		fmt.Printf("ERROR %s\n", err)
	}
}
