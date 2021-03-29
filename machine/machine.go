package machine

import (
	"strings"
	"waitron/config"
)

// Machine configuration
type Machine struct {
	config.Config `yaml:",inline"`

	Hostname  string      `yaml:"hostname,omitempty"`
	ShortName string      `yaml:"shortname,omitempty"`
	Domain    string      `yaml:"domain,omitempty"`
	Network   []Interface `yaml:"network,omitempty"`

	IpmiAddressRaw string `yaml:"ipmi_address"`
	IpmiUser       string `yaml:"ipmi_user"`
	IpmiPassword   string `yaml:"ipmi_password"`

	BuildTypeName string `yaml:"build_type,omitempty"`
}

type IPConfig struct {
	IPAddress   string   `yaml:"ipaddress"`
	Netmask     string   `yaml:"netmask"`
	Cidr        string   `yaml:"cidr"`
	Tags        []string `yaml:"tags`
	Description string   `yaml:"description"`
}

// Interface Configuration
type Interface struct {
	Name                 string     `yaml:"name"`
	Addresses4           []IPConfig `yaml:"addresses4"`
	Addresses6           []IPConfig `yaml:"addresses6"`
	MacAddress           string     `yaml:"macaddress"`
	VlanID               int        `yaml"vlan_id"`
	VlanName             string     `yaml"vlan_name"`
	Gateway4             string     `yaml:"gateway4"`
	Gateway6             string     `yaml:"gateway6"`
	ZSideDevice          string     `yaml:"zside_device"`
	ZSideDeviceInterface string     `yaml:"zside_device_port"`
	Tags                 []string   `yaml:"tags`
	Description          string     `yaml:"description"`
}

func New(hostname string) (*Machine, error) {
	hostname = strings.ToLower(hostname)
	hostSlice := strings.Split(hostname, ".")

	m := &Machine{
		Hostname:  hostname,
		ShortName: hostSlice[0],
		Domain:    strings.Join(hostSlice[1:], "."),
	}

	return m, nil
}
