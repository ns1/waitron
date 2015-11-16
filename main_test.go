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
	pixieHandler(response, request, ps, configuration)

	expected := "Not in build mode"
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is %s, expected %s", response.Body, expected)
	}
	if response.Code != http.StatusNotFound {
		t.Errorf("Response code is %v, should be 404", response.Code)
	}
}

func TestPixieHandler(t *testing.T) {
	request, _ := http.NewRequest("GET", "/boot/00:11:44:24:50", nil)
	response := httptest.NewRecorder()
	configuration, _ := loadConfig("config.yaml")
	ps := httprouter.Params{httprouter.Param{Key: "macaddr", Value: "00:11:44:24:50"}}
	configuration.MachineBuild["00:11:44:24:50"] = "my-service.example.com"

	pixieHandler(response, request, ps, configuration)
	expected := "hostname=my-service.example.com"
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is %s, expected %s", response.Body, expected)
	}
	if response.Code != http.StatusOK {
		t.Errorf("Response code is %v, should be 200", response.Code)
	}
}

func TestPixieHandlerNoMachineDefinition(t *testing.T) {
	request, _ := http.NewRequest("GET", "/boot/00:11:44:24:50", nil)
	response := httptest.NewRecorder()
	configuration, _ := loadConfig("config.yaml")
	ps := httprouter.Params{httprouter.Param{Key: "macaddr", Value: "00:11:44:24:50"}}
	configuration.MachineBuild["00:11:44:24:50"] = "this.is.incorrect"

	pixieHandler(response, request, ps, configuration)
	expected := "Unable to find host definition"
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is %s, expected %s", response.Body, expected)
	}
	if response.Code != http.StatusInternalServerError {
		t.Errorf("Response code is %v, should be 200", response.Code)
	}
}

func TestMachinesHandlerList(t *testing.T) {
	request, _ := http.NewRequest("GET", "/list", nil)
	response := httptest.NewRecorder()
	configuration, _ := loadConfig("config.yaml")

	listMachinesHandler(response, request, nil, configuration)
	expected := `["my-service.example.com.yaml"]`
	if !strings.Contains(response.Body.String(), expected) {
		t.Errorf("Reponse body is %s, expected %s", response.Body, expected)
	}
	if response.Code != http.StatusOK {
		t.Errorf("Response code is %v, should be 200", response.Code)
	}
}
