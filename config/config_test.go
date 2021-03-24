package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	_, err := LoadConfig("../examples/config.yml")
	if err != nil {
		t.Errorf("Failed to load test configuration")
	}
}

func TestInvalidConfig(t *testing.T) {
	_, err := LoadConfig("invalid.yml")
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
