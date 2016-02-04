package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"
)

// Config is our global configuration file
type Config struct {
	TemplatePath    string
	MachinePath     string
	HookPath        string
	BaseURL         string
	DefaultCmdline  string `yaml:"default_cmdline"`
	DefaultKernel   string `yaml:"default_kernel"`
	DefaultInitrd   string `yaml:"default_initrd"`
	DefaultImageURL string `yaml:"default_image_url"`
	Params          map[string]string
	Tokens          map[string]string
	MachineState    map[string]string
	MachineBuild    map[string]string
	PreHooks        []string `yaml:"pre_hooks"`
	PostHooks       []string `yaml:"post_hooks"`
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

func (c Config) listHooks() ([]string, error) {
	var hooks []string
	files, err := ioutil.ReadDir(c.HookPath)
	for _, file := range files {
		name := file.Name()
		if path.Ext(name) == ".sh" {
			hooks = append(hooks, name)
		}
	}
	if err != nil {
		return hooks, err
	}
	return hooks, nil
}
