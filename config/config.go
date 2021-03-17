package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type BuildCommand struct {
	Command        string
	TimeoutSeconds int  `yaml:"timeout_seconds"`
	ErrorsFatal    bool `yaml:"errors_fatal"`
	ShouldLog      bool `yaml:"should_log"`
}

type BuildType struct {
	Cmdline  string `yaml:"cmdline"`
	Kernel   string `yaml:"kernel"`
	Initrd   string `yaml:"initrd"`
	ImageURL string `yaml:"image_url"`

	OperatingSystem string
	Finish          string
	Preseed         string
	Params          map[string]string

	StaleBuildCommands  []BuildCommand `yaml:"stalebuild_commands"`
	PreBuildCommands    []BuildCommand `yaml:"prebuild_commands"`
	PostBuildCommands   []BuildCommand `yaml:"postbuild_commands"`
	CancelBuildCommands []BuildCommand `yaml:"cancelbuild_commands"`
}

type MachineInventoryPluginSettings struct {
	Name             string                 `yaml:"name"`
	Type             string                 `yaml:"type"`
	Source           string                 `yaml:"source"`
	AuthUser         string                 `yaml:"auth_username"`
	AuthPassword     string                 `yaml:"auth_password"`
	AuthToken        string                 `yaml:"auth_token"`
	AdditionalOption map[string]interface{} `yaml:"additional_options"`
}

// Config is our global configuration file
type Config struct {
	TemplatePath    string
	GroupPath       string
	StaticFilesPath string `yaml:"staticspath"`
	BaseURL         string

	MachineInventoryPlugins    []MachineInventoryPluginSettings `yaml:"inventory_plugins,omitempty"`
	BuildTypes                 map[string]BuildType             `yaml:"build_types,omitempty"`
	StaleBuildCheckFrequency   int                              `yaml:"stale_build_check_frequency_secs"`
	StaleBuildThresholdSeconds int                              `yaml:"stale_build_threshold_secs"`

	BuildType `yaml:",inline"`
}

// Loads config.yaml and returns a Config struct
func LoadConfig(configPath string) (Config, error) {

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
