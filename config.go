package main

import (
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "path"
    "sync"
)

// Config is our global configuration file
type State struct {
    Mux                 sync.Mutex
    Tokens              map[string]string
    MachineByUUID       map[string]*Machine
    MachineByMAC        map[string]*Machine
    MachineByHostname   map[string]*Machine
}

type BuildCommand struct {
    Command           string
    TimeoutSeconds    int `yaml:"timeout_seconds"`
    ErrorsFatal       bool `yaml:"errors_fatal"`
    ShouldLog         bool `yaml:"should_log"`
}

type Config struct {
    TemplatePath        string
    GroupPath           string
    MachinePath         string
    StaticFilesPath	    string `yaml:"staticspath"`
    BaseURL             string
    ForemanProxyAddress string `yaml:"foreman_proxy_address"`

    Cmdline             string `yaml:"cmdline"`
    Kernel              string `yaml:"kernel"`
    Initrd              string `yaml:"initrd"`
    ImageURL            string `yaml:"image_url"`

    RescueCmdline       string `yaml:"rescue_cmdline"`
    RescueKernel        string `yaml:"rescue_kernel"`
    RescueInitrd        string `yaml:"rescue_initrd"`
    RescueImageURL      string `yaml:"rescue_image_url"`

    OperatingSystem     string
    Finish              string
    Preseed             string
    Params              map[string]string

    StaleBuildThresholdSeconds  int `yaml:"stale_build_threshold_secs"`
    StaleBuildCheckFrequency    int `yaml:"stale_build_check_frequency_secs"`
    StaleBuildCommands          []BuildCommand `yaml:"stalebuild_commands"`
    PreBuildCommands            []BuildCommand `yaml:"prebuild_commands"`
    PostBuildCommands           []BuildCommand `yaml:"postbuild_commands"`
    CancelBuildCommands         []BuildCommand `yaml:"cancelbuild_commands"`
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
    // Initialize maps
    s.Tokens = make(map[string]string)
    s.MachineByUUID = make(map[string]*Machine)
    s.MachineByMAC = make(map[string]*Machine)
    s.MachineByHostname = make(map[string]*Machine)
    return s
}

func (c Config) listMachines() ([]string, error) {
    var machines []string
    files, err := ioutil.ReadDir(c.MachinePath)
    for _, file := range files {
        name := file.Name()
        if path.Ext(name) == ".yaml" || path.Ext(name) == ".yml" {
            machines = append(machines, name)
        }
    }
    if err != nil {
        return machines, err
    }
    return machines, nil
}
