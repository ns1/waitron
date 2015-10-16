package main

import (
	"errors"
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/satori/go.uuid"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
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
	ImageURL        string `yaml:"image_url"`
	Kernel          string
	Initrd          string
	Cmdline         string
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
	Kernel  string   `json:"kernel" description:"The kernel file"`
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
func (m Machine) setBuildMode(config Config) error {
	// Generate a random token used to authenticate requests
	config.Token[m.Hostname] = uuid.NewV4().String()
	log.Println(fmt.Sprintf("%s installation token: %s", m.Hostname, config.Token[m.Hostname]))
	// Add token to machine struct
	m.Token = config.Token[m.Hostname]
	//Add to the MachineBuild table
	config.MachineBuild[fmt.Sprintf("%s", m.Network[0].MacAddress)] = m.Hostname
	//Change machine state
	config.MachineState[m.Hostname] = "Installing"
	return nil
}

// Sends DELETE to the forman-proxy tftp API removing the pxe configuration
func (m Machine) cancelBuildMode(config Config) error {
	//Delete mac from the building map
	delete(config.MachineBuild, fmt.Sprintf("%s", m.Network[0].MacAddress))
	//Change machine state
	config.MachineState[m.Hostname] = "Installed"

	return nil
}

// Builds pxe config to be sent to pixiecore
func (m Machine) pixieInit(config Config) (Pixie, error) {
	var p Pixie

	p.Kernel = m.ImageURL + m.Kernel
	p.Initrd = []string{m.ImageURL + m.Initrd}

	tpl, err := pongo2.FromString(m.Cmdline)
	if err != nil {
		return Pixie{}, err
	}
	out, err := tpl.Execute(pongo2.Context{"BaseURL": config.BaseURL, "Hostname": m.Hostname, "Token": m.Token})
	if err != nil {
		return Pixie{}, err
	}
	p.Cmdline = out

	return p, nil
}
