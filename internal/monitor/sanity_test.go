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

func TestPowerFlexSanityCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping short integration test. To enable short integration test: export %s=true", enableSanityTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,junit:powerflex-sanity-check-junit-report.xml",
		Paths:         []string{"features"},
		Tags:          "powerflex-int-setup-check",
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

func TestUnitySanityCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping short integration test. To enable short integration test: export %s=true", enableSanityTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,junit:unity-sanity-check-junit-report.xml",
		Paths:         []string{"features"},
		Tags:          "unity-int-setup-check",
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

func TestPowerScaleSanityCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping short integration test. To enable short integration test: export %s=true", enableSanityTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,junit:powerscale-sanity-check-junit-report.xml,cucumber:powerscale-sanity-check-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powerscale-int-setup-check",
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

func TestPowerStoreSanityCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping short integration test. To enable short integration test: export %s=true", enableSanityTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,junit:powerstore-sanity-check-junit-report.xml,cucumber:powersctore-sanity-check-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powerstore-int-setup-check",
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

func TestPowerMaxSanityCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping short integration test. To enable short integration test: export %s=true", enableSanityTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,junit:powermax-sanity-check-junit-report.xml,cucumber:powermax-sanity-check-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powermax-int-setup-check",
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
	log.Printf("Integration test finished")
}

func TestPowerFlexSanityTest(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableSanityTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableSanityTestVar)
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
		Format:        "pretty,junit:powerflex-sanity-integration-junit-report.xml",
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
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableSanityTestVar)
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
		Format:        "pretty,junit:unity-sanity-integration-junit-report.xml",
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
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableSanityTestVar)
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
		Format:        "pretty,junit:powerscale-sanity-integration-junit-report.xml,cucumber:powerscale-sanity-integration-cucumber-report.json",
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
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableSanityTestVar)
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
		Format:        "pretty,junit:powerstore-sanity-integration-junit-report.xml,cucumber:powerstore-sanity-integration-cucumber-report.json",
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
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableSanityTestVar)
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
		Format:        "pretty,junit:powermax-sanity-integration-junit-report.xml,cucumber:powermax-sanity-integration-cucumber-report.json",
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
