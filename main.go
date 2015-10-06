package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

type result struct {
	Token string `json:",omitempty"`
	Error string `json:",omitempty"`
	State string `json:",omitempty"`
}

// templateHandler renders either the finish or the preseed template
func templateHandler(response http.ResponseWriter, request *http.Request,
	config Config) {
	hostname := mux.Vars(request)["hostname"]

	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 400)
		return
	}

	if mux.Vars(request)["token"] != config.Token[hostname] {
		http.Error(response, "Invalid Token", 401)
		return
	}

	// Set token used in template
	m.Token = config.Token[hostname]

	// Render preseed as default
	var template string
	if mux.Vars(request)["template"] == "finish" {
		template = m.Finish
	} else {
		template = m.Preseed
	}

	renderedTemplate, err := m.renderTemplate(template, config)
	if err != nil {
		log.Println(err)
		http.Error(response, "Unable to render template", 400)
		return
	}

	fmt.Fprintf(response, renderedTemplate)
}

func buildHandler(response http.ResponseWriter, request *http.Request, config Config) {
	hostname := mux.Vars(request)["hostname"]

	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 500)
		return
	}

	// Generate a random token used to authenticate requests
	config.Token[hostname] = uuid.NewV4().String()
	log.Println(fmt.Sprintf("%s installation token: %s", hostname, config.Token[hostname]))

	// Add token to machine struct
	m.Token = config.Token[hostname]
	//template, err := m.renderTemplate("pxe.j2", config)

	//err = m.setBuildMode(config, template)
	//if err != nil {
	//	log.Println(err)
	//	http.Error(response, fmt.Sprintf("Failed to set build mode on %s", hostname), 500)
	//	return
	//}

	//Add to the MachineBuild table
	config.MachineBuild[fmt.Sprintf("%s", m.Network[0].MacAddress)] = hostname
	log.Println(config)

	config.MachineState[hostname] = "Installing"
	fmt.Fprintf(response, "OK")
}

/*
doneHandler sends a DELETE to the foreman-proxy telling it the installation
is complete and the pxe configuration can be removed
*/
func doneHandler(response http.ResponseWriter, request *http.Request, config Config) {
	hostname := mux.Vars(request)["hostname"]
	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 500)
		return
	}

	if mux.Vars(request)["token"] != config.Token[hostname] {
		http.Error(response, "Invalid Token", 401)
		return
	}

	//err = m.cancelBuildMode(config)
	//if err != nil {
	//	log.Println(err)
	//	http.Error(response, "Failed to cancel build mode", 500)
	//	return
	//}

	//Delete mac from the building map
	delete(config.MachineBuild, fmt.Sprintf("%s", m.Network[0].MacAddress))

	config.MachineState[hostname] = "Installed"
	fmt.Fprintf(response, "OK")
}

func hostStatus(response http.ResponseWriter, request *http.Request, config Config) {
	status := config.MachineState[mux.Vars(request)["hostname"]]
	if status == "" {
		http.Error(response, "Unknown state", 500)
		return
	}
	fmt.Fprintf(response, status)
}

func listMachinesHandler(response http.ResponseWriter, request *http.Request, config Config) {
	machines, err := config.listMachines()
	if err != nil {
		log.Println(err)
		http.Error(response, "Unable to list machines", 500)
		return
	}
	result, _ := json.Marshal(machines)
	response.Write(result)
}

func status(response http.ResponseWriter, request *http.Request, config Config) {
	result, _ := json.Marshal(&config.MachineState)
	response.Write(result)
}

func pixieHandler(response http.ResponseWriter, request *http.Request, config Config) {

	macaddr := mux.Vars(request)["macaddr"]

	hostname, ok := config.MachineBuild[macaddr]

	if ok {
		if mux.Vars(request)["token"] != config.Token[hostname] {
			http.Error(response, "Invalid Token", 401)
			return
		}
		result, _ := json.Marshal(pixieInit(config))
		log.Println(macaddr)
		response.Write(result)
	} else {
		http.Error(response, "Not in build mode", 404)
	}
}

func main() {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		log.Fatal("environment variables CONFIG_FILE must be set")
	}

	configuration, err := loadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	s := r.PathPrefix("/v1/").Subrouter()
	r.HandleFunc("/{hostname}/build",
		func(response http.ResponseWriter, request *http.Request) {
			buildHandler(response, request, configuration)
		})
	r.HandleFunc("/{hostname}/status",
		func(response http.ResponseWriter, request *http.Request) {
			hostStatus(response, request, configuration)
		})
	r.HandleFunc("/status",
		func(response http.ResponseWriter, request *http.Request) {
			status(response, request, configuration)
		})
	r.HandleFunc("/list",
		func(response http.ResponseWriter, request *http.Request) {
			listMachinesHandler(response, request, configuration)
		})
	r.HandleFunc("/{hostname}/done/{token}",
		func(response http.ResponseWriter, request *http.Request) {
			doneHandler(response, request, configuration)
		})
	r.HandleFunc("/{hostname}/{template}/{token}",
		func(response http.ResponseWriter, request *http.Request) {
			templateHandler(response, request, configuration)
		})
	s.HandleFunc("/boot/{macaddr}/{token}",
		func(response http.ResponseWriter, request *http.Request) {
			pixieHandler(response, request, configuration)
		})

	log.Println("Starting Server")
	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, r)))
}
