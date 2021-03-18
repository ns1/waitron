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
}

func NewFileInventoryPlugin(s *config.MachineInventoryPluginSettings, c *config.Config) MachineInventoryPlugin {

	p := &FileInventoryPlugin{
		settings:      s, // Plugin settings
		waitronConfig: c, // Global waitron config
	}

	return p

}

func (p *FileInventoryPlugin) Init() error {
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

	// Move the path settings and checks to Init so we can blow up early.
	if groupPath, ok := p.settings.AdditionalOptions["grouppath"].(string); ok {
		// Then, load the domain definition.
		data, err := ioutil.ReadFile(path.Join(groupPath, m.Domain+".yaml")) // apc03.prod.yaml

		if os.IsNotExist(err) {
			data, err = ioutil.ReadFile(path.Join(groupPath, m.Domain+".yml")) // Try .yml
			if err != nil && !os.IsNotExist(err) {                             // We should expect the file to not exist, but if it did exist, err happened for a different reason, then it should be reported.
				return m, err
			}
		} else {
			return m, err
		}

		if err = yaml.Unmarshal(data, m); err != nil {
			return m, err
		}

	}
	machinePath := ""

	if machinePath, _ := p.settings.AdditionalOptions["machinepath"].(string); machinePath == "" {
		return m, fmt.Errorf("machine path not found in config of file plugin")
	}

	// Then load the machine definition.
	data, err := ioutil.ReadFile(path.Join(machinePath, hostname+".yaml")) // compute01.apc03.prod.yaml

	if err != nil {
		if os.IsNotExist(err) {
			data, err = ioutil.ReadFile(path.Join(machinePath, hostname+".yml")) // One more try but look for .yml
			if err != nil {                                                      // Whether the error was due to non-existence or something else, report it.  Machine definitions are must.
				return m, err
			}
		} else {
			return m, err
		}
	}

	err = yaml.Unmarshal(data, m)
	if err != nil {
		return m, err
	}

	return m, nil
}
