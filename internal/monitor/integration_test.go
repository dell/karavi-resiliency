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

package monitor

import (
	"os"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
)

const enableIntTestVar = "RESILIENCY_INT_TEST"
const enableStopOnFailure = "RESILIENCY_INT_TEST_STOP_ON_FAILURE"
const outputFormatVar = "GODOG_OPT_FORMAT"

var setupIsGood = false

// stopOnFailure enabled means any failed tests would stop the tests (default: true)
var stopOnFailure = true

func TestPowerFlexFirstCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableIntTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,junit:powerflex-first-check-junit-report.xml,cucumber:powerflex-first-check-cucumber-report.json",
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

func TestUnityFirstCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableIntTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,junit:unity-first-check-junit-report.xml,cucumber:unity-first-check-cucumber-report.json",
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

func TestPowerScaleFirstCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableIntTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty,cucumber:powerscale-first-check-cucumber-report.json",
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

func TestPowerFlexIntegration(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableIntTestVar)
		return
	}

	if !setupIsGood {
		message := "The setup check failed. Tests skipped"
		log.Printf(message)
		t.Errorf(message)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	log.Printf("Starting integration test")
	godogOptions := godog.Options{
		Format:        "pretty,junit:powerflex-integration-junit-report.xml,cucumber:powerflex-integration-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powerflex-integration",
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

func TestUnityIntegration(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableIntTestVar)
		return
	}

	if !setupIsGood {
		message := "The setup check failed. Tests skipped"
		log.Printf(message)
		t.Errorf(message)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	log.Printf("Starting integration test")
	godogOptions := godog.Options{
		Format:        "pretty,junit:unity-integration-junit-report.xml,cucumber:unity-integration-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "unity-integration",
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

func TestPowerScaleIntegration(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableIntTestVar)
		return
	}

	if !setupIsGood {
		message := "The setup check failed. Tests skipped"
		log.Printf(message)
		t.Errorf(message)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	log.Printf("Starting integration test")
	godogOptions := godog.Options{
		Format:        "pretty,cucumber:powerscale-integration-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powerscale-integration",
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
func TestPowerflexArrayInterfaceDown(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableIntTestVar)
		return
	}

	if !setupIsGood {
		message := "The setup check failed. Tests skipped"
		log.Printf(message)
		t.Errorf(message)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	log.Printf("Starting integration test")
	godogOptions := godog.Options{
		Format:        "pretty,junit:powerflex-interface-down-junit-report.xml,cucumber:powerflex-interface-down-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "powerflex-array-interface",
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

func TestUnityArrayInterfaceDown(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableIntTestVar)
		return
	}

	if !setupIsGood {
		message := "The setup check failed. Tests skipped"
		log.Printf(message)
		t.Errorf(message)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	log.Printf("Starting integration test")
	godogOptions := godog.Options{
		Format:        "pretty,junit:unity-interface-down-junit-report.xml,cucumber:unity-interface-down-cucumber-report.json",
		Paths:         []string{"features"},
		Tags:          "unity-array-interface",
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
