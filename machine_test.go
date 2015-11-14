package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestmachineDefinition(t *testing.T) {
	m, err := machineDefinition("my-service.example.com", "machines")
	if err != nil {
		t.Errorf("Unable to load test machine definition")
	}
	if m.Hostname != "my-service.example.com" {
		t.Errorf("expected hostname: my-service.example.com")
	}
	if m.ShortName != "my-service" {
		t.Errorf("invalid shortname")
	}
}

func TestRenderTemplate(t *testing.T) {
	config, _ := loadConfig("config.yaml")
	m, _ := machineDefinition("my-service.example.com", "machines")

	template, err := m.renderTemplate("finish.j2", config)
	if err != nil {
		t.Errorf("failed to render template")
	}

	expected := "example.com"
	if !strings.Contains(template, expected) {
		t.Errorf(fmt.Sprintf("Template does not contain '%s'", expected))
	}
}
