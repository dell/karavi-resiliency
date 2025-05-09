/*
* Copyright (c) 2021-2023 Dell Inc., or its subsidiaries. All Rights Reserved.
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
package monitor

import (
	"os"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
)

func TestOcpVirtPowerStoreCheck(t *testing.T) {

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,junit:powerstore-vm-integration-check-junit-report.xml,cucumber:powersctore-short-check-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powerstore-vm-int-setup-check",
		StopOnFailure: stopOnFailure,
	}
	status := godog.TestSuite{
		Name:                "integration",
		ScenarioInitializer: IntegrationTestScenarioInit,
		Options:             &godogOptions,
	}.Run()
	if status != 0 {
		t.Skip("Integration setup check failed")
	} else {
		setupIsGood = true
	}
	log.Printf("Integration setup check finished")
}

func TestOcpVirtPowerFlexCheck(t *testing.T) {

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,junit:powerflex-vm-integration-check-junit-report.xml",
		Paths:         []string{"features"},
		Tags:          "powerflex-vm-int-setup-check",
		StopOnFailure: stopOnFailure,
	}
	status := godog.TestSuite{
		Name:                "integration",
		ScenarioInitializer: IntegrationTestScenarioInit,
		Options:             &godogOptions,
	}.Run()
	if status != 0 {
		t.Skip("Integration setup check failed")
	} else {
		setupIsGood = true
	}
	log.Printf("Integration setup check finished")
}

// func TestOcpVirtPowerScaleCheck(t *testing.T) {

// 	stopOnFailureStr := os.Getenv(enableStopOnFailure)
// 	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
// 		stopOnFailure = false
// 	}
// 	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

// 	godogOptions := godog.Options{
// 		Format:        "pretty,junit:powerscale-vm-integration-check-junit-report.xml,cucumber:powerscale-vm-integration-check-cucumber-report.json",
// 		Paths:         []string{"features"},
// 		Tags:          "powerscale-vm-int-setup-check",
// 		StopOnFailure: stopOnFailure,
// 	}
// 	status := godog.TestSuite{
// 		Name:                "integration",
// 		ScenarioInitializer: IntegrationTestScenarioInit,
// 		Options:             &godogOptions,
// 	}.Run()
// 	if status != 0 {
// 		t.Skip("Integration setup check failed")
// 	} else {
// 		setupIsGood = true
// 	}
// 	log.Printf("Integration setup check finished")
// }

// func TestOcpVirtPowerMaxCheck(t *testing.T) {
// 	stopOnFailureStr := os.Getenv(enableStopOnFailure)
// 	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
// 		stopOnFailure = false
// 	}
// 	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

// 	godogOptions := godog.Options{
// 		Format:        "pretty,junit:powermax-vm-integration-check-junit-report.xml,cucumber:powermax-vm-integration-check-cucumber-report.json",
// 		Paths:         []string{"features"},
// 		Tags:          "powermax-vm-int-setup-check",
// 		StopOnFailure: stopOnFailure,
// 	}
// 	status := godog.TestSuite{
// 		Name:                "integration",
// 		ScenarioInitializer: IntegrationTestScenarioInit,
// 		Options:             &godogOptions,
// 	}.Run()
// 	if status != 0 {
// 		t.Skip("Integration setup check failed")
// 	} else {
// 		setupIsGood = true
// 	}
// 	log.Printf("Integration setup check finished")
// }

func TestOcpVirtPowerStoreIntegration(t *testing.T) {
	if !setupIsGood {
		message := "The setup check failed. Tests skipped"
		log.Print(message)
		t.Error(message)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	log.Printf("Starting integration test")
	godogOptions := godog.Options{
		Format:        "pretty,junit:powerstore-vm-integration-junit-report.xml,cucumber:powerstore-vm-integration-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powerstore-vm-integration",
		StopOnFailure: stopOnFailure,
	}
	status := godog.TestSuite{
		Name:                "virtualization",
		ScenarioInitializer: IntegrationTestScenarioInit,
		Options:             &godogOptions,
	}.Run()
	if status != 0 {
		t.Error("There were failed integration tests")
	}
	log.Printf("Integration test finished")
}

func TestOcpVirtPowerFlexIntegration(t *testing.T) {
	if !setupIsGood {
		message := "The setup check failed. Tests skipped"
		log.Print(message)
		t.Error(message)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	log.Printf("Starting integration test")
	godogOptions := godog.Options{
		Format:        "pretty,junit:powerflex-vm-integration-integration-junit-report.xml",
		Paths:         []string{"features"},
		Tags:          "powerflex-vm-integration",
		StopOnFailure: stopOnFailure,
	}
	status := godog.TestSuite{
		Name:                "integration",
		ScenarioInitializer: IntegrationTestScenarioInit,
		Options:             &godogOptions,
	}.Run()
	if status != 0 {
		t.Error("There were failed integration tests")
	}
	log.Printf("Integration test finished")
}

// func TestOcpVirtPowerScaleIntegration(t *testing.T) {
// 	if !setupIsGood {
// 		message := "The setup check failed. Tests skipped"
// 		log.Print(message)
// 		t.Error(message)
// 		return
// 	}

// 	stopOnFailureStr := os.Getenv(enableStopOnFailure)
// 	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
// 		stopOnFailure = false
// 	}
// 	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

// 	log.Printf("Starting integration test")
// 	godogOptions := godog.Options{
// 		Format:        "pretty,junit:powerscale-vm-integration-junit-report.xml,cucumber:powerscale-vm-integration-cucumber-report.json",
// 		Paths:         []string{"features"},
// 		Tags:          "powerscale-vm-integration",
// 		StopOnFailure: stopOnFailure,
// 	}
// 	status := godog.TestSuite{
// 		Name:                "integration",
// 		ScenarioInitializer: IntegrationTestScenarioInit,
// 		Options:             &godogOptions,
// 	}.Run()
// 	if status != 0 {
// 		t.Error("There were failed integration tests")
// 	}
// 	log.Printf("Integration test finished")
// }

// func TestOcpVirtPowerMaxIntegration(t *testing.T) {
// 	if !setupIsGood {
// 		message := "The setup check failed. Tests skipped"
// 		log.Print(message)
// 		t.Error(message)
// 		return
// 	}

// 	stopOnFailureStr := os.Getenv(enableStopOnFailure)
// 	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
// 		stopOnFailure = false
// 	}
// 	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

// 	log.Printf("Starting integration test")
// 	godogOptions := godog.Options{
// 		Format:        "pretty,junit:powermax-vm-integration-junit-report.xml,cucumber:powermax-vm-integration-cucumber-report.json",
// 		Paths:         []string{"features"},
// 		Tags:          "powermax-vm-integration",
// 		StopOnFailure: stopOnFailure,
// 	}
// 	status := godog.TestSuite{
// 		Name:                "integration",
// 		ScenarioInitializer: IntegrationTestScenarioInit,
// 		Options:             &godogOptions,
// 	}.Run()
// 	if status != 0 {
// 		t.Error("There were failed integration tests")
// 	}
// 	log.Printf("Integration test finished")
// }
