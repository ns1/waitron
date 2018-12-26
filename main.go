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
	"flag"
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
// @Router /template/{template}/{hostname}/{token} [GET]
func templateHandler(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params,
	config Config) {
	hostname := ps.ByName("hostname")

	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 400)
		return
	}

	if ps.ByName("token") != config.Tokens[hostname] {
		http.Error(response, "Invalid Token", 401)
		return
	}

	// Set token used in template
	m.Token = config.Tokens[hostname]

	// Render preseed as default
	var template string
	if ps.ByName("template") == "finish" {
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
// @Router build/{hostname} [PUT]
func buildHandler(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params, config Config) {
	hostname := ps.ByName("hostname")

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

	fmt.Fprintf(response, "OK")
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
	result, _ := json.Marshal(machines)
	response.Write(result)
}

// @Title status
// @Description Dictionary with machines and its status
// @Success 200	{object} string "Dictionary with machines and its status"
// @Router /status [GET]
func status(response http.ResponseWriter, request *http.Request,
	ps httprouter.Params, config Config) {
	result, _ := json.Marshal(&config.MachineState)
	response.Write(result)
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
	result, _ := json.Marshal(pxeconfig)
	response.Write(result)

}

func main() {
	
	config := flag.String("config", "", "Path to config file.")
	address := flag.String("address", "", "Address to listen for requests.")
	port := flag.String("port", "9090", "Port to listen for requests.")
	flag.Parse()
	
	configFile := *config

	if configFile == "" {
		if configFile = os.Getenv("CONFIG_FILE"); configFile == "" {
			log.Fatal("environment variables CONFIG_FILE must be set")
		}
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


	log.Println("Starting Server on " + *address + ":" + *port)
	log.Fatal(http.ListenAndServe(*address + ":" + *port, handlers.LoggingHandler(os.Stdout, r)))
}
