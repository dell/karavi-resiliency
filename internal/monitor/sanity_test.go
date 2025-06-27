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

const enableSanityTestVar = "RESILIENCY_SANITY_TEST"

func TestPowerFlexSanityTest(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableShortIntTestVar)
		return
	}

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
		Format:        "pretty,junit:powerflex-short-integration-junit-report.xml",
		Paths:         []string{"features"},
		Tags:          "powerflex-sanity-test",
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

func TestUnitySanityTest(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableShortIntTestVar)
		return
	}

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
		Format:        "pretty,junit:unity-short-integration-junit-report.xml",
		Paths:         []string{"features"},
		Tags:          "unity-sanity-test",
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

func TestPowerScaleSanityTest(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableShortIntTestVar)
		return
	}

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
		Format:        "pretty,junit:powerscale-short-integration-junit-report.xml,cucumber:powerscale-short-integration-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powerscale-sanity-test",
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

func TestPowerStoreSanityTest(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableShortIntTestVar)
		return
	}

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
		Format:        "pretty,junit:powerstore-short-integration-junit-report.xml,cucumber:powerstore-short-integration-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powerstore-sanity-test",
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

func TestPowerMaxSanitytest(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableShortIntTestVar)
		return
	}

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
		Format:        "pretty,junit:powermax-short-integration-junit-report.xml,cucumber:powermax-short-integration-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powermax-sanity-test",
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
