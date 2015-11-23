package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"
)

// Config is our global configuration file
type Config struct {
	TemplatePath        string
	MachinePath         string
	BaseURL             string
	ForemanProxyAddress string `yaml:"foreman_proxy_address"`
	DefaultCmdline      string `yaml:"default_cmdline"`
	DefaultKernel       string `yaml:"default_kernel"`
	DefaultInitrd       string `yaml:"default_initrd"`
	DefaultImageURL     string `yaml:"default_image_url"`
	Params              map[string]string
	Tokens              map[string]string
	MachineState        map[string]string
	MachineBuild        map[string]string
}

// Loads config.yaml and returns a Config struct
func loadConfig(configPath string) (Config, error) {
	var c Config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return Config{}, err
	}

	// Initialize map containing hostname[token]
	c.Tokens = make(map[string]string)
	c.MachineState = make(map[string]string)
	c.MachineBuild = make(map[string]string)
	return c, nil
}

func (c Config) listMachines() ([]string, error) {
	var machines []string
	files, err := ioutil.ReadDir(c.MachinePath)
	for _, file := range files {
		name := file.Name()
		if path.Ext(name) == ".yaml" {
			machines = append(machines, name)
		}
	}
	if err != nil {
		return machines, err
	}
	return machines, nil
}
