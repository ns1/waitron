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
	"github.com/julienschmidt/httprouter"
)

type result struct {
	Token string `json:",omitempty"`
	Error string `json:",omitempty"`
	State string `json:",omitempty"`
}

type HttpResponse struct {
	Message    string
	StatusCode int
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
// @Router /template/{template}/{hostname}/{token} [GET]
func templateHandler(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params,
	config Config) {

	hostname := ps.ByName("hostname")

	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), http.StatusNotFound)
		return
	}

	if ps.ByName("token") != config.Tokens[hostname] {
		http.Error(response, "Invalid Token", http.StatusUnauthorized)
		return
	}

	// Set token used in template
	m.Token = config.Tokens[hostname]

	// Render preseed as default
	var template string

	switch ps.ByName("template") {
	case "preseed":
		template = m.Preseed
	case "finish":
		template = m.Finish
	case "cloud-init":
		renderedTemplate, err := m.renderCloudInit(hostname, config)
		if err != nil {
			log.Println(err)
			http.Error(response, "Unable to render template", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(response, renderedTemplate)
		return
	}

	renderedTemplate, err := m.renderTemplate(template, config)
	if err != nil {
		log.Println(err)
		http.Error(response, "Unable to render template", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(response, renderedTemplate)
}

// @Title hostConfigHandler
// @Description Renders the host configuration
// @Param hostname  path  string  true  "Hostname"
// @Success 200 {object} string "Rendered template"
// @Failure 400 {object} string "Unable to find host definition for hostname"
// @Router /config/{hostname} [GET]
func hostConfigHandler(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params,
	config Config) {

	hostname := ps.ByName("hostname")

	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, "", http.StatusNotFound)
		return
	}

	response.Header().Set("content-type", "application/json")
	result, _ := json.Marshal(m)
	response.Write(result)
}

// @Title buildHandler
// @Description Put the server in build mode
// @Param hostname	path	string	true	"Hostname"
// @Success 200	{object} string "OK"
// @Failure 500	{object} string "Unable to find host definition for hostname"
// @Failure 500	{object} string "Failed to set build mode on hostname"
// @Router build/{hostname} [PUT]
func buildHandler(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params, config Config) {
	hostname := ps.ByName("hostname")

	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), http.StatusNotFound)
		return
	}

	err = m.setBuildMode(config)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Failed to set build mode on %s", hostname), http.StatusInternalServerError)
		return
	}
	response.Header().Set("content-type", "application/json")
	msg := HttpResponse{"ok", http.StatusOK}
	js, err := json.Marshal(msg)
	response.Write(js)
}

// @Title doneHandler
// @Description Removes the server from build mode
// @Param hostname	path	string	true	"Hostname"
// @Param token		path	string	true	"Token"
// @Success 200	{object} string "OK"
// @Failure 500	{object} string "Unable to find host definition for hostname"
// @Failure 500	{object} string "Failed to cancel build mode"
// @Failure 401	{object} string "Invalid token"
// @Router /done/{hostname}/{token} [GET]
func doneHandler(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params, config Config) {
	hostname := ps.ByName("hostname")
	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 500)
		return
	}

	if ps.ByName("token") != config.Tokens[hostname] {
		http.Error(response, "Invalid Token", 401)
		return
	}

	err = m.cancelBuildMode(config)
	if err != nil {
		log.Println(err)
		http.Error(response, "Failed to cancel build mode", 500)
		return
	}
	response.Header().Set("content-type", "application/json")
	msg := HttpResponse{"ok", http.StatusOK}
	js, err := json.Marshal(msg)
	response.Write(js)
}

// @Title hostStatus
// @Description Build status of the server
// @Param hostname	path	string	true	"Hostname"
// @Success 200	{object} string "The status: (installing or installed)"
// @Failure 500	{object} string "Unknown state"
// @Router /status/{hostname} [GET]
func hostStatus(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params, config Config) {
	status := config.MachineState[ps.ByName("hostname")]
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
// @Router /list [GET]
func listMachinesHandler(response http.ResponseWriter, request *http.Request,
	_ httprouter.Params, config Config) {
	machines, err := config.listMachines()
	if err != nil {
		log.Println(err)
		http.Error(response, "Unable to list machines", 500)
		return
	}
	js, _ := json.Marshal(machines)
	response.Header().Set("content-type", "application/json")
	response.Write(js)
}

// @Title status
// @Description Dictionary with machines and its status
// @Success 200	{object} string "Dictionary with machines and its status"
// @Router /status [GET]
func status(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params, config Config) {
	js, _ := json.Marshal(&config.MachineState)
	response.Header().Set("content-type", "application/json")
	response.Write(js)
}

// @Title pixieHandler
// @Description Dictionary with kernel, intrd(s) and commandline for pixiecore
// @Param macaddr	path	string	true	"MacAddress"
// @Success 200	{object} string "Dictionary with kernel, intrd(s) and commandline for pixiecore"
// @Failure 404	{object} string "Not in build mode"
// @Failure 500	{object} string "Unable to find host definition for hostname"
// @Router /v1/boot/{macaddr} [GET]
func pixieHandler(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params, config Config) {

	macaddr := ps.ByName("macaddr")
	hostname, found := config.MachineBuild[macaddr]

	if found == false {
		log.Println(found)
		http.Error(response, "Not in build mode", 404)
		return
	}

	m, err := machineDefinition(hostname, config.MachinePath)

	m.Token = config.Tokens[hostname]

	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 500)
		return
	}

	pxeconfig, _ := m.pixieInit(config)
	js, _ := json.Marshal(pxeconfig)
	response.Header().Set("content-type", "application/json")
	response.Write(js)

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

	r := httprouter.New()
	r.GET("/list",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			listMachinesHandler(response, request, ps, configuration)
		})
	r.PUT("/build/:hostname",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			buildHandler(response, request, ps, configuration)
		})
	r.GET("/status/:hostname",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			hostStatus(response, request, ps, configuration)
		})
	r.GET("/config/:hostname",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			hostConfigHandler(response, request, ps, configuration)
		})
	r.GET("/status",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			status(response, request, ps, configuration)
		})
	r.GET("/done/:hostname/:token",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			doneHandler(response, request, ps, configuration)
		})
	r.GET("/template/:template/:hostname/:token",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			templateHandler(response, request, ps, configuration)
		})
	r.GET("/v1/boot/:macaddr",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			pixieHandler(response, request, ps, configuration)
		})

	log.Println("Starting Server")
	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, r)))
}
