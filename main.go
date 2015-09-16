package main

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

type Config struct {
	TemplatePath string
	MachinePath  string
	Params       map[string]string
}

func loadConfig(configPath string) (Config, error) {
	var c Config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}
	yaml.Unmarshal(data, &c)
	return c, nil
}

func templateHandler(response http.ResponseWriter, request *http.Request,
	config Config) {
	hostname := mux.Vars(request)["hostname"]
	render := mux.Vars(request)["template"]

	m, err := machineDefinition(hostname, config.MachinePath)
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
func main() {

	configFile := os.Getenv("CONFIG_FILE")

	if configFile == "" {
		log.Fatal("environment variables CONFIG_FILE")
	}

	configuration, err := loadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/{hostname}/{template}",
		func(response http.ResponseWriter, request *http.Request) {
			templateHandler(response, request, configuration)
		}).Methods("GET")

	log.Println("Starting Server")
	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, r)))
}
