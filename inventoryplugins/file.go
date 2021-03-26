package inventoryplugins

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"waitron/config"
	"waitron/machine"

	"gopkg.in/yaml.v2"
)

func init() {
	if err := AddMachineInventoryPlugin("file", NewFileInventoryPlugin); err != nil {
		panic(err)
	}
}

type FileInventoryPlugin struct {
	settings      *config.MachineInventoryPluginSettings
	waitronConfig *config.Config
	Log           func(string, config.LogLevel) bool

	machinePath string
}

func NewFileInventoryPlugin(s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, config.LogLevel) bool) MachineInventoryPlugin {

	p := &FileInventoryPlugin{
		settings:      s, // Plugin settings
		waitronConfig: c, // Global waitron config
		Log:           lf,
	}

	return p

}

func (p *FileInventoryPlugin) Init() error {
	if p.machinePath, _ = p.settings.AdditionalOptions["machinepath"].(string); p.machinePath == "" {
		return fmt.Errorf("machine path not found in config of file plugin")
	}

	p.machinePath = strings.TrimRight(p.machinePath, "/") + "/"

	return nil
}

func (p *FileInventoryPlugin) Deinit() error {
	return nil
}

func (p *FileInventoryPlugin) PutMachine(m *machine.Machine) error {
	return nil
}

func (p *FileInventoryPlugin) GetMachine(hostname string, macaddress string) (*machine.Machine, error) {
	hostname = strings.ToLower(hostname)
	hostSlice := strings.Split(hostname, ".")

	m := &machine.Machine{
		Hostname:  hostname,
		ShortName: hostSlice[0],
		Domain:    strings.Join(hostSlice[1:], "."),
	}

	p.Log(fmt.Sprintf("looking for %s.[yml|yaml] in %s", hostname, p.machinePath), config.LogLevelDebug)

	// Then load the machine definition.
	data, err := ioutil.ReadFile(path.Join(p.machinePath, hostname+".yaml")) // compute01.apc03.prod.yaml

	p.Log(fmt.Sprintf("first attempt at slurping %s.[yml|yaml] in %s", hostname, p.machinePath), config.LogLevelDebug)

	if err != nil {
		if os.IsNotExist(err) {

			data, err = ioutil.ReadFile(path.Join(p.machinePath, hostname+".yml")) // One more try but look for .yml
			p.Log(fmt.Sprintf("second attempt at slurping %s.[yml|yaml] in %s", hostname, p.machinePath), config.LogLevelDebug)

			if err != nil {
				if os.IsNotExist(err) { // Whether the error was due to non-existence or something else, report it.  Machine definitions are must.
					p.Log(fmt.Sprintf("%s.[yml|yaml] not found in %s", hostname, p.machinePath), config.LogLevelDebug)
					return nil, nil
				} else {
					p.Log(fmt.Sprintf("%v", err), config.LogLevelDebug)
					return nil, err // Some error beyond just "not found"
				}
			}
		} else {
			p.Log(fmt.Sprintf("%v", err), config.LogLevelDebug)
			return nil, err // Some error beyond just "not found"
		}
	}

	p.Log(fmt.Sprintf("%s.[yml|yaml] slurped in from %s", hostname, p.machinePath), config.LogLevelDebug)

	err = yaml.Unmarshal(data, m)
	if err != nil {
		// Don't blow everything up on bad data.  Only truly critical errors should be passed back.
		// Log and return "nothing" so that the next inventory plugin can do stuff.
		// If this was the only inventory plugin used, then the build request will fail, anyway.
		p.Log(fmt.Sprintf("unable to unmarshal %s.[yml|yaml]: %v", hostname, err), config.LogLevelError)
		return nil, nil
	}

	p.Log(fmt.Sprintf("got machine from plugin in: %v", m), config.LogLevelDebug)

	return m, nil
}
