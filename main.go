package main

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"path"
)

func templateHandler(response http.ResponseWriter, request *http.Request,
	templatePath string, machinePath string) {
	hostname := mux.Vars(request)["hostname"]
	render := mux.Vars(request)["template"]

	m, err := machineDefinition(hostname, machinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 400)
		return
	}

	var template string
	if render == "finish" {
		template = m.Finish
	} else {
		template = m.Preseed
	}

	renderedTemplate, err := m.renderTemplate(path.Join(templatePath, template))
	if err != nil {
		log.Println(err)
		http.Error(response, "Unable to render template", 400)
		return
	}

	fmt.Fprintf(response, renderedTemplate)
}
func main() {

	templatePath := os.Getenv("TEMPLATE_PATH")
	machinePath := os.Getenv("MACHINE_PATH")

	if templatePath == "" || machinePath == "" {
		log.Fatal("environment variables TEMPLATE_PATH and MACHINE_PATH must be set")
	}

	r := mux.NewRouter()
	r.HandleFunc("/{hostname}/{template}",
		func(response http.ResponseWriter, request *http.Request) {
			templateHandler(response, request, templatePath, machinePath)
		}).Methods("GET")

	log.Println("Starting Server")
	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, r)))
}
