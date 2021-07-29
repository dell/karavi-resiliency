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
	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"testing"
)

const enableShortIntTestVar = "RESILIENCY_SHORT_INT_TEST"

func TestPowerFlexShortCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableShortIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping short integration test. To enable short integration test: export %s=true", enableShortIntTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        "pretty",
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

func TestUnityShortCheck(t *testing.T) {
	outputFormat := os.Getenv(outputFormatVar)
	if outputFormat == "" {
		// Default output is Cucumber format
		outputFormat = "cucumber"
	}

	intTestEnvVarStr := os.Getenv(enableShortIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping short integration test. To enable short integration test: export %s=true", enableShortIntTestVar)
		return
	}

	stopOnFailureStr := os.Getenv(enableStopOnFailure)
	if stopOnFailureStr != "" && strings.ToLower(stopOnFailureStr) == "false" {
		stopOnFailure = false
	}
	log.Printf("%s = %v", enableStopOnFailure, stopOnFailure)

	godogOptions := godog.Options{
		Format:        outputFormat,
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

func TestPowerFlexShortIntegration(t *testing.T) {
	outputFormat := os.Getenv(outputFormatVar)
	if outputFormat == "" {
		// Default output is Cucumber format
		outputFormat = "cucumber"
	}

	intTestEnvVarStr := os.Getenv(enableShortIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableShortIntTestVar)
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
		Format:        outputFormat,
		Paths:         []string{"features"},
		Tags:          "powerflex-short-integration",
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

func TestUnityShortIntegration(t *testing.T) {
	outputFormat := os.Getenv(outputFormatVar)
	if outputFormat == "" {
		// Default output is Cucumber format
		outputFormat = "cucumber"
	}

	intTestEnvVarStr := os.Getenv(enableShortIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableShortIntTestVar)
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
		Format:        outputFormat,
		Paths:         []string{"features"},
		Tags:          "unity-short-integration",
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
