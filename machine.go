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
	Network         []Interface `yaml:"network"`
	Params          map[string]string
	ImageURL        string `yaml:"image_url"`
	Kernel          string
	Initrd          string
	Cmdline         string
}

type IPConfig struct {
	IPAddress  string `yaml:"ipaddress"`
	Netmask    string `yaml:"netmask"`
	Cidr       string `yaml:"cidr"`
}

// Interface Configuration
type Interface struct {
	Name       string `yaml:"name"`
	Addresses4 []IPConfig `yaml:"addresses4"`
	Addresses6 []IPConfig `yaml:"addresses6"`
	MacAddress string `yaml:"macaddress"`
	Gateway4    string `yaml:"gateway4"`
	Gateway6    string `yaml:"gateway6"`
}

// PixieConfig boot configuration
type PixieConfig struct {
	Kernel  string   `json:"kernel" description:"The kernel file"`
	Initrd  []string `json:"initrd"`
	Cmdline string   `json:"cmdline"`
}

func FilterGetValueByKey(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
        m := in.Interface().(map[string]string)
        if val, ok := m[param.String()]; ok {
	        return pongo2.AsValue(val), nil
        } else {
        	return pongo2.AsValue(""), nil
        }
}


func machineDefinition(hostname string, machinePath string, config Config) (Machine, error) {


	hostname = strings.ToLower(hostname)

	pongo2.RegisterFilter("key", FilterGetValueByKey)

	// First load a machine definition.
	var m Machine
	data, err := ioutil.ReadFile(path.Join(machinePath, hostname+".yaml")) // compute01.apc03.prod.yml
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
	
	// Then, load the domain definition.
	data, err = ioutil.ReadFile(path.Join(machinePath, m.Domain+".yaml")) // apc03.prod.yml
	if err != nil {
		if !os.IsNotExist(err) { // We should expect the file to frequently not exist, but if it did exist, err happened for a different reason, then it should be reported. 
			return m, err
		}
	} else {
		if err = yaml.Unmarshal(data, &m); err != nil {
			return m, err
		}
	}
	
	// Then, Merge in the "global" config.  The marshal/unmarshal combo looks funny, but it's clean and we aren't shooting for warp speed here.
	if c, err := yaml.Marshal(config); err == nil {
	    if err = yaml.Unmarshal(c, &m); err != nil {
	    	return m, err
	    }
	} else {
		return m, err
	}	
	
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
func (m Machine) setBuildMode(config Config, state State) error {
	// Generate a random token used to authenticate requests
	uuid, err := uuid.NewV4();
	
	if err != nil {
		return err
	}
	
	state.Tokens[m.Hostname] = uuid.String()
	log.Println(fmt.Sprintf("%s installation token: %s", m.Hostname, state.Tokens[m.Hostname]))
	// Add token to machine struct
	m.Token = state.Tokens[m.Hostname]
	//Add to the MachineBuild table
	state.MachineBuild[fmt.Sprintf("%s", m.Network[0].MacAddress)] = m.Hostname
	//Change machine state
	state.MachineState[m.Hostname] = "Installing"
	return nil
}

/*
cancelBuild remove the machines mac address from the MachineBuild map
which stops waitron from serving the PixieConfig used by pixiecore
*/
func (m Machine) cancelBuildMode(config Config, state State) error {
	//Delete mac from the building map
	delete(state.MachineBuild, fmt.Sprintf("%s", m.Network[0].MacAddress))
	//Change machine state
	state.MachineState[m.Hostname] = "Installed"
	return nil
}

// Return string2 if string1 is empty
func defaultString(string1 string, string2 string) string {
	if string1 == "" {
		return string2
	}
	return string1
}

// Builds pxe config to be sent to pixiecore
func (m Machine) pixieInit(config Config) (PixieConfig, error) {
	pixieConfig := PixieConfig{}
	tpl, err := pongo2.FromString(m.Cmdline)
	if err != nil {
		return pixieConfig, err
	}
	cmdline, err := tpl.Execute(pongo2.Context{"machine": m, "BaseURL": config.BaseURL, "Hostname": m.Hostname, "Token": m.Token})
	if err != nil {
		return pixieConfig, err
	}

	pixieConfig.Kernel = m.ImageURL+m.Kernel
	pixieConfig.Initrd = []string{m.ImageURL+m.Initrd}
	pixieConfig.Cmdline = cmdline
	
	log.Println(pixieConfig)
	log.Println(m)
	
	return pixieConfig, nil
}
