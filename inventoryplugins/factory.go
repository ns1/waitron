package inventoryplugins

import (
	"errors"

	"waitron/config"
	"waitron/machine"
)

var machineInventoryPlugins map[string]func(*config.MachineInventoryPluginSettings, *config.Config, func(string, int) bool) MachineInventoryPlugin = make(map[string]func(*config.MachineInventoryPluginSettings, *config.Config, func(string, int) bool) MachineInventoryPlugin)

type MachineInventoryPlugin interface {
	Init() error
	GetMachine(string, string) (*machine.Machine, error)
	PutMachine(*machine.Machine) error
	Deinit() error
}

func AddMachineInventoryPlugin(t string, f func(*config.MachineInventoryPluginSettings, *config.Config, func(string, int) bool) MachineInventoryPlugin) error {
	if _, found := machineInventoryPlugins[t]; found {
		return errors.New("plugin type already exists: " + t)
	}

	machineInventoryPlugins[t] = f

	return nil
}

func GetPlugin(t string, s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, int) bool) (MachineInventoryPlugin, error) {
	pNew, found := machineInventoryPlugins[t]

	if !found {
		return nil, errors.New("plugin type not found: " + t)
	}

	return pNew(s, c, lf), nil
}
