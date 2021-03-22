package machine

import (
	"waitron/config"
)

// Machine configuration
type Machine struct {
	config.Config `yaml:",inline"`
	Hostname      string      `yaml:"hostname,omitempty"`
	ShortName     string      `yaml:"shortname,omitempty"`
	Domain        string      `yaml:"domain,omitempty"`
	Network       []Interface `yaml:"network,omitempty"`
	BuildTypeName string      `yaml:"build_type,omitempty"`
}

type IPConfig struct {
	IPAddress string `yaml:"ipaddress"`
	Netmask   string `yaml:"netmask"`
	Cidr      string `yaml:"cidr"`
}

// Interface Configuration
type Interface struct {
	Name       string     `yaml:"name"`
	Addresses4 []IPConfig `yaml:"addresses4"`
	Addresses6 []IPConfig `yaml:"addresses6"`
	MacAddress string     `yaml:"macaddress"`
	Gateway4   string     `yaml:"gateway4"`
	Gateway6   string     `yaml:"gateway6"`
}
