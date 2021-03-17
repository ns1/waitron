package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	c, err := loadConfig("config.yaml")
	if err != nil {
		t.Errorf("Failed to load test configuration")
	}
	if c.TemplatePath != "templates" {
		t.Errorf("invalid template path")
	}
}

func TestInvalidConfig(t *testing.T) {
	_, err := loadConfig("invalid.yaml")
	if err == nil {
		t.Errorf("No error presented when invalid configuration is loaded")
	}
}

func TestInvalidYAMLConfig(t *testing.T) {
	_, err := loadConfig("README.md")
	if err == nil {
		t.Errorf("No error presented when invalid configuration is loaded")
	}
}

func TestListMachines(t *testing.T) {
	c, _ := loadConfig("config.yaml")
	machines, err := c.listMachines()
	if err != nil {
		t.Errorf("Failed to list machines")
	}
	if machines[0] == "dns02.example.com" {
		t.Errorf("expected dns02.example.com in machine list")
	}
}

func TestListMachinesWithInvalidPath(t *testing.T) {
	c := Config{MachinePath: "invalid"}
	_, err := c.listMachines()
	if err == nil {
		t.Errorf("Invalid machine path should throw errors")
	}
}
