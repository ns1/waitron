package waitron_test

import (
	"testing"

	"waitron/config"
	"waitron/inventoryplugins"
	"waitron/machine"

	"waitron/waitron"
)

// Test plugin #1
type TestPlugin struct {
}

func (t *TestPlugin) Init() error {
	return nil
}

func (t *TestPlugin) GetMachine(s string, m string) (*machine.Machine, error) {

	return &machine.Machine{Hostname: "test01.prod", ShortName: "test01"}, nil
}

func (t *TestPlugin) PutMachine(m *machine.Machine) error {
	return nil
}

func (t *TestPlugin) Deinit() error {
	return nil
}

// Test plugin #2
type TestPlugin2 struct {
}

func (t *TestPlugin2) Init() error {
	return nil
}

func (t *TestPlugin2) GetMachine(s string, m string) (*machine.Machine, error) {

	mm := &machine.Machine{
		Hostname:  "test01.prod",
		ShortName: "test02",
		Domain:    "domain02",
		Network: []machine.Interface{
			machine.Interface{
				MacAddress: "de:ad:be:ef",
			},
		},
	}

	return mm, nil
}

func (t *TestPlugin2) PutMachine(m *machine.Machine) error {
	return nil
}

func (t *TestPlugin2) Deinit() error {
	return nil
}

func TestWaitron(t *testing.T) {
	cf := config.Config{
		BuildType: config.BuildType{
			Cmdline:  "cmd",
			ImageURL: "image.com",
			Kernel:   "popcorn",
			Initrd:   "initrd",
		},
		MachineInventoryPlugins: []config.MachineInventoryPluginSettings{
			config.MachineInventoryPluginSettings{
				Name: "test1",
				Type: "test1",
			},
			config.MachineInventoryPluginSettings{
				Name: "test2",
				Type: "test2",
			},
		},
	}

	w := waitron.New(cf)

	/************** Stand up **************/
	if err := inventoryplugins.AddMachineInventoryPlugin("test1", func(s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, int) bool) inventoryplugins.MachineInventoryPlugin {
		return &TestPlugin{}
	}); err != nil {
		t.Errorf("Plugin factory failed to add test1 type: %v", err)
		return
	}

	if err := inventoryplugins.AddMachineInventoryPlugin("test2", func(s *config.MachineInventoryPluginSettings, c *config.Config, lf func(string, int) bool) inventoryplugins.MachineInventoryPlugin {
		return &TestPlugin2{}
	}); err != nil {
		t.Errorf("Plugin factory failed to add test1 type: %v", err)
		return
	}

	if err := w.Init(); err != nil {
		t.Errorf("Failed to init: %v", err)
		return
	}

	if err := w.Run(); err != nil {
		t.Errorf("Failed to run: %v", err)
		return
	}

	/******************************************************************/

	m, err := w.GetMergedMachine("", "")

	if err != nil {
		t.Errorf("Failed to get merge machine: %v", err)
		return
	}

	if m.Domain == "" {
		t.Errorf("Machine details were not merged")
		return
	}

	if m.ShortName != "test02" {
		t.Errorf("Plugin ordering was not reserved")
		return
	}

	/******************************************************************/

	_, err = w.Build("test01.prod", "invalid_build_type")

	if err == nil {
		t.Errorf("Allowed build with invalid build type")
		return
	}

	token, err := w.Build("test01.prod", "")

	if err != nil {
		t.Errorf("Failed to set build: %v", err)
		return
	}

	if token == "" {
		t.Errorf("invalid token returned: %s", token)
		return
	}

	if token2, _ := w.Build("test01.prod", ""); token2 != "" {
		t.Errorf("simultaneous builds for a single host were permitted: %s", token)
		return
	}

	/******************************************************************/

	status, err := w.GetJobStatus(token)
	if err != nil {
		t.Errorf("Failed to get job status: %v", err)
		return
	}

	if status != "pending" {
		t.Errorf("Incorrect status returned: %s", status)
		return
	}

	/******************************************************************/

	status, err = w.GetMachineStatus("test01.prod")
	if err != nil {
		t.Errorf("Failed to get machine status: %v", err)
		return
	}

	if status != "pending" {
		t.Errorf("Incorrect status returned")
		return
	}

	/******************************************************************/

	if _, err = w.GetPxeConfig("de:ad:c0:de:ca:fe"); err == nil {
		t.Errorf("Returned PXE config for unknown MAC")
		return
	}

	pCfg, err := w.GetPxeConfig("deadbeef")

	if err != nil {
		t.Errorf("Failed to return PXE config for known MAC v3: %v", err)
		return
	}

	pCfg, err = w.GetPxeConfig("de:ad:be:ef")

	if err != nil {
		t.Errorf("Failed to return PXE config for known MAC v2: %v", err)
		return
	}

	pCfg, err = w.GetPxeConfig("DE-AD-BE-EF")

	if err != nil {
		t.Errorf("Failed to return PXE config for known MAC v3: %v", err)
		return
	}

	if pCfg.Kernel == "" {
		t.Errorf("Empty PXE config returned.  Machine: %v", m)
		return
	}

	if pCfg.Kernel != "image.com/popcorn" {
		t.Errorf("Unexpected PXE config returned: %s", pCfg.Kernel)
		return
	}

	status, err = w.GetMachineStatus("test01.prod")
	if err != nil {
		t.Errorf("Failed to get machine status after pxe: %v", err)
		return
	}

	if status != "installing" {
		t.Errorf("Incorrect status returned: %s", status)
		return
	}

	/******************************************************************/

	if err = w.FinishBuild("test01.prod", token); err != nil {
		t.Errorf("Failed to finish build: %v", err)
		return
	}

	_, err = w.GetActiveJobStatus(token)
	if err == nil {
		t.Errorf("Job found active after finish")
		return
	}

	_, err = w.GetMachineStatus(token)
	if err == nil {
		t.Errorf("Able to trace job status to machine after finish")
		return
	}

	status, err = w.GetJobStatus(token)
	if err != nil {
		t.Errorf("Failed to get historical job status after finish: %v", err)
		return
	}

	if status != "completed" {
		t.Errorf("Incorrect status returned: %s", status)
		return
	}

	if err = w.CancelBuild("test01.prod", token); err == nil {
		t.Errorf("Permitted to cancel build after finish")
		return
	}

	/******************************************************************/
	blob, err := w.GetJobsHistoryBlob()
	if err != nil {
		t.Errorf("Failed to get jobs history blob: %v", err)
		return
	}

	if len(blob) == 0 {
		t.Errorf("History blob was unexpectedly empty: %v", blob)
		return
	}

	if string(blob) == "[]" {
		t.Errorf("History blob was unexpectedly has no jobs: %v", blob)
		return
	}

	if j, err := w.GetJobBlob(token); err != nil || len(j) == 0 {
		t.Errorf("Failed to get job blob for known token: err(%v) job(%v)", err, j)
		return
	}

	err = w.CleanHistory()
	if err != nil {
		t.Errorf("Failed to clean history: %v", err)
		return
	}

	_, err = w.GetJobStatus(token)
	if err == nil {
		t.Errorf("Able to get historical job status after cleaning")
		return
	}

	/******************************************************************/

	if err := w.Stop(); err != nil {
		t.Errorf("Failed to stop: %v", err)
		return
	}
}
