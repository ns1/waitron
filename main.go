package main

import (
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
)

// Config is our global configuration file
type Config struct {
	TemplatePath string
	MachinePath  string
	BaseURL      string
	Params       map[string]string
	Token        map[string]string
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
	token := mux.Vars(request)["token"]

	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 400)
		return
	}

	if token != config.Token[hostname] {
		http.Error(response, "Invalid Token", 401)
		return
	}

	// Set token used in template
	m.Token = config.Token[hostname]

	// Render preseed as default
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

	// Load template from config
	tpl, err := pongo2.FromString(config.Params["pxe_config"])
	if err != nil {
		log.Println(err)
		http.Error(response, "Unable to parse build template", 500)
		return
	}
	// Format template
	out, err := tpl.Execute(pongo2.Context{"machine": m, "config": config})
	if err != nil {
		log.Println(err)
		http.Error(response, "Unable to format build template", 500)
		return
	}
	// Send PXE config to foreman proxy
	foremanURL := fmt.Sprintf("%s/tftp/%s", config.Params["foremanproxy_address"], m.Network[0].MacAddress)
	_, err = http.PostForm(foremanURL, url.Values{"syslinux_config": {out}})
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Failed to reach foreman-proxy on %s", foremanURL), 500)
		return
	}
	fmt.Fprintf(response, "OK")
}

func doneHandler(response http.ResponseWriter, request *http.Request, config Config) {
	hostname := mux.Vars(request)["hostname"]
	token := mux.Vars(request)["token"]
	m, err := machineDefinition(hostname, config.MachinePath)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s", hostname), 500)
		return
	}

	if token != config.Token[hostname] {
		http.Error(response, "Invalid Token", 401)
		return
	}

	foremanURL := fmt.Sprintf("%s/tftp/%s", config.Params["foremanproxy_address"], m.Network[0].MacAddress)
	req, _ := http.NewRequest("DELETE", foremanURL, nil)
	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Failed to reach foreman-proxy on %s", foremanURL), 500)
		return
	}
	fmt.Fprintf(response, "OK")
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
	// Initialize map containing hostname[token]
	configuration.Token = make(map[string]string)

	r := mux.NewRouter()
	r.HandleFunc("/{hostname}/build",
		func(response http.ResponseWriter, request *http.Request) {
			buildHandler(response, request, configuration)
		}).Methods("GET")
	r.HandleFunc("/{hostname}/done/{token}",
		func(response http.ResponseWriter, request *http.Request) {
			doneHandler(response, request, configuration)
		}).Methods("GET")
	r.HandleFunc("/{hostname}/{template}/{token}",
		func(response http.ResponseWriter, request *http.Request) {
			templateHandler(response, request, configuration)
		}).Methods("GET")

	log.Println("Starting Server")
	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, r)))
}
