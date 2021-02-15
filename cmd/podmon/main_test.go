package main

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

func TestMainFunc(t *testing.T) {
	log.Printf("Starting main-func test")
	godogOptions := godog.Options{
		Format: "pretty",
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