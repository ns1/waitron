package main

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)

func templateHandler(response http.ResponseWriter, request *http.Request) {
	hostname := mux.Vars(request)["hostname"]
	render := mux.Vars(request)["template"]

	m, err := machineDefinition(hostname)
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

	renderedTemplate, err := m.renderTemplate(template)
	if err != nil {
		log.Println(err)
		http.Error(response, "Unable to render template", 400)
		return
	}

	fmt.Fprintf(response, renderedTemplate)
}
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/{hostname}/{template}",
		func(response http.ResponseWriter, request *http.Request) {
			templateHandler(response, request)
		}).Methods("GET")

	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, r)))
}
