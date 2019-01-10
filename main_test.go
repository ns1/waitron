package main

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPixieHandlerNotInBuildMode(t *testing.T) {
	request, _ := http.NewRequest("GET", "/boot/11:22:33:44:51", nil)
	response := httptest.NewRecorder()
	configuration, _ := loadConfig("config.yaml")
	ps := httprouter.Params{httprouter.Param{Key: "macaddr", Value: "1"}}
	
	state := loadState()
	
	pixieHandler(response, request, ps, configuration, state)

	expected := "Not in build mode"
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is %s, expected %s", response.Body, expected)
	}
	if response.Code != http.StatusNotFound {
		t.Errorf("Response code is %v, should be 404", response.Code)
	}
}

func TestPixieHandler(t *testing.T) {
	request, _ := http.NewRequest("GET", "/boot/de:ad:c0:de:ca:fe", nil)
	response := httptest.NewRecorder()
	configuration, _ := loadConfig("config.yaml")
	state := loadState()
	
	
	m, _ := machineDefinition("dns02.example.com", "machines", configuration)
	
	ps := httprouter.Params{httprouter.Param{Key: "macaddr", Value: "de:ad:c0:de:ca:fe"}}
	state.MachineByMAC["de:ad:c0:de:ca:fe"] = &m

	pixieHandler(response, request, ps, configuration, state)
	expected := "hostname=dns02.example.com"
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is %s, expected %s", response.Body, expected)
	}
	if response.Code != http.StatusOK {
		t.Errorf("Response code is %v, should be 200", response.Code)
	}
}

func TestPixieHandlerNoMachineDefinition(t *testing.T) {
	request, _ := http.NewRequest("GET", "/boot/de:ad:c0:de:ca:fe", nil)
	response := httptest.NewRecorder()
	configuration, _ := loadConfig("config.yaml")
	ps := httprouter.Params{httprouter.Param{Key: "macaddr", Value: "de:ad:c0:de:ca:fe"}}
	
	state := loadState()

	pixieHandler(response, request, ps, configuration, state)
	expected := "Not in build mode or definition does not exist"
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is %s, expected %s", response.Body, expected)
	}
	if response.Code != http.StatusNotFound {
		t.Errorf("Response code is %v, should be 404", response.Code)
	}
}

func TestMachinesHandlerList(t *testing.T) {
	request, _ := http.NewRequest("GET", "/list", nil)
	response := httptest.NewRecorder()
	configuration, _ := loadConfig("config.yaml")
	state := loadState()

	listMachinesHandler(response, request, nil, configuration, state)
	expected := `["dns02.example.com.yaml"]`
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is %s, expected %s", response.Body, expected)
	}
	if response.Code != http.StatusOK {
		t.Errorf("Response code is %v, should be 200", response.Code)
	}
}
