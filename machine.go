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
    "os/exec"
    "path"
    "strings"
    "time"    
    "syscall"
)

// Machine configuration
type Machine struct {
    Config `yaml:",inline"`
    Hostname        string
    ShortName       string
    Domain          string
    Token           string // This is set by the service
    Network         []Interface `yaml:"network"`
    Status           string
    BuildStart		time.Time
    RescueMode		bool
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


    pongo2.RegisterFilter("key", FilterGetValueByKey)

    hostname = strings.ToLower(hostname)
    hostSlice := strings.Split(hostname, ".")


    m := Machine{
        Hostname: hostname,
        ShortName: hostSlice[0],
        Domain: strings.Join(hostSlice[1:], "."),
    }

    // Merge in the "global" config.  The marshal/unmarshal combo looks funny, but it's clean and we aren't shooting for warp speed here.
    if c, err := yaml.Marshal(config); err == nil {
        if err = yaml.Unmarshal(c, &m); err != nil {
            return m, err
        }
    } else {
        return m, err
    }    

    // Then, load the domain definition.
    data, err := ioutil.ReadFile(path.Join(config.GroupPath, m.Domain+".yaml")) // apc03.prod.yaml

    if err != nil {
        if os.IsNotExist(err){
            data, err = ioutil.ReadFile(path.Join(config.GroupPath, m.Domain+".yml")) // Try .yml
            if err != nil && !os.IsNotExist(err) { // We should expect the file to not exist, but if it did exist, err happened for a different reason, then it should be reported.
                return m, err
            } else if os.IsNotExist(err) {
                log.Println("No group file found for " + m.Domain + ". Is that intentional?")
            }
        } else {
            return m, err
        }
    }

            
    if err = yaml.Unmarshal(data, &m); err != nil {
        return m, err
    }


    // Then load the machine definition.
    data, err = ioutil.ReadFile(path.Join(machinePath, hostname+".yaml")) // compute01.apc03.prod.yaml

    if err != nil {
        if os.IsNotExist(err) {
            data, err = ioutil.ReadFile(path.Join(machinePath, hostname+".yml")) // One more try but look for .yml
            if err != nil { // Whether the error was due to non-existence or something else, report it.  Machine definitions are must.
                return Machine{}, err
            }
        } else {
            return Machine{}, err
        }
    }
    
    err = yaml.Unmarshal(data, &m)
    if err != nil {
        return Machine{}, err
    }

    return m, nil
}

// Render templatewith machine and config struct
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

func (m Machine) setBuildMode(config Config, state State) (string, error) {
    // Generate a random token used to authenticate requests
    uuid, err := uuid.NewV4();
    
    if err != nil {
        return "", err
    }
    
    // Perform any desired operations needed prior to setting build mode.
    if err := m.RunBuildCommands(m.PreBuildCommands); err != nil {
        return "", err
    }
    
    
    state.Mux.Lock()

    state.Tokens[m.Hostname] = uuid.String()
    log.Println(fmt.Sprintf("%s installation token: %s", m.Hostname, state.Tokens[m.Hostname]))

    // Add token to machine struct
    m.Token = state.Tokens[m.Hostname]

    //Add to the Machine* tables
    state.MachineByUUID[uuid.String()] = &m
    state.MachineByMAC[fmt.Sprintf("%s", m.Network[0].MacAddress)] = &m
    state.MachineByHostname[m.Hostname] = &m
    m.BuildStart = time.Now()
    //Change machine state
    m.Status = "Installing"

    state.Mux.Unlock()

    return m.Token, nil
}

/*
should remove the machines mac address from the MachineBy* maps
which stops waitron from serving the PixieConfig used by pixiecore.
Runs any configured commands for normal build completion.
*/
func (m Machine) doneBuildMode(config Config, state State) error {
    
    state.Mux.Lock()
    //Delete mac from the building map
    delete(state.MachineByHostname, fmt.Sprintf("%s", m.Hostname))
    delete(state.MachineByMAC, fmt.Sprintf("%s", m.Network[0].MacAddress))
    delete(state.MachineByUUID, m.Token)
    
    //Change machine state
    m.Status = "Installed"
    state.Mux.Unlock()

    // Perform any desired operations needed after a machine has been taken out of build mode because install has completed.
    err := m.RunBuildCommands(m.PostBuildCommands)
        
    return err
}


/*
Should remove the machines mac address from the MachineBuild map
which stops waitron from serving the PixieConfig used by pixiecore.
Runs any configured commands for requested cancellations.
*/
func (m Machine) cancelBuildMode(config Config, state State) error {
    
    state.Mux.Lock()
    //Delete mac from the building map
    delete(state.MachineByHostname, fmt.Sprintf("%s", m.Hostname))
    delete(state.MachineByMAC, fmt.Sprintf("%s", m.Network[0].MacAddress))
    delete(state.MachineByUUID, m.Token)
    
    //Change machine state
    m.Status = "Terminated"
    state.Mux.Unlock()

    // Perform any desired operations needed after a machine has been taken out of build mode by request.
    err := m.RunBuildCommands(m.CancelBuildCommands)
        
    return err
}

// Builds pxe config to be sent to pixiecore
func (m Machine) pixieInit() (PixieConfig, error) {
    pixieConfig := PixieConfig{}
    
    var cmdline, imageURL, kernel, initrd string
        
    if m.RescueMode {
    	cmdline = m.RescueCmdline
    	imageURL = m.RescueImageURL
    	kernel = m.RescueKernel
    	initrd = m.RescueInitrd
    } else {
    	cmdline = m.Cmdline
    	imageURL = m.ImageURL
    	kernel = m.Kernel  	
    	initrd = m.Initrd    	
    }
    
    tpl, err := pongo2.FromString(cmdline)
    if err != nil {
        return pixieConfig, err
    }
    
    cmdline, err = tpl.Execute(pongo2.Context{"machine": m, "BaseURL": m.BaseURL, "Hostname": m.Hostname, "Token": m.Token})
    if err != nil {
        return pixieConfig, err
    }

    pixieConfig.Kernel = imageURL+kernel
    pixieConfig.Initrd = []string{imageURL+initrd}
    pixieConfig.Cmdline = cmdline

    return pixieConfig, nil
}

// This should ensure that even commands that spawn child processes are cleaned up correctly, along with their children.
func (m Machine) TimedCommandOutput(timeout time.Duration, command string) (out []byte, err error) {
    cmd := exec.Command("bash", "-c", command)
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

    time.AfterFunc(timeout, func() {
        syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
    })

    out, err = cmd.Output()
   
    return out, err
}

func (m Machine) RunBuildCommands(b []BuildCommand) error {
    for _, buildCommand := range b {

        if buildCommand.TimeoutSeconds == 0 {
            buildCommand.TimeoutSeconds = 5
        }
    
        tpl, err := pongo2.FromString(buildCommand.Command)
        if err != nil {
            return err
        }
        
        cmdline, err := tpl.Execute(pongo2.Context{"machine": m, "Token": m.Token})

        if buildCommand.ShouldLog {
            log.Println(cmdline)
        }
    
        if err != nil {
            return err
        }
        
        // Now actually execute the command and return err if ErrorsFatal
        out, err := m.TimedCommandOutput(time.Duration(buildCommand.TimeoutSeconds) * time.Second, cmdline)

        if err != nil && buildCommand.ErrorsFatal {
            return errors.New(err.Error() + ":" + string(out))
        }
    }
    
    return nil
}
