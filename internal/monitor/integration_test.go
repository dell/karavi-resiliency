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

const enableIntTestVar = "RESILIENCY_INT_TEST"

var setupIsGood = false

func TestFirstCheck(t *testing.T) {
	intTestEnvVarStr := os.Getenv(enableIntTestVar)
	if intTestEnvVarStr == "" || strings.ToLower(intTestEnvVarStr) != "true" {
		log.Printf("Skipping integration test. To enable integration test: export %s=true", enableIntTestVar)
		return
	}

	godogOptions := godog.Options{
		Format: "pretty",
		Paths:  []string{"features"},
		Tags:   "int-setup-check",
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

func TestIntegration(t *testing.T) {
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

	log.Printf("Starting integration test")
	godogOptions := godog.Options{
		Format: "pretty",
		Paths:  []string{"features"},
		Tags:   "integration",
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
