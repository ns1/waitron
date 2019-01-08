package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"
)

// Config is our global configuration file
type State struct {
	Tokens              map[string]string
	MachineState        map[string]string
	MachineBuild        map[string]string
}

type BuildCommand struct {
	Command			string
	TimeoutSeconds	int `yaml:"timeout_seconds"`
	ErrorsFatal		bool `yaml:"errors_fatal"`
	ShouldLog		bool `yaml:"should_log"`
}

type Config struct {
	TemplatePath        string
	GroupPath			string
	MachinePath         string
	BaseURL             string
	ForemanProxyAddress string `yaml:"foreman_proxy_address"`
	Cmdline      string `yaml:"cmdline"`
	Kernel       string `yaml:"kernel"`
	Initrd       string `yaml:"initrd"`
	ImageURL     string `yaml:"image_url"`
	OperatingSystem string
	Finish          string
	Preseed         string
	Params              map[string]string
	PreBuildCommands	[]BuildCommand `yaml:"prebuild_commands"`
	PostBuildCommands	[]BuildCommand `yaml:"postbuild_commands"`
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

	return c, nil
}


func loadState() (State) {
	var s State
	// Initialize map containing hostname[token]
	s.Tokens = make(map[string]string)
	s.MachineState = make(map[string]string)
	s.MachineBuild = make(map[string]string)
	return s
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
