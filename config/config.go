package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarning
	LogLevelInfo
	LogLevelDebug
)

func (l LogLevel) String() string {
	return [...]string{"ERROR", "WARN", "INFO", "DEBUG"}[l]
}

type BuildCommand struct {
	Command        string
	TimeoutSeconds int  `yaml:"timeout_seconds"`
	ErrorsFatal    bool `yaml:"errors_fatal"`
	ShouldLog      bool `yaml:"should_log"`
}

type BuildType struct {
	Cmdline  string `yaml:"cmdline,omitempty"`
	Kernel   string `yaml:"kernel,omitempty"`
	Initrd   string `yaml:"initrd,omitempty"`
	ImageURL string `yaml:"image_url,omitempty"`

	OperatingSystem string `yaml:"operatingsystem,omitempty"`
	Finish          string `yaml:"finish,omitempty"`
	Preseed         string `yaml:"preseed,omitempty"`
	Params          map[string]string

	StaleBuildThresholdSeconds int `yaml:"stale_build_threshold_secs,omitempty"`

	StaleBuildCommands  []BuildCommand `yaml:"stalebuild_commands,omitempty"`
	PreBuildCommands    []BuildCommand `yaml:"prebuild_commands,omitempty"`
	PostBuildCommands   []BuildCommand `yaml:"postbuild_commands,omitempty"`
	CancelBuildCommands []BuildCommand `yaml:"cancelbuild_commands,omitempty"`
}

type MachineInventoryPluginSettings struct {
	Name              string                 `yaml:"name"`
	Type              string                 `yaml:"type"`
	Source            string                 `yaml:"source"`
	AuthUser          string                 `yaml:"auth_user"`
	AuthPassword      string                 `yaml:"auth_password"`
	AuthToken         string                 `yaml:"auth_token"`
	AdditionalOptions map[string]interface{} `yaml:"additional_options"`
	WriteEnabled      bool                   `yaml:"writable"`
	Disabled          bool                   `yaml:"disabled"`
}

// Config is our global configuration file
type Config struct {
	TemplatePath    string `yaml:"templatepath,omitempty"`
	GroupPath       string `yaml:"grouppath,omitempty"`
	StaticFilesPath string `yaml:"staticspath,omitempty"`
	BaseURL         string `yaml:"baseurl,omitempty"`

	MachineInventoryPlugins  []MachineInventoryPluginSettings `yaml:"inventory_plugins,omitempty"`
	BuildTypes               map[string]BuildType             `yaml:"build_types,omitempty"`
	StaleBuildCheckFrequency int                              `yaml:"stale_build_check_frequency_secs,omitempty"`
	HistoryCacheSeconds      int                              `yaml:"history_cache_seconds"`
	LogLevel                 LogLevel                         `yaml:"log_level"`

	BuildType `yaml:",inline"`
}

// Loads config.yaml and returns a Config struct
func LoadConfig(configPath string) (*Config, error) {

	var c Config

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
