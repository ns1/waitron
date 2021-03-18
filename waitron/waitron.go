package waitron

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"waitron/config"
	"waitron/inventory_plugins"
	"waitron/machine"

	"github.com/flosch/pongo2"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

// PixieConfig boot configuration
type PixieConfig struct {
	Kernel  string   `json:"kernel" description:"The kernel file"`
	Initrd  []string `json:"initrd"`
	Cmdline string   `json:"cmdline"`
}

type Jobs struct {
	sync.RWMutex  `json:"-"`
	jobByToken    map[string]*Job
	jobByMAC      map[string]*Job
	jobByHostname map[string]*Job
}

type JobsHistory struct {
	sync.RWMutex `json:"-"`
	jobByToken   map[string]*Job // By Token
}

type Job struct {
	Start time.Time
	End   time.Time

	sync.RWMutex `json:"-"`
	Status       string
	StatusReason string

	Type    *config.BuildType
	Machine *machine.Machine
	Token   string // This is set by the service
}

type Waitron struct {
	config  config.Config
	jobs    Jobs
	history JobsHistory

	done chan struct{}
	wg   sync.WaitGroup

	activePlugins map[string]inventory_plugins.MachineInventoryPlugin
}

func FilterGetValueByKey(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {

	m := in.Interface().(map[string]string)

	if val, ok := m[param.String()]; ok {
		return pongo2.AsValue(val), nil
	} else {
		return pongo2.AsValue(""), nil
	}
}

func init() {

	pongo2.RegisterFilter("key", FilterGetValueByKey)
}

func New(c config.Config) *Waitron {
	w := &Waitron{
		config:  c,
		jobs:    Jobs{},
		history: JobsHistory{},
		done:    make(chan struct{}, 1),
		wg:      sync.WaitGroup{},
	}

	w.history.jobByToken = make(map[string]*Job)

	w.jobs.jobByToken = make(map[string]*Job)
	w.jobs.jobByMAC = make(map[string]*Job)
	w.jobs.jobByHostname = make(map[string]*Job)

	return w
}

func (w *Waitron) initPlugins() error {
	for _, cp := range w.config.MachineInventoryPlugins {
		if cp.Enabled {

			p, err := inventory_plugins.GetPlugin(cp.Name, &cp, &w.config)

			if err != nil {
				return err
			}

			if err = p.Init(); err != nil {
				return err
			}

			w.activePlugins[cp.Name] = p
		}
	}
	return nil
}

func (w *Waitron) Run() error {

	if err := w.initPlugins(); err != nil {
		return err
	}

	if w.config.StaleBuildCheckFrequency <= 0 {
		w.config.StaleBuildCheckFrequency = 300
	}

	ticker := time.NewTicker(time.Duration(w.config.StaleBuildCheckFrequency) * time.Second)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer ticker.Stop()
		for {
			select {
			case _, _ = <-w.done:
				ticker.Stop()
				return
			case <-ticker.C:
				w.checkForStaleJobs()
			}
		}
	}()

	return nil
}

func (w *Waitron) Stop() {
	close(w.done) // Was going to use <- struct{}{} since the use case is so simple but figured close() will get my attention if we make sync-related changes in the future.
	w.wg.Wait()
}

func (w *Waitron) checkForStaleJobs() {

	staleJobs := make([]*Job, 0)

	w.jobs.RLock()
	for _, j := range w.jobs.jobByToken {
		if j.Machine.StaleBuildThresholdSeconds > 0 && int(time.Now().Sub(j.Start).Seconds()) >= j.Machine.StaleBuildThresholdSeconds {
			staleJobs = append(staleJobs, j)
		}
	}
	w.jobs.RUnlock()

	for _, j := range staleJobs {
		go func() {
			if err := w.runBuildCommands(j, j.Machine.StaleBuildCommands); err != nil {
				log.Print(err)
			}
		}()
	}
}

// This should ensure that even commands that spawn child processes are cleaned up correctly, along with their children.
func (w *Waitron) timedCommandOutput(timeout time.Duration, command string) (out []byte, err error) {
	cmd := exec.Command("bash", "-c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	time.AfterFunc(timeout, func() {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	})

	out, err = cmd.Output()

	return out, err
}

func (w *Waitron) runBuildCommands(j *Job, b []config.BuildCommand) error {
	for _, buildCommand := range b {

		if buildCommand.TimeoutSeconds == 0 {
			buildCommand.TimeoutSeconds = 5
		}

		tpl, err := pongo2.FromString(buildCommand.Command)
		if err != nil {
			return err
		}

		j.RLock()
		cmdline, err := tpl.Execute(pongo2.Context{"machine": j.Machine, "Token": j.Token})
		j.RUnlock()

		if buildCommand.ShouldLog {
			log.Println(cmdline)
		}

		if err != nil {
			return err
		}

		// Now actually execute the command and return err if ErrorsFatal
		out, err := w.timedCommandOutput(time.Duration(buildCommand.TimeoutSeconds)*time.Second, cmdline)

		if err != nil && buildCommand.ErrorsFatal {
			return errors.New(err.Error() + ":" + string(out))
		}
	}

	return nil
}

// buildType can be normal, rescue, etc.
// Waitron can load a table from config of build_types with separate definitions, which can include whether "stale" make sense, so we can stop stale alerts for rescued machines.
func (w *Waitron) Build(hostname string, buildTypeName string) (string, error) {
	// Since the details of a BuildType can also exist directly in the root config,
	// An empty buildtype can be assumed to mean we'll use that.
	// But, it's important to remember that things will be merged, and using the root config as a "default"
	// Might give you more items in pre/post/stale/cancel command lists than expected.
	// Build type will be passed in
	// Build type is how we will know what specific pre-build commands exist
	// Groups and Machines can also have specific pre-build commands, but this should all be handled by how we merge in the configs starting at config->group->machine
	// We can also allow build-type to come from the config of the machine itself.
	// If present, we should be merging on top of that build type and not the one passed in herethen have to "rebase" the machine onto the build type it's requesting.
	// If not present, then it will be set from buildType - This must happen so that when the macaddress comes in for the pxe config, we will know what to serve.
	/*
		Check for the existence of build type in the compiled machine details.
		Take the global config and create a Machine from that.
		Take the build type and merge on top of that
		Merge the previous, compiled Machine object into that so that Machine-specific config takes precedence over the global and build-type config.
		Generate a job UUID
		Create a *Job using the UUID, the final *machine.Machine, and the *BuildType
		Add that job to w.jobs - I'll have hostname, token, and uuid.
		Take all MacAddress found in *machine.Machine->Network and register them all in JobByMac.

	*/

	// Generate a job token, which can optionally be used to authenticate requests.
	token := uuid.New().String()

	log.Println(fmt.Sprintf("%s job token: %s", hostname, token))

	hostname = strings.ToLower(hostname)

	baseMachine := &machine.Machine{}

	foundMachine, err := w.GetMergedMachine(hostname)

	// Merge in the "global" config.  The marshal/unmarshal combo looks funny, but it's clean and we aren't shooting for warp speed here.
	if c, err := yaml.Marshal(w.config); err == nil {
		if err = yaml.Unmarshal(c, baseMachine); err != nil {
			return "", err
		}
	} else {
		return "", err
	}

	// Merge in the build type, but allow machines to select their own build type.
	if foundMachine.BuildTypeName != "" {
		buildTypeName = foundMachine.BuildTypeName
	}

	if buildTypeName != "" {
		if b, err := yaml.Marshal(w.config.BuildTypes[buildTypeName]); err == nil {
			if err = yaml.Unmarshal(b, baseMachine); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	// Finally, merge in the machine-specific details.
	if f, err := yaml.Marshal(foundMachine); err == nil {
		if err = yaml.Unmarshal(f, baseMachine); err != nil {
			return "", err
		}
	} else {
		return "", err
	}

	if err != nil {
		return "", err
	}

	j := &Job{
		Start:        time.Now(),
		RWMutex:      sync.RWMutex{},
		Status:       "pending",
		StatusReason: "",
		Machine:      baseMachine,
		Token:        token,
	}

	// Perform any desired operations needed prior to setting build mode.
	if err := w.runBuildCommands(j, j.Machine.PreBuildCommands); err != nil {
		return "", err
	}

	// normalize interface MAC addresses
	macs := make([]string, 0, len(j.Machine.Network))
	r := strings.NewReplacer(":", "", "-", "", ".", "")

	for i := 0; i < len(j.Machine.Network); i++ {
		macs = append(macs, strings.ToLower(r.Replace(j.Machine.Network[i].MacAddress)))
	}

	err = w.addJob(j, token, hostname, macs)

	return token, nil
}

func (w *Waitron) GetMergedMachine(hostname string) (*machine.Machine, error) {
	/*
			Take the hostname and start looping through the inventory plugins
			Merge details as you get them into a single, compiled Machine object


		m := &machine.Machine{}

		for _, i := range w.config.MachineInventoryPlugins {

		}*/

	return nil, nil
}

func (w *Waitron) GetMachineStatus(hostname string) (string, error) {
	j, err := w.getJob(hostname, "")
	if err != nil {
		return "unknown", err
	}

	j.RLock()
	defer j.RUnlock()

	return j.Status, nil
}

func (w *Waitron) GetJobStatus(token string) (string, error) {
	j, err := w.getJob(token, "")
	if err != nil {
		return "unknown", err
	}

	j.RLock()
	defer j.RUnlock()

	return j.Status, nil
}

func (w *Waitron) addJob(j *Job, token string, hostname string, macs []string) error {
	w.jobs.Lock()
	defer w.jobs.Unlock()

	w.jobs.jobByToken[token] = j
	w.jobs.jobByHostname[hostname] = j

	for _, mac := range macs {
		w.jobs.jobByHostname[mac] = j
	}

	return nil
}

func (w *Waitron) getJob(hostname string, token string) (*Job, error) {
	w.jobs.RLock()
	defer w.jobs.RUnlock()

	var j *Job
	found := false

	// If both are passed, check that they both point to the same job.

	if hostname != "" {
		j, found = w.jobs.jobByHostname[hostname]
	}

	if token != "" {
		jAgain, foundAgain := w.jobs.jobByToken[token]

		if (found && foundAgain) && j != jAgain {
			return nil, errors.New("hostname/Job mismatch")
		}

		found = foundAgain
		j = jAgain
	}

	if !found {
		return nil, fmt.Errorf("job not found: '%s' '%s' ", hostname, token)
	}

	return j, nil

}

func (w *Waitron) GetPxeConfig(macaddress string) (PixieConfig, error) {

	// Look up the *Job by MAC
	// Build the pxe config based on the compiled machine details.

	w.jobs.RLock()
	j, found := w.jobs.jobByMAC[macaddress]
	w.jobs.RUnlock()

	if !found {
		return PixieConfig{}, fmt.Errorf("job not found for  '%s'", macaddress)
	}

	pixieConfig := PixieConfig{}

	var cmdline, imageURL, kernel, initrd string

	j.RLock()

	cmdline = j.Machine.Cmdline
	imageURL = j.Machine.ImageURL
	kernel = j.Machine.Kernel
	initrd = j.Machine.Initrd

	tpl, err := pongo2.FromString(cmdline)
	if err != nil {
		return pixieConfig, err
	}

	cmdline, err = tpl.Execute(pongo2.Context{"machine": j.Machine, "BaseURL": j.Machine.BaseURL, "Hostname": j.Machine.Hostname, "Token": j.Token})
	if err != nil {
		j.RUnlock()
		j.Lock()
		j.Status = "failed"
		j.StatusReason = "pxe config build failed"
		j.Unlock()
		return pixieConfig, err
	}

	j.RUnlock()

	pixieConfig.Kernel = imageURL + kernel
	pixieConfig.Initrd = []string{imageURL + initrd}
	pixieConfig.Cmdline = cmdline
	return pixieConfig, nil
}

func (w *Waitron) cleanUpJob(j *Job, status string) error {
	// Take the list of all macs found in that Jobs Machine->Network
	// Use host, token, and list of MACs to clean out the details from Jobs

	j.Lock()
	j.Status = status
	j.End = time.Now()
	j.Unlock()

	j.RLock()
	defer j.RUnlock()

	w.jobs.Lock()
	defer w.jobs.Unlock()

	for _, iface := range j.Machine.Network {
		delete(w.jobs.jobByMAC, iface.MacAddress)
	}

	delete(w.jobs.jobByToken, j.Token)
	delete(w.jobs.jobByHostname, j.Machine.Hostname)

	return nil
}

func (w *Waitron) FinishBuild(hostname string, token string) error {

	j, err := w.getJob(hostname, token)

	if err != nil {
		return err
	}

	if err := w.runBuildCommands(j, j.Machine.PostBuildCommands); err != nil {
		return err
	}

	// Run clean-up if all finish commands were successful (or non-fatal).
	return w.cleanUpJob(j, "completed")
}

func (w *Waitron) CancelBuild(hostname string, token string) error {

	j, err := w.getJob(hostname, token)

	if err != nil {
		return err
	}

	if err := w.runBuildCommands(j, j.Machine.CancelBuildCommands); err != nil {
		return err
	}

	// Run clean-up if all cancel commands were successful (or non-fatal).
	return w.cleanUpJob(j, "terminated")
}

func (w *Waitron) CleanHistory() error {
	// Loop through all items in JobsHistory and check existence in JobByToken
	// If not found, it's either completed or terminated and can be cleaned out.
	w.history.Lock()
	defer w.history.Unlock()

	w.jobs.RLock()
	defer w.jobs.RUnlock()

	for token := range w.history.jobByToken {
		if _, found := w.jobs.jobByToken[token]; !found {
			delete(w.history.jobByToken, token)
		}
	}

	return nil
}

func (w *Waitron) GetJobsHistoryBlob() ([]byte, error) {
	w.history.RLock()
	defer w.history.RUnlock()

	// Seems efficient...
	// https://github.com/golang/go/blob/0bd308ff27822378dc2db77d6dd0ad3c15ed2e08/src/runtime/map.go#L118
	if len(w.history.jobByToken) == 0 {
		return []byte("[]"), nil
	}

	// Each of the jobs in here needs to be RLock'ed as they are processed.
	// I need to loop through them.  Just Marshal'ing the history isn't acceptable. :(

	// This is simple but seems kind of dumb, but every suggested solution wen't crazy with marshal and unmarshal,
	// which also seems dumb here but less simple. Did I miss something silly?
	blob := make([]byte, 1, 256*len(w.history.jobByToken))
	blob[0] = '['

	for _, job := range w.history.jobByToken {

		job.RLock()
		b, err := json.Marshal(job)
		job.RUnlock()

		if err != nil {
			return b, err
		}

		blob = append(blob, ',')
		blob = append(blob, b...) // So it's not _quite_ as bad as it looks? --> https://stackoverflow.com/questions/16248241/concatenate-two-slices-in-go#comment40751903_16248257
	}

	blob = append(blob, ']')
	blob[1] = ' ' // Get rid of that prepended comma of the first item.

	return blob, nil
}

//  Can be used to query for one or more machines.
func (w *Waitron) GetMachines(hostnames []string, macs []string) ([]*machine.Machine, error) {
	ms := make([]*machine.Machine, 1, 10)
	// Loop through inventory plugins and query by hostnames and macs (really only one should be set).
	// Compile a list of found machines
	// Empty hostnames and empty macs list means get all
	return ms, nil
}

func (w *Waitron) RenderStageTemplate(token string, template string) (string, error) {

	j, err := w.getJob(token, "")
	if err != nil {
		return "unknown", err
	}

	j.RLock()
	defer j.RUnlock()

	// Render preseed as default
	if template == "finish" {
		template = j.Machine.Finish
	} else {
		template = j.Machine.Preseed
	}

	return w.renderTemplate(template, j)
}

func (w *Waitron) renderTemplate(templateName string, j *Job) (string, error) {

	templateName = path.Join(w.config.TemplatePath, templateName)
	if _, err := os.Stat(templateName); err != nil {
		return "", errors.New("Template does not exist")
	}

	var tpl = pongo2.Must(pongo2.FromFile(templateName))
	result, err := tpl.Execute(pongo2.Context{"machine": j.Machine, "config": w.config})
	if err != nil {
		return "", err
	}
	return result, err
}
