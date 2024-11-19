//  Copyright © 2021-2022 Dell Inc. or its subsidiaries. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//       http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	status := 0
	if st := m.Run(); st > status {
		status = st
	}
	fmt.Printf("status %d\n", status)
	os.Exit(status)
}

func TestMainFunc(t *testing.T) {
	log.Printf("Starting main-func test")
	godogOptions := godog.Options{
		Format: "pretty,junit:main-func-junit-report.xml",
		Paths:  []string{"features"},
	}
	status := godog.TestSuite{
		Name:                "main-func",
		ScenarioInitializer: ScenarioInit,
		Options:             &godogOptions,
	}.Run()
	if status != 0 {
		t.Error("There were failed main-func tests")
	}
	log.Printf("Main-func test finished")
}
