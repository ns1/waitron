package main

import (
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
