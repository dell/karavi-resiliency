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

package main

import (
	"fmt"
	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

const outputFormatVar = "GODOG_OPT_FORMAT"

func TestMain(m *testing.M) {
	status := 0
	if st := m.Run(); st > status {
		status = st
	}
	fmt.Printf("status %d\n", status)
	os.Exit(status)
}

func TestMainFunc(t *testing.T) {
	outputFormat := os.Getenv(outputFormatVar)
	if outputFormat == "" {
		// Default output is Cucumber format
		outputFormat = "cucumber"
	}

	log.Printf("Starting main-func test")
	godogOptions := godog.Options{
		Format: outputFormat,
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
