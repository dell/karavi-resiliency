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

func TestMonitor(t *testing.T) {
	log.Printf("Starting monitor test")
	godogOptions := godog.Options{
		Format: "pretty,junit:monitor-junit-report.xml,cucumber:monitor-cucumber-report.json",
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
	log.Printf("Starting controller-mode test")
	godogOptions := godog.Options{
		Format: "pretty,junit:controller-mode-junit-report.xml,cucumber:controller-mode-cucumber-report.json",
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
	log.Printf("Starting node-mode test")
	godogOptions := godog.Options{
		Format: "pretty,junit:node-mode-junit-report.xml,cucumber:node-mode-cucumber-report.json",
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

func TestMapEqualsMap(t *testing.T) {
	cases := []struct {
		mapA, mapB map[string]string
		result     bool
	}{
		{
			map[string]string{"key": "value"},
			map[string]string{"key": "value"},
			true,
		},
		{
			map[string]string{"key": "value"},
			map[string]string{"value": "key"},
			false,
		},
		{
			map[string]string{"key": "value"},
			map[string]string{},
			false,
		},
	}
	for caseNum, acase := range cases {
		result := mapEqualsMap(acase.mapA, acase.mapB)
		if result != acase.result {
			t.Errorf("Case %d: Expected %t got %t", caseNum, acase.result, result)
		}
	}
}
