package inventory_plugins

import (
	"errors"

	"waitron/config"
	"waitron/machine"
)

var machineInventoryPlugins map[string]func(*config.MachineInventoryPluginSettings, *config.Config) MachineInventoryPlugin = make(map[string]func(*config.MachineInventoryPluginSettings, *config.Config) MachineInventoryPlugin)

type MachineInventoryPlugin interface {
	Init() error
	GetMachine(string, string) (*machine.Machine, error)
	PutMachine(*machine.Machine) error
	Deinit() error
}

func AddMachineInventoryPlugin(t string, f func(*config.MachineInventoryPluginSettings, *config.Config) MachineInventoryPlugin) error {
	if _, found := machineInventoryPlugins[t]; found {
		return errors.New("plugin type already exists: " + t)
	}

	machineInventoryPlugins[t] = f

	return nil
}

func GetPlugin(t string, s *config.MachineInventoryPluginSettings, c *config.Config) (MachineInventoryPlugin, error) {
	pNew, found := machineInventoryPlugins[t]

	if !found {
		return nil, errors.New("plugin type not found: " + t)
	}

	return pNew(s, c), nil
}
