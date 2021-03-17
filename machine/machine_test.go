package machine

import (
	"fmt"
	"strings"
	"testing"
)

func TestmachineDefinition(t *testing.T) {
	config, _ := loadConfig("config.yaml")
	m, err := machineDefinition("dns02.example.com", "machines", config)

	if err != nil {
		t.Errorf("Unable to load test machine definition")
	}
	if m.Hostname != "dns02.example.com" {
		t.Errorf("expected hostname: dns02.example.com")
	}
	if m.ShortName != "dns02" {
		t.Errorf("invalid shortname")
	}
}

func TestRenderTemplate(t *testing.T) {
	config, _ := loadConfig("config.yaml")
	m, _ := machineDefinition("dns02.example.com", "machines", Config{})

	template, err := m.renderTemplate("finish.j2", config)
	if err != nil {
		t.Errorf("failed to render template")
	}

	expected := "example.com"
	if !strings.Contains(template, expected) {
		t.Errorf(fmt.Sprintf("Template does not contain '%s'", expected))
	}
}

func TestRenderTemplateNotFound(t *testing.T) {
	config, _ := loadConfig("config.yaml")
	m, _ := machineDefinition("dns02.example.com", "machines", Config{})
	_, err := m.renderTemplate("invalid.j2", config)

	expected := "Template does not exist"
	if err.Error() != expected {
		t.Errorf(fmt.Sprintf("Expected: %s, got: %s", expected, err.Error()))
	}
}
