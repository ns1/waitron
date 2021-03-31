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
	if err := AddMachineInventoryPlugin("groups", NewGroupsInventoryPlugin); err != nil {
		panic(err)
	}
}

type GroupsInventoryPlugin struct {
	settings      *config.MachineInventoryPluginSettings
	waitronConfig *config.Config
	Log           func(string, config.LogLevel) bool

	groupPath string
}

func NewGroupsInventoryPlugin(s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, config.LogLevel) bool) MachineInventoryPlugin {

	p := &GroupsInventoryPlugin{
		settings:      s, // Plugin settings
		waitronConfig: c, // Global waitron config
		Log:           lf,
	}

	return p

}

func (p *GroupsInventoryPlugin) Init() error {
	if p.groupPath, _ = p.settings.AdditionalOptions["grouppath"].(string); p.groupPath == "" {
		return fmt.Errorf("group path not found in config of file plugin")
	}

	p.groupPath = strings.TrimRight(p.groupPath, "/") + "/"

	return nil
}

func (p *GroupsInventoryPlugin) Deinit() error {
	return nil
}

func (p *GroupsInventoryPlugin) PutMachine(m *machine.Machine) error {
	return nil
}

func (p *GroupsInventoryPlugin) GetMachine(hostname string, macaddress string) (*machine.Machine, error) {
	hostname = strings.ToLower(hostname)

	m, err := machine.New(hostname)

	if err != nil {
		return nil, err
	}

	p.Log(fmt.Sprintf("first attempt at slurping %s.[yml|yaml] in %s", m.Domain, p.groupPath), config.LogLevelDebug)

	data, err := ioutil.ReadFile(path.Join(p.groupPath, m.Domain+".yaml")) // apc03.prod.yaml

	if os.IsNotExist(err) {
		p.Log(fmt.Sprintf("second attempt at slurping %s.[yml|yaml] in %s", m.Domain, p.groupPath), config.LogLevelDebug)
		data, err = ioutil.ReadFile(path.Join(p.groupPath, m.Domain+".yml")) // Try .yml
		if err != nil && !os.IsNotExist(err) {                               // We should expect the file to not exist, but if it did exist, err happened for a different reason, then it should be reported.
			return nil, err // Some error beyond just "not found"
		}
	} else {
		return nil, err // Some error beyond just "not found"
	}

	p.Log(fmt.Sprintf("%s.[yml|yaml] slurped in from %s", m.Domain, p.groupPath), config.LogLevelDebug)

	if len(data) > 0 { // Group Files are optional.  So we shouldn't be failing unless they were requested and found.
		if err = yaml.Unmarshal(data, m); err != nil {
			return nil, err
		}
	}

	p.Log(fmt.Sprintf("got machine from plugin in: %v", m), config.LogLevelDebug)

	return m, nil
}
