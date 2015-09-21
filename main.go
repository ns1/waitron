package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

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

	tpl := path.Join(config.TemplatePath, template)

	if _, err := os.Stat(tpl); err != nil {
		log.Println(err)
		http.Error(response, "Template file does not exist", 400)
		return
	}

	renderedTemplate, err := m.renderTemplate(tpl, config)
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
	template, err := m.renderTemplate(path.Join(config.TemplatePath, "pxe.j2"), config)

	err = m.setBuildMode(config, template)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Failed to set build mode on %s", hostname), 500)
		return
	}
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

	err = m.cancelBuildMode(config)
	if err != nil {
		log.Println(err)
		http.Error(response, "Failed to cancel build mode", 500)
		return
	}
	fmt.Fprintf(response, "OK")
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
	// Initialize map containing hostname[token]
	configuration.Token = make(map[string]string)

	r := mux.NewRouter()
	r.HandleFunc("/{hostname}/build",
		func(response http.ResponseWriter, request *http.Request) {
			buildHandler(response, request, configuration)
		})
	r.HandleFunc("/{hostname}/done/{token}",
		func(response http.ResponseWriter, request *http.Request) {
			doneHandler(response, request, configuration)
		})
	r.HandleFunc("/{hostname}/{template}/{token}",
		func(response http.ResponseWriter, request *http.Request) {
			templateHandler(response, request, configuration)
		})

	log.Println("Starting Server")
	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, r)))
}
