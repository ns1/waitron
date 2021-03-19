package inventoryplugins_test

import (
	"testing"

	"waitron/config"
	"waitron/inventoryplugins"
	"waitron/machine"
)

type TestPlugin struct {
}

func (t *TestPlugin) Init() error {
	return nil
}

func (t *TestPlugin) GetMachine(s string, m string) (*machine.Machine, error) {

	return &machine.Machine{}, nil
}

func (t *TestPlugin) PutMachine(m *machine.Machine) error {
	return nil
}

func (t *TestPlugin) Deinit() error {
	return nil
}

func TestNew(t *testing.T) {

	if _, err := inventoryplugins.GetPlugin("test", &config.MachineInventoryPluginSettings{}, &config.Config{}, func(s string, i int) bool { return true }); err == nil {
		t.Errorf("Plugin factory did not return error for unknown type.")
	}

	if err := inventoryplugins.AddMachineInventoryPlugin("test", func(s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, int) bool) inventoryplugins.MachineInventoryPlugin {
		return &TestPlugin{}
	}); err != nil {
		t.Errorf("Plugin factory failed to add new type.")
	}

	if err := inventoryplugins.AddMachineInventoryPlugin("test", func(s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, int) bool) inventoryplugins.MachineInventoryPlugin {
		return &TestPlugin{}
	}); err == nil {
		t.Errorf("Plugin factory allowed duplicate type.")
	}

	if _, err := inventoryplugins.GetPlugin("test", &config.MachineInventoryPluginSettings{}, &config.Config{}, func(s string, i int) bool { return true }); err != nil {
		t.Errorf("Plugin factory failed to return known type.")
	}

}
