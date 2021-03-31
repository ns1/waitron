package inventoryplugins

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"waitron/config"
	"waitron/machine"

	"gopkg.in/yaml.v2"
)

func init() {
	if err := AddMachineInventoryPlugin("netbox", NewNetboxInventoryPlugin); err != nil {
		panic(err)
	}
}

type netboxInterfaceResults struct {
	Results []struct {
		ID           int    `yaml:"id"`
		Name         string `yaml:"name"`
		MacAddress   string `yaml:"mac_address"`
		Description  string `yaml:"description"`
		ParentDevice struct {
			Name string `yaml:"name"`
		} `yaml:"device"`
		ConnectedEndpoint struct {
			Name   string `yaml:"name"`
			Device struct {
				ID   int    `yaml:"id"`
				Name string `yaml:"name"`
			} `yaml:"device"`
		} `yaml:"connected_endpoint"`
		UntaggedVlan struct {
			Vid  int    `yaml:"vid"`
			Name string `yaml:"name"`
		} `yaml:"untagged_vlan"`
		Tags []struct {
			Name string `yaml:"name"`
		} `yaml:"tags"`
	} `yaml:"results"`
}

type netboxIpAddressResults struct {
	Results []struct {
		Family struct {
			Value int `yaml:"value"`
		} `yaml:"family"`
		AssignedObjectID int    `yaml:"assigned_object_id"`
		Address          string `yaml:"address"`
	} `yaml:"results"`
}

type netboxDeviceResults struct {
	Results []struct {
		ConfigContext map[string]interface{} `yaml:"config_context"`
	} `yaml:"results"`
}

type annotatedIface struct {
	iface  *machine.Interface
	isIpmi bool
}

type NetboxInventoryPlugin struct {
	settings      *config.MachineInventoryPluginSettings
	waitronConfig *config.Config
	Log           func(string, config.LogLevel) bool

	enabledAssetsFilter string
	machinePath         string
}

func NewNetboxInventoryPlugin(s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, config.LogLevel) bool) MachineInventoryPlugin {

	p := &NetboxInventoryPlugin{
		settings:      s, // Plugin settings
		waitronConfig: c, // Global waitron config
		Log:           lf,
	}

	return p

}

func (p *NetboxInventoryPlugin) Init() error {
	if p.settings.Source == "" {
		return fmt.Errorf("source for netbox plugin must not be empty")
	}

	if p.settings.AuthToken == "" {
		return fmt.Errorf("auth token for netbox plugin must not be empty")
	}

	return nil
}

func (p *NetboxInventoryPlugin) Deinit() error {
	return nil
}

func (p *NetboxInventoryPlugin) PutMachine(m *machine.Machine) error {
	return nil
}

func (p *NetboxInventoryPlugin) GetMachine(hostname string, macaddress string) (*machine.Machine, error) {
	hostname = strings.ToLower(hostname)

	m, err := machine.New(hostname)

	m.Params = make(map[string]string)

	if err != nil {
		return nil, err
	}

	if _, ok := p.settings.AdditionalOptions["enabled_assets_only"]; ok {
		if enabledAssetsOnly, ok := p.settings.AdditionalOptions["enabled_assets_only"].(bool); ok && enabledAssetsOnly {
			p.enabledAssetsFilter = "&enabled=true"
		}
	}

	// Let hostname win, but if it's not present, then we'll try to pull it from an interface that matchces the MAC passed. in.
	if hostname == "" && macaddress != "" {

		macResults := &netboxInterfaceResults{}

		response, err := p.queryNetbox(p.settings.Source + "/dcim/interfaces/?mac_address=" + macaddress)
		p.Log(fmt.Sprintf("retrieved interface data from netbox: %v", string(response)), config.LogLevelDebug)

		if err != nil {
			return nil, err
		}

		if err = yaml.Unmarshal(response, macResults); err != nil {
			return nil, err
		}

		// It wasn't an error, but it didn't result in finding a hostname.
		if macResults.Results[0].ParentDevice.Name == "" {
			p.Log(fmt.Sprintf("MAC '%s' used for netbox query, but no related hostname found", macaddress), config.LogLevelInfo)
			return nil, nil
		}
		hostname = macResults.Results[0].ParentDevice.Name
	}

	/*
		Try to grab the host and pull its config_context so that we can stuff it into a params value for people
		to use later with the from_yaml filter if they want.
		We need to do it this way because nested json data will blow up JSON marshalling in API responses.
	*/
	deviceResults := &netboxDeviceResults{}

	response, err := p.queryNetbox(p.settings.Source + "/dcim/devices/?device=" + hostname)

	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(response, &deviceResults); err != nil {
		return nil, err
		p.Log(fmt.Sprintf("ignoring config_context beacause unmarshal of content is somehow bad for '%s'", hostname), config.LogLevelError)
	} else {

		if len(deviceResults.Results) == 0 {
			p.Log(fmt.Sprintf("no matching device results for netbox query with '%s'", hostname), config.LogLevelInfo)
			return nil, nil
		}

		if len(deviceResults.Results) > 1 {
			p.Log(fmt.Sprintf("more than one device named '%s' found, so using the first one", hostname), config.LogLevelWarning)
		}

		// There should be only one.
		cc, err := yaml.Marshal(deviceResults.Results[0].ConfigContext)

		if err != nil {
			p.Log(fmt.Sprintf("ignoring config_context beacause re-marshal of content is somehow bad for '%s'", hostname), config.LogLevelError)
		} else {
			m.Params["config_context"] = string(cc)
		}
	}

	results := &netboxInterfaceResults{}

	response, err = p.queryNetbox(p.settings.Source + "/dcim/interfaces/?device=" + hostname)
	p.Log(fmt.Sprintf("retrieved interface data from netbox: %v", string(response)), config.LogLevelDebug)

	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(response, results); err != nil {
		return nil, err
	}

	/*
		We're implicitly saying that netbox as a datasource is only meant to provide a machine
		that has at least one interface.
		This doesn't have to be the case, but the only real job of this plugin is to provide interface/IP details.
		If they don't exist, then it shouldn't return a machine.
	*/

	if len(results.Results) == 0 {
		p.Log(fmt.Sprintf("no matching interface results for netbox query with '%v'", results), config.LogLevelInfo)
		return nil, nil
	}

	p.Log(fmt.Sprintf("have interface structure from netbox interface: %v", results), config.LogLevelDebug)

	annotatedInterfaces := make(map[int]*annotatedIface)

	netboxIfaces := results.Results

	// Making sure the array underneath doesn't change so that I can just grab references to the entries as needed.
	m.Network = make([]machine.Interface, len(netboxIfaces))

	// Grab all the interfaces for the device
	for idx, iface := range netboxIfaces {

		p.Log(fmt.Sprintf("found netbox interface: %v", iface), config.LogLevelDebug)

		m.Network[idx] = machine.Interface{Name: iface.Name, MacAddress: iface.MacAddress}
		newIface := &m.Network[idx]

		// We'll need to attach IP addresses to the interface in a little bit.
		annotatedInterfaces[iface.ID] = &annotatedIface{iface: newIface}

		newIface.ZSideDeviceInterface = iface.ConnectedEndpoint.Name
		newIface.ZSideDevice = iface.ConnectedEndpoint.Device.Name

		newIface.VlanName = iface.UntaggedVlan.Name
		newIface.VlanID = iface.UntaggedVlan.Vid
		newIface.Description = iface.Description

		for _, tag := range iface.Tags {

			newIface.Tags = append(newIface.Tags, tag.Name)

			if tag.Name == "waitron_ipmi" {
				annotatedInterfaces[iface.ID].isIpmi = true
				p.Log(fmt.Sprintf("found ipmi interface: %v", m.Network[idx]), config.LogLevelDebug)
			}
		}

	}

	ipResults := &netboxIpAddressResults{}

	response, err = p.queryNetbox(p.settings.Source + "/ipam/ip-addresses/?device=" + hostname)

	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(response, ipResults); err != nil {
		return nil, err
	}

	p.Log(fmt.Sprintf("retrieved ip address data from netbox: %v", ipResults), config.LogLevelDebug)

	addrs := ipResults.Results

	// Grab all the ip addresses for the device
	for _, addr := range addrs {

		annotatedIface := annotatedInterfaces[addr.AssignedObjectID]
		iface := annotatedIface.iface

		_, ipNet, err := net.ParseCIDR(addr.Address)

		if err != nil {
			p.Log(fmt.Sprintf("skipping unparseable address '%s' for interface %s", addr.Address, iface.Name), config.LogLevelWarning)
			continue
		}

		addressParts := strings.Split(addr.Address, "/")

		// Watch out!  We're assuming there's only a single IPMI address of either v4 or v6.
		// Operators can always get around this by passing in IPMI details other ways.
		if annotatedIface.isIpmi {
			m.IpmiAddressRaw = addressParts[0]
			p.Log(fmt.Sprintf("added ipmi address to interface %s for %s: %s", iface.Name, hostname, addressParts[0]), config.LogLevelDebug)
			p.Log(fmt.Sprintf("interface %v", *iface), config.LogLevelDebug)
		}

		if addr.Family.Value == 4 {
			netmask := fmt.Sprintf("%d.%d.%d.%d", ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3])

			// Update the list of addresses in the related interface
			iface.Addresses4 = append(iface.Addresses4, machine.IPConfig{IPAddress: addressParts[0], Cidr: addressParts[1], Netmask: netmask})
			p.Log(fmt.Sprintf("added ipv4 address to interface %s for %s: %s", iface.Name, hostname, addressParts[0]), config.LogLevelDebug)

			if iface.Gateway4 == "" {

				gw, err := p.getGateway(iface, addr.Address)

				if gw == "" || err != nil {
					p.Log(fmt.Sprintf("no gateway address found for '%s' for interface %s: %v", addr.Address, iface.Name, err), config.LogLevelWarning)
					if err != nil {
						return nil, err
					}
				}
				iface.Gateway4 = gw
			}

		} else if addr.Family.Value == 6 {
			netmask := fmt.Sprintf("%x%x:%x%x:%x%x:%x%x:%x%x:%x%x:%x%x:%x%x",
				ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3],
				ipNet.Mask[4], ipNet.Mask[5], ipNet.Mask[6], ipNet.Mask[7],
				ipNet.Mask[8], ipNet.Mask[9], ipNet.Mask[10], ipNet.Mask[11],
				ipNet.Mask[12], ipNet.Mask[13], ipNet.Mask[14], ipNet.Mask[15])

			// Update the list of addresses in the related interface
			iface.Addresses6 = append(iface.Addresses6, machine.IPConfig{IPAddress: addressParts[0], Cidr: addressParts[1], Netmask: netmask})
			p.Log(fmt.Sprintf("added ipv6 address to interface %s for %s: %s", iface.Name, hostname, addressParts[0]), config.LogLevelDebug)

			if iface.Gateway6 == "" {

				gw, err := p.getGateway(iface, addr.Address)

				if gw == "" || err != nil {
					p.Log(fmt.Sprintf("no gateway address found for '%s' for interface %s: %v", addr.Address, iface.Name, err), config.LogLevelWarning)
					if err != nil {
						return nil, err
					}
				}
				iface.Gateway6 = gw
			}
		}

	}

	return m, nil

}

func (p *NetboxInventoryPlugin) getGateway(iface *machine.Interface, addr string) (string, error) {

	gwResponse, err := p.queryNetbox(p.settings.Source + "/ipam/ip-addresses/?tag=waitron_gateway&parent=" + addr)

	if err != nil {
		return "", err
	}

	gwResults := &netboxIpAddressResults{}

	if err := yaml.Unmarshal(gwResponse, gwResults); err != nil {
		return "", err
	}

	gateways := gwResults.Results
	if len(gateways) > 1 {
		p.Log(fmt.Sprintf("multiple gateways found for '%s' for interface %s", addr, iface.Name), config.LogLevelWarning)
	} else if len(gateways) == 0 {
		p.Log(fmt.Sprintf("no gateways found for '%s' for interface %s", addr, iface.Name), config.LogLevelWarning)
		return "", nil
	}

	for _, gateway := range gateways {
		if gateway.Address != "" {
			return strings.Split(gateway.Address, "/")[0], nil
		}
	}
	return "", nil
}

func (p *NetboxInventoryPlugin) queryNetbox(q string) ([]byte, error) {

	q = q + p.enabledAssetsFilter

	p.Log(fmt.Sprintf("going to query %s", q), config.LogLevelDebug)

	tr := &http.Transport{
		ResponseHeaderTimeout: 10 * time.Second,
	}

	client := &http.Client{Transport: tr, Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", q, nil)

	if err != nil {
		p.Log(fmt.Sprintf("unable to create request for querying %s: %v", q, err), config.LogLevelDebug)
		return nil, err
	}

	req.Header.Add("Authorization", "Token "+string(p.settings.AuthToken))

	resp, err := client.Do(req)

	if err != nil {
		p.Log(fmt.Sprintf("error while querying %s: %v", q, err), config.LogLevelDebug)
		return nil, err
	}

	if resp.StatusCode >= 400 {
		p.Log(fmt.Sprintf("error while querying %s: %v", q, err), config.LogLevelDebug)
		return nil, err
	}

	response, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		p.Log(fmt.Sprintf("error while reading body of query to %s: %v", q, err), config.LogLevelDebug)
		return nil, err
	}

	return response, nil
}
