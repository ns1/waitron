package main

import (
	"fmt"
	"github.com/flosch/pongo2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Machine configuration
type Machine struct {
	Hostname        string
	OperatingSystem string
	Finish          string
	Preseed         string
	ShortName       string
	Domain          string
	Token           string // This is set by the service
	Network         []Interface
	Params          map[string]string
}

// Interface Configuration
type Interface struct {
	Name       string
	IPAddress  string
	MacAddress string
	Gateway    string
	Netmask    string
}

func machineDefinition(hostname string, machinePath string) (Machine, error) {
	var m Machine
	data, err := ioutil.ReadFile(path.Join(machinePath, hostname+".yaml"))
	if err != nil {
		return Machine{}, err
	}
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return Machine{}, err
	}
	hostSlice := strings.Split(m.Hostname, ".")
	m.ShortName = hostSlice[0]
	m.Domain = strings.Join(hostSlice[1:], ".")
	return m, nil
}

// Render template among with machine and config struct
func (m Machine) renderTemplate(templatePath string, config Config) (string, error) {
	var tpl = pongo2.Must(pongo2.FromFile(templatePath))
	result, err := tpl.Execute(pongo2.Context{"machine": m, "config": config})
	if err != nil {
		return "", err
	}
	return result, err
}

// Posts machine macaddress to the forman proxy among with pxe configuration
func (m Machine) setBuildMode(config Config) error {
	pxeConfig, err := config.getPXEConfig(m)
	if err != nil {
		return err
	}

	foremanURL := fmt.Sprintf("%s/tftp/%s", config.ForemanProxyAddress, m.Network[0].MacAddress)
	_, err = http.PostForm(foremanURL, url.Values{"syslinux_config": {pxeConfig}})
	if err != nil {
		return err
	}
	return nil
}

// Sends DELETE to the forman-proxy tftp API removing the pxe configuration
func (m Machine) cancelBuildMode(config Config) error {
	foremanURL := fmt.Sprintf("%s/tftp/%s", config.ForemanProxyAddress, m.Network[0].MacAddress)
	req, _ := http.NewRequest("DELETE", foremanURL, nil)
	client := &http.Client{}
	_, err := client.Do(req)
	if err != nil {
		return err
	}
	return nil
}
