package main

// @APITitle Waitron
// @APIDescription Templates for server provisioning
// @License BSD
// @LicenseUrl http://opensource.org/licenses/BSD-2-Clause
import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type result struct {
	Token string `json:",omitempty"`
	Error string `json:",omitempty"`
	State string `json:",omitempty"`
}

// @Title templateHandler
// @Description Renders either the finish or the preseed template
// @Param hostname	path	string	true	"Hostname"
// @Param template	path	string	true	"The template to be rendered"
// @Param token		path	string	true	"Token"
// @Success 200	{object} string "Rendered template"
// @Failure 400	{object} string "Unable to find host definition for hostname"
// @Failure 400	{object} string "Unable to render template"
// @Failure 401	{object} string "Invalid token"
// @Router {hostname}/{template}/{token} [get]
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

// @Title buildHandler
// @Description Put the server in build mode
// @Param hostname	path	string	true	"Hostname"
// @Success 200	{object} string "OK"
// @Failure 500	{object} string "Unable to find host definition for hostname"
// @Failure 500	{object} string "Failed to set build mode on hostname"
// @Router /{hostname}/build [get]
func buildHandler(response http.ResponseWriter, request *http.Request, config Config) {
	hostname := mux.Vars(request)["hostname"]

	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 500)
		return
	}

	err = m.setBuildMode(config)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Failed to set build mode on %s", hostname), 500)
		return
	}

	fmt.Fprintf(response, "OK")
}

// @Title doneHandler
// @Description Removes the server from build mode
// @Param hostname	path	string	true	"Hostname"
// @Param token		path	string	true	"Token"
// @Success 200	{object} string "OK"
// @Failure 500	{object} string "Unable to find host definition for hostname"
// @Failure 500	{object} string "Failed to cancel build mode"
// @Failure 401	{object} string "Invalid token"
// @Router /{hostname}/done/{token} [get]
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

	err = m.cancelBuildMode(config)
	if err != nil {
		log.Println(err)
		http.Error(response, "Failed to cancel build mode", 500)
		return
	}

	fmt.Fprintf(response, "OK")
}

// @Title hostStatus
// @Description Build status of the server
// @Param hostname	path	string	true	"Hostname"
// @Success 200	{object} string "The status: (installing or installed)"
// @Failure 500	{object} string "Unknown state"
// @Router /{hostname}/status [get]
func hostStatus(response http.ResponseWriter, request *http.Request, config Config) {
	status := config.MachineState[mux.Vars(request)["hostname"]]
	if status == "" {
		http.Error(response, "Unknown state", 500)
		return
	}
	fmt.Fprintf(response, status)
}

// @Title listMachinesHandler
// @Description List machines handled by waitron
// @Success 200	{array} string "List of machines"
// @Failure 500	{object} string "Unable to list machines"
// @Router /list [get]
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

// @Title status
// @Description Dictionary with machines and its status
// @Success 200	{object} string "Dictionary with machines and its status"
// @Router /status [get]
func status(response http.ResponseWriter, request *http.Request, config Config) {
	result, _ := json.Marshal(&config.MachineState)
	response.Write(result)
}

// @Title pixieHandler
// @Description Dictionary with kernel, intrd(s) and commandline for pixiecore
// @Param macaddr	path	string	true	"MacAddress"
// @Success 200	{object} string "Dictionary with kernel, intrd(s) and commandline for pixiecore"
// @Failure 404	{object} string "Not in build mode"
// @Failure 500	{object} string "Unable to find host definition for hostname"
// @Router /v1/boot/{macaddr} [get]
func pixieHandler(response http.ResponseWriter, request *http.Request, config Config) {

	macaddr := mux.Vars(request)["macaddr"]
	hostname, found := config.MachineBuild[macaddr]

	if found == false {
		log.Println(found)
		http.Error(response, "Not in build mode", 404)
		return
	}

	m, err := machineDefinition(hostname, config.MachinePath)

	m.Token = config.Token[hostname]

	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 500)
		return
	}

	pxeconfig, _ := m.pixieInit(config)
	result, _ := json.Marshal(pxeconfig)
	response.Write(result)

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
	s.HandleFunc("/boot/{macaddr}",
		func(response http.ResponseWriter, request *http.Request) {
			pixieHandler(response, request, configuration)
		})

	log.Println("Starting Server")
	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, r)))
}
