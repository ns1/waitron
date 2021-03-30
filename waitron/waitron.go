package waitron

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"waitron/config"
	"waitron/inventoryplugins"
	"waitron/machine"

	"github.com/flosch/pongo2"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

/*
	TODO:
		Figure out logging.

		Take a look at what actually needs to be exported here.  Seems like not much, so either
		move some of the Job* stuff to a separate package and make the rest of the fields public, or stop exporting the struct and also just make the properties private.
*/

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
	jobByToken   map[string]*Job
}

type Job struct {
	Start time.Time
	End   time.Time

	sync.RWMutex `json:"-"`
	Status       string
	StatusReason string

	BuildTypeName        string
	Machine              *machine.Machine
	TriggerMacRaw        string // The MAC that actually came in looking for a PXE boot.
	TriggerMacNormalized string
	Token                string
}

type activePlugin struct {
	plugin   inventoryplugins.MachineInventoryPlugin
	settings *config.MachineInventoryPluginSettings
}

type Waitron struct {
	config  *config.Config
	jobs    Jobs
	history JobsHistory

	historyBlobLastCached time.Time
	historyBlobCache      []byte

	done chan struct{}
	wg   sync.WaitGroup

	activePlugins []activePlugin

	logs chan string
}

func FilterRegexReplace(in *pongo2.Value, inR *pongo2.Value) (*pongo2.Value, *pongo2.Error) {

	s := in.String()
	rgxRpl, ok := inR.Interface().([]string)

	if !ok {
		return in, nil
	}

	re, err := regexp.Compile(rgxRpl[0])

	if err != nil {
		return pongo2.AsValue(""), &pongo2.Error{Sender: "filter:regex_replace", OrigError: err}
	}

	return pongo2.AsValue(re.ReplaceAllString(s, rgxRpl[1])), nil
}

func FilterFromYaml(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {

	s := in.String()

	out := make(map[interface{}]interface{})

	if err := yaml.Unmarshal([]byte(s), out); err != nil {
		return nil, &pongo2.Error{Sender: "filter:from_yaml", OrigError: err}
	}

	return pongo2.AsSafeValue(out), nil
}

func init() {

	pongo2.RegisterFilter("regex_replace", FilterRegexReplace)
	pongo2.RegisterFilter("from_yaml", FilterFromYaml)
}

func New(c *config.Config) *Waitron {
	w := &Waitron{
		config:                c,
		jobs:                  Jobs{},
		history:               JobsHistory{},
		historyBlobLastCached: time.Time{},
		done:                  make(chan struct{}, 1),
		wg:                    sync.WaitGroup{},
		activePlugins:         make([]activePlugin, 0, 1),
		logs:                  make(chan string, 1000),
	}

	w.history.jobByToken = make(map[string]*Job)

	w.jobs.jobByToken = make(map[string]*Job)
	w.jobs.jobByMAC = make(map[string]*Job)
	w.jobs.jobByHostname = make(map[string]*Job)

	return w
}

/*
	Just some quick and dirty buffered logging.  This function can/should/will be passed around to plugins.
*/
func (w *Waitron) addLog(s string, l config.LogLevel) bool {

	if l > w.config.LogLevel {
		return true
	}

	select {
	case w.logs <- fmt.Sprintf("[%s] %s", l, s):
		return true
	default:
		return false
	}
}

/*********** super hacky, sorry...  *************/
type WaitronLogger struct {
	wf func(string, config.LogLevel) bool
}

func (wl WaitronLogger) Write(b []byte) (int, error) {

	if wl.wf(string(b), config.LogLevelInfo) {
		return len(b), nil
	} else {
		return 0, fmt.Errorf("log channel is full")
	}

}
func (w *Waitron) GetLogger() WaitronLogger {
	return WaitronLogger{wf: w.addLog}
}

/************************************************/

/*
	Create an array of plugin instances.  Only enabled/active plugins will be loaded.
*/
func (w *Waitron) initPlugins() error {
	for idx := 0; idx < len(w.config.MachineInventoryPlugins); idx++ { // for-range and pointers don't mix.

		cp := &(w.config.MachineInventoryPlugins[idx])

		if !cp.Disabled {

			p, err := inventoryplugins.GetPlugin(cp.Name, cp, w.config, w.addLog)

			if err != nil {
				return err
			}

			if err = p.Init(); err != nil {
				return err
			}

			w.activePlugins = append(w.activePlugins, activePlugin{plugin: p, settings: cp})
		}
	}
	return nil
}

/*
	Perform any init work that needs to be done before running things.
*/
func (w *Waitron) Init() error {

	if err := w.initPlugins(); err != nil {
		return err
	}

	return nil
}

/*
	Start up any necessary go-routines.
*/
func (w *Waitron) Run() error {

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

	w.wg.Add(1)
	go func() {
		w.wg.Done()
		for lm := range w.logs {
			log.Print(lm)
			select {
			case <-w.done:
				return
			default:
			}
		}

	}()

	return nil
}

/*
	Broadcast "done" and wait for any go-routines to return.
*/
func (w *Waitron) Stop() error {
	close(w.done) // Was going to use <- struct{}{} since the use case is so simple but figured close() will get my attention if we make sync-related changes in the future.
	w.wg.Wait()

	return nil
}

/*
	Loop through all active jobs and run stale-commands for any that have crossed their StaleBuildThresholdSeconds
*/
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
				w.addLog(err.Error(), config.LogLevelError)
			}
		}()
	}
}

/*
	This should ensure that even commands that spawn child processes are cleaned up correctly, along with their children.
*/
func (w *Waitron) timedCommandOutput(timeout time.Duration, command string) ([]byte, error) {

	tmpfile, err := ioutil.TempFile(w.config.TempPath, "waitron.timedCommandOutput")
	if err != nil {
		return []byte{}, err
	}

	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.Write([]byte(command)); err != nil {
		return []byte{}, err
	}

	if err = tmpfile.Close(); err != nil {
		return []byte{}, err
	}

	if err = os.Chmod(tmpfile.Name(), 0700); err != nil {
		return []byte{}, err
	}

	// Fair credit: Decided to migrate to a compact version of github user abh's idea for a temp file vs straight to bash -c.
	// 				Not as nice for simple commands but more pleasant for large scripts.
	cmd := exec.Command(tmpfile.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	outP, err := cmd.StdoutPipe() // Set up the stdout pipe

	if err != nil {
		return []byte{}, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return []byte{}, err
	}

	// Grab the pid now that we've started and set up the timeout function.
	pid := cmd.Process.Pid
	time.AfterFunc(timeout, func() {
		syscall.Kill(-pid, syscall.SIGKILL)
	})

	out := make([]byte, 512, 512)
	n, err := outP.Read(out)

	if err != nil {
		return out, err
	}

	// Wait for the command to finish/terminate.
	if err := cmd.Wait(); err != nil {
		return []byte{}, err
	}

	return out[:n], nil
}

/*
	Loop through any passed in commands, render them, and execute them.
*/
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
		cmdline, err := tpl.Execute(pongo2.Context{"job": j, "machine": j.Machine, "token": j.Token})
		j.RUnlock()

		if err != nil {
			return err
		}

		if buildCommand.ShouldLog {
			w.addLog(cmdline, config.LogLevelInfo)
		}

		// Now actually execute the command and return err if ErrorsFatal
		out, err := w.timedCommandOutput(time.Duration(buildCommand.TimeoutSeconds)*time.Second, cmdline)

		if err != nil && err != io.EOF && buildCommand.ErrorsFatal {
			return errors.New(err.Error() + ":" + string(out))
		}
	}

	return nil
}

/*
	Create a register a new job for the specified hostname, and optionally the build type.
*/
func (w *Waitron) Build(hostname string, buildTypeName string) (string, error) {
	/*
		Since the details of a BuildType can also exist directly in the root config,
		an empty buildtype can be assumed to mean we'll use that.

		But, it's important to remember that things will be merged, and using the root config as a "default"
		might give you more items in pre/post/stale/cancel command lists than expected.

		Build type is how we will know what specific pre-build commands exist
		Machines can also have specific pre-build commands, but this should all be handled by how we merge in the configs starting at config->build-type->machine.

		We can also allow build-type to come from the config of the machine itself.

		If present, we should be merging on top of that build type and not the one passed in herethen have to "rebase" the machine onto the build type it's requesting.
		If not present, then it will be set from buildType - This must happen so that when the macaddress comes in for the pxe config, we will know what to serve.
	*/

	w.addLog(fmt.Sprintf("looking for already active job for '%s'", hostname), config.LogLevelDebug)

	// Error or not, if an existing job was found, no new job permitted.
	if _, found, _ := w.getActiveJob(hostname, ""); found {
		return "", fmt.Errorf("active job for '%s' must complete or be terminated before new job", hostname)
	}

	// Generate a job token, which can optionally be used to authenticate requests.
	token := uuid.New().String()

	w.addLog(fmt.Sprintf("%s job token generated: %s", hostname, token), config.LogLevelInfo)

	hostname = strings.ToLower(hostname)

	w.addLog(fmt.Sprintf("retrieving complied machine details for job %s", token), config.LogLevelDebug)

	// Get the compiled machine details from any config, build type, and plugins being used
	foundMachine, err := w.GetMergedMachine(hostname, "", buildTypeName)

	if err != nil {
		return "", err
	}

	// Prep the new Job
	j := &Job{
		Start:         time.Now(),
		RWMutex:       sync.RWMutex{},
		Status:        "pending",
		StatusReason:  "",
		Machine:       foundMachine,
		BuildTypeName: buildTypeName,
		Token:         token,
	}

	w.addLog(fmt.Sprintf("running pre-build commands for job %s", token), config.LogLevelDebug)

	// Perform any desired operations needed prior to setting build mode.
	if err := w.runBuildCommands(j, j.Machine.PreBuildCommands); err != nil {
		w.addLog(fmt.Sprintf("pre-build commands for %s returned errors %v", token, err), config.LogLevelDebug)
		return "", err
	}

	w.addLog(fmt.Sprintf("normalizing macs for job %s", token), config.LogLevelDebug)

	// normalize interface MAC addresses
	macs := make([]string, 0, len(j.Machine.Network))
	r := strings.NewReplacer(":", "", "-", "", ".", "")

	for i := 0; i < len(j.Machine.Network); i++ {
		if j.Machine.Network[i].MacAddress != "" {
			j.Machine.Network[i].MacAddress = strings.ToLower(r.Replace(j.Machine.Network[i].MacAddress))
			macs = append(macs, j.Machine.Network[i].MacAddress)
		}
	}

	w.addLog(fmt.Sprintf("adding job %s", token), config.LogLevelDebug)

	if err = w.addJob(j, token, hostname, macs); err != nil {
		return "", err
	}

	w.addLog(fmt.Sprintf("job %s added", token), config.LogLevelInfo)

	return token, nil
}

/*
	This produces a Machine with data compiled from all enabled plugins.
	This is not pulling data from Waitron.  It's pulling external data,
	compiling it, and returning that.
*/
func (w *Waitron) getMergedInventoryMachine(hostname string, mac string) (*machine.Machine, error) {
	m := &machine.Machine{}

	anyFound := false

	w.addLog(fmt.Sprintf("looping through %d active plugins", len(w.activePlugins)), config.LogLevelInfo)

	/*
		Take the hostname and start looping through the inventory plugins
		Merge details as you get them into a single, compiled Machine object
	*/
	maxWeightSeen := 0
	for _, ap := range w.activePlugins {

		/*
			If we've already found details in a higher-precedence plugins, there's no need to even check the current one.
			This would mean that a plugin of greater weight was executed AND returned data.
		*/
		if ap.settings.Weight < maxWeightSeen {
			continue
		}

		pm, err := ap.plugin.GetMachine(hostname, mac)

		if err != nil {
			w.addLog(fmt.Sprintf("failed to get machine from plugin in: %v", err), config.LogLevelInfo)
			return nil, err
		}

		if pm != nil {
			// Just keep merging in details that we find
			if b, err := yaml.Marshal(pm); err == nil {

				/*
					But if we are now working on the response from a plugin with a greater weight than all previous plugins that returned data,
					then we need to clobber all the previous data and let this current one replace it all.
				*/
				if ap.settings.Weight > maxWeightSeen {
					m = &machine.Machine{}
					maxWeightSeen = ap.settings.Weight
				}

				if err = yaml.Unmarshal(b, m); err != nil {
					return nil, err
				}
			} else {
				// Just log.  Don't let one plugin break everything.
				w.addLog(fmt.Sprintf("failed to marshal plugin data during machine merging: %v", err), config.LogLevelError)
				continue
			}

			/*
				We found details, but we've been told not to treat them as inidicitive of finding a true machine definition.
				I.e., the user probably wants this treated as supplmental information if a machine is found in some other plugin.
			*/
			if !ap.settings.SupplementalOnly {
				anyFound = true
			}
		}
	}

	// Bail out if we didn't find the machine anywhere.
	if !anyFound {
		w.addLog(fmt.Sprintf("machine not found in any non-supplemental plugin"), config.LogLevelDebug)
		return nil, nil
	}

	return m, nil
}

/*
  This produces the final merge machine with config and build type details.
*/
func (w *Waitron) GetMergedMachine(hostname string, mac string, buildTypeName string) (*machine.Machine, error) {

	/*
		We need the "merge" order to go config -> build type -> machine
		But a machine can also specify a build type for itself, which could have come from any of the available plugins,
		so we need to take the initial machine compile from plugins, then create a new base machine and start merging
		things in the order we want because we wouldn't know the true final build type until after the plugins have provided all the details.
	*/
	baseMachine := &machine.Machine{}

	foundMachine, err := w.getMergedInventoryMachine(hostname, mac)

	if err != nil {
		return nil, err
	}

	if foundMachine == nil {
		return nil, fmt.Errorf("'%s' '%s' not found using any active plugin", hostname, mac)
	}

	// Merge in the "global" config.  The marshal/unmarshal combo looks funny, but we've given up completely on speed at this point.
	if c, err := yaml.Marshal(w.config); err == nil {
		if err = yaml.Unmarshal(c, baseMachine); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	// Merge in the build type, but allow machines to select their own build type first.
	if foundMachine.BuildTypeName != "" {
		buildTypeName = foundMachine.BuildTypeName
	}

	if buildTypeName != "" {
		buildType, found := w.config.BuildTypes[buildTypeName]

		if !found {
			return nil, fmt.Errorf("build type '%s' not found", buildTypeName)
		}

		if b, err := yaml.Marshal(buildType); err == nil {
			if err = yaml.Unmarshal(b, baseMachine); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Finally, merge in the machine-specific details.
	if f, err := yaml.Marshal(foundMachine); err == nil {
		if err = yaml.Unmarshal(f, baseMachine); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	return baseMachine, nil
}

/*
	Retrieves the current ACTIVE job status for related to a hostname
*/
func (w *Waitron) GetMachineStatus(hostname string) (string, error) {
	j, _, err := w.getActiveJob(hostname, "")
	if err != nil {
		return "", err
	}

	j.RLock()
	defer j.RUnlock()

	return j.Status, nil
}

/*
	Retrieves the job status related to a job token if it's currently active.
*/
func (w *Waitron) GetActiveJobStatus(token string) (string, error) {
	j, _, err := w.getActiveJob("", token)
	if err != nil {
		return "", err
	}

	j.RLock()
	defer j.RUnlock()

	return j.Status, nil
}

/*
	Retrieves the job status related to a job token, whether or not it's current active
*/
func (w *Waitron) GetJobStatus(token string) (string, error) {
	w.history.RLock()
	defer w.history.RUnlock()

	j, found := w.history.jobByToken[token]

	if !found {
		return "", fmt.Errorf("job '%s' not found", token)
	}

	j.RLock()
	defer j.RUnlock()

	return j.Status, nil
}

/*
	Adds a new build job
*/
func (w *Waitron) addJob(j *Job, token string, hostname string, macs []string) error {
	w.jobs.Lock()
	defer w.jobs.Unlock()

	w.jobs.jobByToken[token] = j
	w.jobs.jobByHostname[hostname] = j

	for _, mac := range macs {
		w.jobs.jobByMAC[mac] = j
	}

	w.history.Lock()
	w.history.jobByToken[token] = j
	w.history.Unlock()

	return nil
}

/*
	Retrieves the job struct to a job token or hostname if it's currently active.
	If hostname and token are both passed, they much point to the same job.
*/
func (w *Waitron) getActiveJob(hostname string, token string) (*Job, bool, error) {
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
			return nil, found, errors.New("hostname/Job mismatch")
		}

		found = foundAgain
		j = jAgain
	}

	if !found {
		return nil, found, fmt.Errorf("job not found: '%s' '%s' ", hostname, token)
	}

	return j, found, nil

}

/*
	This handles special cases if requested by the config used with Waitron.
	If there doesn't appear to be any job associated with a MAC but the config contains the '_unknown'_
	build type, Waitron will serve what it has, but it won't perform any status tracking.
	This is simply a hook to allow power users to load in special "registration" OS images that they can use
	to, for example, collect and register machine details for new machines into their inventory management system.
*/
func (w *Waitron) getPxeConfigForUnknown(b *config.BuildType, macaddress string) (PixieConfig, error) {

	m, err := w.getMergedInventoryMachine("", macaddress)

	if err != nil {
		return PixieConfig{}, err
	}

	// _unknown_ is only for machines we don't know about at all.
	if m != nil {
		return PixieConfig{}, fmt.Errorf("job not found for  '%s' and _unknown_ builds not requested", macaddress)
	}

	w.addLog(fmt.Sprintf("running unknown-build commands for job %s", macaddress), config.LogLevelDebug)

	// Perform any desired operations when an unknown MAC is seen.
	if len(w.config.UnknownBuildCommands) > 0 {
		/*
			I don't want runBuildCommands to accept an empty interface.
			For now, at least, I'd prefer sending in a nearly empty job and repurposing the Token field to send the MAC
		*/
		j := &Job{
			Token: macaddress,
		}

		if err := w.runBuildCommands(j, w.config.UnknownBuildCommands); err != nil {
			w.addLog(fmt.Sprintf("unknown-build commands for %s returned errors %v", macaddress, err), config.LogLevelDebug)
			return PixieConfig{}, err
		}
	}

	w.addLog("going to send _unknown_ details to unknown mac", config.LogLevelInfo)

	pixieConfig := PixieConfig{}

	var cmdline, imageURL, kernel, initrd string

	cmdline = b.Cmdline
	imageURL = b.ImageURL
	kernel = b.Kernel
	initrd = b.Initrd

	tpl, err := pongo2.FromString(cmdline)
	if err != nil {
		return pixieConfig, err
	}

	cmdline, err = tpl.Execute(pongo2.Context{"machine": b, "BaseURL": w.config.BaseURL, "Hostname": macaddress, "MAC": macaddress})

	if err != nil {
		return pixieConfig, err
	}

	imageURL = strings.TrimRight(imageURL, "/")

	pixieConfig.Kernel = imageURL + "/" + kernel
	pixieConfig.Initrd = []string{imageURL + "/" + initrd}
	pixieConfig.Cmdline = cmdline

	return pixieConfig, nil
}

/*
	Retrieves the PXE config based on the details of the job related to the specified MAC.
	This will/should be called when Waitron receives a request from something pixiecore, which is basically forwarding along
	the MAC from the DHCP request.
*/
func (w *Waitron) GetPxeConfig(macaddress string) (PixieConfig, error) {

	// Normalize the MAC
	r := strings.NewReplacer(":", "", "-", "", ".", "")
	normMacaddress := strings.ToLower(r.Replace(macaddress))

	// Look up the *Job by MAC
	w.jobs.RLock()
	j, found := w.jobs.jobByMAC[normMacaddress]
	w.jobs.RUnlock()

	if !found {
		if uBuild, ok := w.config.BuildTypes["_unknown_"]; ok {
			return w.getPxeConfigForUnknown(&uBuild, normMacaddress)
		} else {
			return PixieConfig{}, fmt.Errorf("job not found for  '%s'", normMacaddress)
		}
	}

	// Build the pxe config based on the compiled machine details.

	pixieConfig := PixieConfig{}

	/*
		It's entirely possible for multiple requests to come in for the same MAC, either from retries or because pixiecore/dhcp
		has been set up as "cluster" and you have "duplicate" pxe requests, but only one will ultimately be selected.
		We'll only want to trigger certain things when we're seeing a PXE for a MAC for the first time, such as when a set a network cards attempt PXE in order.

		Unique is a bit of a lie, though, since e.g. a machine looping endlessly through two network cards would keep toggling this var as it rotates through NICs/MACs
	*/
	uniquePxeRequest := false

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

	j.RUnlock()
	j.Lock()
	/*
		The deferred unlock was removed.  Even though the (read-locking) runBuildCommands call near the end
		is happening in a go-routine, having it happen while this function is holding a write-lock
		makes me too nervous.  It just feels too dead-lockish.
	*/

	if j.TriggerMacRaw != macaddress {
		uniquePxeRequest = true
		j.TriggerMacRaw = macaddress
		j.TriggerMacNormalized = normMacaddress
	}

	if err != nil {
		j.Status = "failed"
		j.StatusReason = "pxe config build failed"

		j.Unlock()

		return pixieConfig, err
	} else {
		j.Status = "installing"
		j.StatusReason = "pxe config sent"
	}

	j.Unlock()

	imageURL = strings.TrimRight(imageURL, "/")

	pixieConfig.Kernel = imageURL + "/" + kernel
	pixieConfig.Initrd = []string{imageURL + "/" + initrd}
	pixieConfig.Cmdline = cmdline

	/*
		It can be pretty valuable to be able to run commands when a PXE is received,
		but they shouldn't be allowed to block an install at this point.

		This would probably be good spot where go-routines could leak if a user were to create super-long running commands
		that don't, or practically don't, timeout.
	*/
	if uniquePxeRequest {
		go func() {
			if err := w.runBuildCommands(j, j.Machine.PxeEventCommands); err != nil {
				w.addLog(fmt.Sprintf("pxe-event commands for %s returned errors %v", macaddress, err), config.LogLevelError)
			}
		}()
	}

	w.addLog(fmt.Sprintf("PXE config for %s: %v", macaddress, pixieConfig), config.LogLevelDebug)

	return pixieConfig, nil
}

/*
	Clean up the references to the job, excluding from the job history
*/
func (w *Waitron) cleanUpJob(j *Job, status string) error {
	// Take the list of all macs found in that Jobs Machine->Network
	// Use host, token, and list of MACs to clean out the details from Jobs

	j.Lock()
	j.Status = status
	j.StatusReason = ""
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

/*
	Perform any final/post-build actions and then clean up the job refernces.
*/
func (w *Waitron) FinishBuild(hostname string, token string) error {

	j, _, err := w.getActiveJob(hostname, token)

	if err != nil {
		return err
	}

	if err := w.runBuildCommands(j, j.Machine.PostBuildCommands); err != nil {
		return err
	}

	// Run clean-up if all finish commands were successful (or non-fatal).
	return w.cleanUpJob(j, "completed")
}

/*
	Perform any final/cancel actions and then clean up the job references.
*/
func (w *Waitron) CancelBuild(hostname string, token string) error {

	j, _, err := w.getActiveJob(hostname, token)

	if err != nil {
		return err
	}

	if err := w.runBuildCommands(j, j.Machine.CancelBuildCommands); err != nil {
		return err
	}

	// Run clean-up if all cancel commands were successful (or non-fatal).
	return w.cleanUpJob(j, "terminated")
}

/*
	Remove all completed (non-active) jobs from the Job history.
	Eventually the in-memory job history will need pruning.  This handles that.
*/
func (w *Waitron) CleanHistory() error {
	// Loop through all items in JobsHistory and check existence in Waitron.jobs.JobByToken
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

	/*
		We're not invalidating the history cache here.
		Cleaning history will clean out complete jobs, which doesn't seem much different from
		adding in new jobs.  If the purpose of the history blob is to reduce load when history
		is queried frequently, this holds true even after cleaning since cleaning could still
		leave you with a large job history if many jobs are in flight when the cleaning happens.
	*/

	return nil
}

/*
	Returns a binary-blob representation of the current job history.
*/
func (w *Waitron) GetJobsHistoryBlob() ([]byte, error) {
	w.history.RLock()
	defer w.history.RUnlock()

	// This is the only place that touches historyBlobCache, so the history RLock's above end up working as RW locks for it.

	// Seems efficient...
	// https://github.com/golang/go/blob/0bd308ff27822378dc2db77d6dd0ad3c15ed2e08/src/runtime/map.go#L118
	if len(w.history.jobByToken) == 0 {
		w.addLog("no jobs, so returning empty job history", config.LogLevelInfo)

		/*
			If you do a lot of building, then prime the cache, then CleanHistory before ever calling GetJobsHistory again,
			you'll end up holding onto the cache until you build things and check history again.
			This really just seems like a symptom of the silly way the cache is built, so when someone does something smarter,
			this will just go away.
		*/
		if len(w.historyBlobCache) > 2 {
			w.historyBlobCache = []byte("[]")
		}

		return []byte("[]"), nil
	}

	// This is simple but seems kind of dumb, but every suggested solution went crazy with marshal and unmarshal,
	// which also seems dumb here but less simple. Did I miss something silly?
	if w.config.HistoryCacheSeconds > 0 && int(time.Now().Sub(w.historyBlobLastCached).Seconds()) < w.config.HistoryCacheSeconds {
		w.addLog("returning valid history cache", config.LogLevelInfo)
		return w.historyBlobCache, nil
	}

	w.addLog(fmt.Sprintf("rebuilding stale history blob cache of %d jobs", len(w.history.jobByToken)), config.LogLevelInfo)
	w.historyBlobCache = make([]byte, 1, 256*len(w.history.jobByToken))
	w.historyBlobCache[0] = '['

	// Each of the jobs in here needs to be RLock'ed as they are processed.
	// I need to loop through them.  Just Marshal'ing the history isn't acceptable. :(
	for _, job := range w.history.jobByToken {

		job.RLock()
		b, err := json.Marshal(job)
		job.RUnlock()

		if err != nil {
			return b, err
		}

		w.historyBlobCache = append(w.historyBlobCache, ',')
		w.historyBlobCache = append(w.historyBlobCache, b...) // So it's not _quite_ as bad as it looks? --> https://stackoverflow.com/questions/16248241/concatenate-two-slices-in-go#comment40751903_16248257
	}

	w.historyBlobCache = append(w.historyBlobCache, ']')
	w.historyBlobCache[1] = ' ' // Get rid of that prepended comma of the first item.

	w.historyBlobLastCached = time.Now()

	return w.historyBlobCache, nil
}

/*
	Returns a binary-blob representation of the specified job.
*/
func (w *Waitron) GetJobBlob(token string) ([]byte, error) {

	w.history.RLock()
	j, found := w.history.jobByToken[token]
	w.history.RUnlock()

	if !found {
		return []byte{}, fmt.Errorf("job '%s' not found", token)
	}

	j.RLock()
	b, err := json.Marshal(j)
	j.RUnlock()

	if err != nil {
		return []byte{}, err
	}

	return b, nil
}

/*
	Returns a fully rendered template for the ACTIVE job specified by the token.
*/
func (w *Waitron) RenderStageTemplate(token string, templateStage string) (string, error) {

	j, _, err := w.getActiveJob("", token)
	if err != nil {
		return "", err
	}

	// Render preseed as default
	templateName := j.Machine.Preseed

	if templateStage == "finish" {
		templateName = j.Machine.Finish
	}

	return w.renderTemplate(templateName, templateStage, j)
}

/*
	Performs the actual template rendering for a job and specified template.
*/
func (w *Waitron) renderTemplate(templateName string, templateStage string, j *Job) (string, error) {

	j.Lock()
	j.Status = templateStage
	j.StatusReason = "processing " + templateName
	j.Unlock()

	j.RLock()
	defer j.RUnlock()

	templateName = path.Join(w.config.TemplatePath, templateName)
	if _, err := os.Stat(templateName); err != nil {
		return "", errors.New("Template does not exist")
	}

	var tpl = pongo2.Must(pongo2.FromFile(templateName))
	result, err := tpl.Execute(pongo2.Context{"job": j, "machine": j.Machine, "config": w.config, "Token": j.Token})
	if err != nil {
		return "", err
	}
	return result, err
}
