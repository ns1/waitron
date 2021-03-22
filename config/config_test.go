package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	c, err := LoadConfig("../config.yaml")
	if err != nil {
		t.Errorf("Failed to load test configuration")
	}
	if c.TemplatePath != "templates" {
		t.Errorf("invalid template path")
	}
}

func TestInvalidConfig(t *testing.T) {
	_, err := LoadConfig("invalid.yaml")
	if err == nil {
		t.Errorf("No error presented when invalid configuration is loaded")
	}
}

func TestInvalidYAMLConfig(t *testing.T) {
	_, err := LoadConfig("README.md")
	if err == nil {
		t.Errorf("No error presented when invalid configuration is loaded")
	}
}
