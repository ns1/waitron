package main

import (
	"github.com/flosch/pongo2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Config is our global configuration file
type Config struct {
	TemplatePath        string
	MachinePath         string
	BaseURL             string
	ForemanProxyAddress string `yaml:"foreman_proxy_address"`
	Params              map[string]string
	Token               map[string]string
	PXEConfig           string `yaml:"pxe_config"`
}

// returns a rendered template based on PXEConfig
func (c Config) getPXEConfig(machine Machine) (string, error) {
	// Load template from config
	tpl, err := pongo2.FromString(c.PXEConfig)
	if err != nil {
		return "", err
	}

	// Format template
	pxeConfig, err := tpl.Execute(pongo2.Context{"machine": machine, "config": c})
	if err != nil {
		return "", err
	}
	return pxeConfig, nil
}

// Loads config.yaml and returns a Config struct
func loadConfig(configPath string) (Config, error) {
	var c Config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}
	yaml.Unmarshal(data, &c)
	return c, nil
}
