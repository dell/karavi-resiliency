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

//  Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"podmon/test/ssh"
	"time"
)

func main() {
	ctx := context.Background()
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

	err := client.Copy(ctx, filepath.Join("C:\\", "workspace", "karavi-resiliency", "test", "sh", "bounce.ip"), "/tmp/bounce.ip")
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
