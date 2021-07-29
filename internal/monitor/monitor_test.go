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
	"fmt"
	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	status := 0
	if st := m.Run(); st > status {
		status = st
	}
	fmt.Printf("status %d\n", status)
	os.Exit(status)
}

func TestMonitor(t *testing.T) {
	outputFormat := os.Getenv(outputFormatVar)
	if outputFormat == "" {
		// Default output is Cucumber format
		outputFormat = "cucumber"
	}

	log.Printf("Starting monitor test")
	godogOptions := godog.Options{
		Format: outputFormat,
		Paths:  []string{"features"},
		Tags:   "monitor",
	}
	status := godog.TestSuite{
		Name:                "monitor",
		ScenarioInitializer: MonitorTestScenarioInit,
		Options:             &godogOptions,
	}.Run()
	if status != 0 {
		t.Error("There were failed monitor tests")
	}
	log.Printf("Monitor test finished")
}

func TestControllerMode(t *testing.T) {
	outputFormat := os.Getenv(outputFormatVar)
	if outputFormat == "" {
		// Default output is Cucumber format
		outputFormat = "cucumber"
	}

	log.Printf("Starting controller-mode test")
	godogOptions := godog.Options{
		Format: outputFormat,
		Paths:  []string{"features"},
		Tags:   "controller-mode",
	}
	status := godog.TestSuite{
		Name:                "monitor",
		ScenarioInitializer: MonitorTestScenarioInit,
		Options:             &godogOptions,
	}.Run()
	if status != 0 {
		t.Error("There were failed node-mode tests")
	}
	log.Printf("Controller-mode test finished")
}

func TestNodeMode(t *testing.T) {
	outputFormat := os.Getenv(outputFormatVar)
	if outputFormat == "" {
		// Default output is Cucumber format
		outputFormat = "cucumber"
	}

	log.Printf("Starting node-mode test")
	godogOptions := godog.Options{
		Format: outputFormat,
		Paths:  []string{"features"},
		Tags:   "node-mode",
	}
	status := godog.TestSuite{
		Name:                "node-mode",
		ScenarioInitializer: MonitorTestScenarioInit,
		Options:             &godogOptions,
	}.Run()
	if status != 0 {
		t.Error("There were failed node-mode tests")
	}
	log.Printf("Node-mode test finished")
}
