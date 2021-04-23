package config

import (
	"io/ioutil"
	"strings"

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

var ll = map[string]LogLevel{
	"ERROR": LogLevelError,
	"WARN":  LogLevelWarning,
	"INFO":  LogLevelInfo,
	"DEBUG": LogLevelDebug,
}

type BuildCommand struct {
	Command        string
	TimeoutSeconds int  `yaml:"timeout_seconds"`
	ErrorsFatal    bool `yaml:"errors_fatal"`
	ShouldLog      bool `yaml:"should_log"`
}

type BuildType struct {
	Cmdline  string   `yaml:"cmdline,omitempty"`
	Kernel   string   `yaml:"kernel,omitempty"`
	Initrd   []string `yaml:"initrd,omitempty"`
	ImageURL string   `yaml:"image_url,omitempty"`

	OperatingSystem string            `yaml:"operatingsystem,omitempty"`
	Finish          string            `yaml:"finish,omitempty"`
	Preseed         string            `yaml:"preseed,omitempty"`
	Params          map[string]string `yaml:"params,omitempty"`

	StaleBuildThresholdSeconds int `yaml:"stale_build_threshold_secs,omitempty"`

	StaleBuildCommands   []BuildCommand `yaml:"stalebuild_commands,omitempty"`
	PreBuildCommands     []BuildCommand `yaml:"prebuild_commands,omitempty"`
	PostBuildCommands    []BuildCommand `yaml:"postbuild_commands,omitempty"`
	CancelBuildCommands  []BuildCommand `yaml:"cancelbuild_commands,omitempty"`
	UnknownBuildCommands []BuildCommand `yaml:"unknownbuild_commands,omitempty"`
	PxeEventCommands     []BuildCommand `yaml:"pxeevent_commands,omitempty"`

	Tags        []string `yaml:"tags`
	Description string   `yaml:"description`
}

/*
	All the wacky marshal/unmarshal stuff being done internall uses the yaml lib,
	and we only start doing JSON when we want to respond to API calls.
	That means, for now, we can easily hide password values with a custom MarshalJSON.
*/
type Password string

func (pw *Password) MarshalJSON() ([]byte, error) {
	return []byte{'"', '*', '*', '*', '"'}, nil
}

type MachineInventoryPluginSettings struct {
	Name              string                 `yaml:"name"`
	Type              string                 `yaml:"type"`
	Source            string                 `yaml:"source"`
	AuthUser          string                 `yaml:"auth_user"`
	AuthPassword      Password               `yaml:"auth_password"`
	AuthToken         Password               `yaml:"auth_token"`
	AdditionalOptions map[string]interface{} `yaml:"additional_options"`
	Weight            int                    `yaml:"weight"`
	WriteEnabled      bool                   `yaml:"writable"`
	Disabled          bool                   `yaml:"disabled"`
	SupplementalOnly  bool                   `yaml:"supplemental_only"`
}

// Config is our global configuration file
/*
	The omitempty's need to be cleaned up.  They're mostly there to let someone see the state of things when they requested a build.
	If they try to override some of the values in a machine definition from an inventory plugin, it'll show in the JSON
	response that the API endpoints provide, but it'll be a lie because they won't have actually changed the config.
*/
type Config struct {
	TempPath        string `yaml:"temp_path,omitempty"`
	TemplatePath    string `yaml:"templatepath,omitempty"`
	StaticFilesPath string `yaml:"staticspath,omitempty"`
	BaseURL         string `yaml:"baseurl,omitempty"`

	MachineInventoryPlugins  []MachineInventoryPluginSettings `yaml:"inventory_plugins,omitempty"`
	BuildTypes               map[string]BuildType             `yaml:"build_types,omitempty"`
	StaleBuildCheckFrequency int                              `yaml:"stale_build_check_frequency_secs,omitempty"`
	HistoryCacheSeconds      int                              `yaml:"history_cache_seconds,omitempty"`
	LogLevelName             string                           `yaml:"log_level,omitempty"`
	LogLevel                 LogLevel                         `yaml:"-,omitempty"`

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

	c.LogLevel = ll[strings.ToUpper(c.LogLevelName)]

	return &c, nil
}
