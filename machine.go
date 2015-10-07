package main

import (
	"errors"
	"fmt"
	"github.com/flosch/pongo2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
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

// PXE boot configuration
type Pixie struct {
	Kernel  string   `json:"kernel"`
	Initrd  []string `json:"initrd"`
	Cmdline string   `json:"cmdline"`
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
func (m Machine) renderTemplate(template string, config Config) (string, error) {

	template = path.Join(config.TemplatePath, template)
	if _, err := os.Stat(template); err != nil {
		return "", errors.New("Template does not exist")
	}

	var tpl = pongo2.Must(pongo2.FromFile(template))
	result, err := tpl.Execute(pongo2.Context{"machine": m, "config": config})
	if err != nil {
		return "", err
	}
	return result, err
}

// Posts machine macaddress to the forman proxy among with pxe configuration
func (m Machine) setBuildMode(config Config, pxeConfig string) error {
	foremanURL := fmt.Sprintf("%s/tftp/%s", config.ForemanProxyAddress, m.Network[0].MacAddress)
	result, err := http.PostForm(foremanURL, url.Values{"syslinux_config": {pxeConfig}})
	if err != nil {
		return err
	}
	if result.StatusCode != 200 {
		return errors.New("foreman-proxy responded with a non 200 exit code")
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

// Builds pxe config to be sent to pixiecore
func (m Machine) pixieInit(config Config) (Pixie, error) {
	var p Pixie

	p.Kernel = config.ImageURL + config.Kernel
	p.Initrd = []string{config.ImageURL + config.Initrd}
	p.Cmdline = "interface=auto url=" + config.BaseURL + "/" + m.Hostname + "/preseed/" + m.Token + " ramdisk_size=10800 root=/dev/rd/0 rw auto hostname=" + m.Hostname + " console-setup/ask_detect=false console-setup/layout=USA console-setup/variant=USA keyboard-configuration/layoutcode=us localechooser/translation/warn-light=true localechooser/translation/warn-severe=true locale=en_US"

	return p, nil
}
