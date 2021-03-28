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

type annotatedIface struct {
	iface          *machine.Interface
	isIpmi         bool
	isProvisioning bool
}

type NetboxInventoryPlugin struct {
	settings      *config.MachineInventoryPluginSettings
	waitronConfig *config.Config
	Log           func(string, config.LogLevel) bool

	machinePath string
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

	if err != nil {
		return nil, err
	}

	m.Params = make(map[string]string)

	var results map[string]interface{}

	if hostname != "" {
		results, err = p.queryNetbox(p.settings.Source + "/dcim/interfaces/?device=" + hostname)

		if err != nil {
			return nil, err
		}
	} else if macaddress != "" {
		// Fill in!
	}

	p.Log(fmt.Sprintf("retrieved interface data from netbox: %v", results), config.LogLevelDebug)

	annotatedInterfaces := make(map[int]*annotatedIface)

	ifaces, ok := results["results"].([]interface{})

	if !ok {
		return nil, fmt.Errorf("failed to assert interfaces list")
	}

	// Making sure the array underneath doesn't change so that I can just grab references to the entries as needed.
	m.Network = make([]machine.Interface, len(ifaces))

	// Grab all the interfaces for the device
	for idx, i := range ifaces {

		iface := i.(map[interface{}]interface{})

		id := iface["id"].(int)
		name := iface["name"].(string)
		mac, ok := iface["mac_address"].(string)

		isProvisioningIface := false

		// MAC is not required on all interfaces in netbox.
		if !ok {
			mac = ""
		}

		m.Network[idx] = machine.Interface{Name: name, MacAddress: mac}
		newIface := &m.Network[idx]

		// We'll need to attach IP addresses to the interface in a little bit.
		annotatedInterfaces[id] = &annotatedIface{iface: newIface}

		if zSide, ok := iface["connected_endpoint"].(map[interface{}]interface{}); ok {
			newIface.ZSideDeviceInterface, _ = zSide["name"].(string)
			newIface.ZSideDevice, _ = zSide["device"].(map[interface{}]interface{})["name"].(string)
		}

		if primaryVlan, ok := iface["untagged_vlan"].(map[interface{}]interface{}); ok {
			newIface.VlanName, _ = primaryVlan["name"].(string)
			newIface.VlanID, _ = primaryVlan["vid"].(int)
		}

		if tags, ok := iface["tags"].([]interface{}); ok {
			for _, sTag := range tags {
				if tag, ok := sTag.(map[interface{}]interface{}); ok {
					if tagName, ok := tag["name"].(string); ok {
						newIface.Tags = append(newIface.Tags, tagName)

						if tagName == "ipmi" {
							annotatedInterfaces[id].isIpmi = true
						} else if tagName == "provisioning" {
							annotatedInterfaces[id].isProvisioning = true
						}
					}
				}
			}
		}

		if isProvisioningIface {
			// Entirely possible this isn't set.
			if desc, ok := iface["description"].(string); ok {
				m.Params["provisioning_zside_interface"] = strings.Split(desc, ";")[0]
			}

			m.Params["provisioning_vlan"] = newIface.VlanName
			m.Params["provisioning_vlan_id"] = fmt.Sprintf("%d", newIface.VlanID)
			m.Params["provisioning_interface"] = name
		}

	}

	if hostname != "" {
		results, err = p.queryNetbox(p.settings.Source + "/ipam/ip-addresses/?device=" + hostname)

		if err != nil {
			return nil, err
		}
	} else if macaddress != "" {
		// Fill in! OR with our first hostname vs mac if-block, use mac to trace back to a hostname and then set it at that point.
	}

	p.Log(fmt.Sprintf("retrieved ip address data from netbox: %v", results), config.LogLevelDebug)

	addrs, ok := results["results"].([]interface{})

	if !ok {
		return nil, fmt.Errorf("failed to assert ip addresses list")
	}

	// Grab all the ip addresses for the device
	for _, a := range addrs {

		addr, ok := a.(map[interface{}]interface{})

		if !ok {
			return nil, fmt.Errorf("failed to assert ip address")
		}

		iface_id := addr["assigned_object_id"].(int)
		address := addr["address"].(string)
		family := addr["family"].(map[interface{}]interface{})["value"].(int)

		iface := annotatedInterfaces[iface_id].iface

		_, ipNet, err := net.ParseCIDR(address)

		if err != nil {
			p.Log(fmt.Sprintf("skipping unparseable address '%s' for interface %s", address, iface.Name), config.LogLevelWarning)
			continue
		}

		// Watch out!  We're assuming there's only a single IPMI address of either v4 or v6.
		// Operators can always get around this by passing in IPMI details other ways.
		if family == 4 {
			addressParts := strings.Split(address, "/")
			netmask := fmt.Sprintf("%d.%d.%d.%d", ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3])

			// Update the list of addresses in the related interface
			iface.Addresses4 = append(iface.Addresses4, machine.IPConfig{IPAddress: addressParts[0], Cidr: addressParts[1], Netmask: netmask})
			p.Log(fmt.Sprintf("added ipv4 address to interface %s for %s: %s", iface.Name, hostname, addressParts[0]), config.LogLevelDebug)

			if annotatedInterfaces[iface_id].isIpmi {
				m.IpmiAddressRaw = addressParts[0]
				p.Log(fmt.Sprintf("added ipv4 ipmi address to interface %s for %s: %s", iface.Name, hostname, addressParts[0]), config.LogLevelDebug)
				p.Log(fmt.Sprintf("interface %v", *iface), config.LogLevelDebug)
			}

			if annotatedInterfaces[iface_id].isProvisioning {
				p.Log(fmt.Sprintf("added ipv4 provisioning details to params %s", hostname), config.LogLevelDebug)
				m.Params["provisioning_address"] = addressParts[0]
				m.Params["provisioning_netmask"] = netmask
				m.Params["provisioning_cidr"] = addressParts[1]
			}

			if iface.Gateway4 == "" {
				gwResults, err := p.queryNetbox(p.settings.Source + "/ipam/ip-addresses/?tag=gateway&parent=" + address)

				if err != nil {
					p.Log(fmt.Sprintf("no gateway details found for '%s' for interface %s", address, iface.Name), config.LogLevelWarning)
				}

				if gateways, ok := gwResults["results"].([]interface{}); ok {
					if len(gateways) > 1 {
						p.Log(fmt.Sprintf("multiple gateways found for '%s' for interface %s", address, iface.Name), config.LogLevelWarning)
					}

					for idx := 0; idx < len(gateways) && iface.Gateway4 == ""; idx++ {
						gateway, ok := gateways[idx].(map[interface{}]interface{})
						if !ok {
							p.Log(fmt.Sprintf("gateway object #%d found for '%s' for interface %s is the wrong structure", idx, address, iface.Name), config.LogLevelWarning)
							continue
						}

						iface.Gateway4, _ = gateway["address"].(string)

						if iface.Gateway4 == "" {
							p.Log(fmt.Sprintf("no good gateway address in gateway #%d found for '%s' for interface %s", idx, address, iface.Name), config.LogLevelWarning)
							continue
						}

						iface.Gateway4 = strings.Split(iface.Gateway4, "/")[0]
					}
				}
			}

		} else if family == 6 {
			addressParts := strings.Split(address, "/")
			netmask := fmt.Sprintf("%x%x:%x%x:%x%x:%x%x:%x%x:%x%x:%x%x:%x%x", ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3], ipNet.Mask[4], ipNet.Mask[5], ipNet.Mask[6], ipNet.Mask[7], ipNet.Mask[8], ipNet.Mask[9], ipNet.Mask[10], ipNet.Mask[11], ipNet.Mask[12], ipNet.Mask[13], ipNet.Mask[14], ipNet.Mask[15])

			// Update the list of addresses in the related interface
			iface.Addresses6 = append(iface.Addresses6, machine.IPConfig{IPAddress: addressParts[0], Cidr: addressParts[1], Netmask: netmask})
			p.Log(fmt.Sprintf("added ipv6 address to interface %s for %s: %s", iface.Name, hostname, addressParts[0]), config.LogLevelDebug)

			if annotatedInterfaces[iface_id].isIpmi {
				p.Log(fmt.Sprintf("added ipv6 ipmi address to interface %s for %s: %s", iface.Name, hostname, addressParts[0]), config.LogLevelDebug)
				m.IpmiAddressRaw = addressParts[0]
			}

			if annotatedInterfaces[iface_id].isProvisioning {
				p.Log(fmt.Sprintf("added ipv4 provisioning details to params %s", hostname), config.LogLevelDebug)
				m.Params["provisioning_address"] = addressParts[0]
				m.Params["provisioning_netmask"] = netmask
				m.Params["provisioning_cidr"] = addressParts[1]
			}

			if iface.Gateway6 == "" {
				gwResults, err := p.queryNetbox(p.settings.Source + "/ipam/ip-addresses/?tag=gateway&parent=" + address)

				if err != nil {
					p.Log(fmt.Sprintf("no gateway details found for '%s' for interface %s", address, iface.Name), config.LogLevelWarning)
				}

				if gateways, ok := gwResults["results"].([]interface{}); ok {
					if len(gateways) > 1 {
						p.Log(fmt.Sprintf("multiple gateways found for '%s' for interface %s", address, iface.Name), config.LogLevelWarning)
					}

					for idx := 0; idx < len(gateways) && iface.Gateway6 == ""; idx++ {
						gateway, ok := gateways[idx].(map[interface{}]interface{})
						if !ok {
							p.Log(fmt.Sprintf("gateway object #%d found for '%s' for interface %s is the wrong structure", idx, address, iface.Name), config.LogLevelWarning)
							continue
						}

						iface.Gateway6, _ = gateway["address"].(string)

						if iface.Gateway6 == "" {
							p.Log(fmt.Sprintf("no good gateway address in gateway #%d found for '%s' for interface %s", idx, address, iface.Name), config.LogLevelWarning)
							continue
						}
						iface.Gateway6 = strings.Split(iface.Gateway6, "/")[0]
					}
				}
			}

		}

	}

	return m, nil

}

func (p *NetboxInventoryPlugin) queryNetbox(q string) (map[string]interface{}, error) {

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

	req.Header.Add("Authorization", "Token "+p.settings.AuthToken)

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

	i := make(map[string]interface{})

	if err = yaml.Unmarshal(response, i); err != nil {
		return nil, err
	}

	return i, nil
}
