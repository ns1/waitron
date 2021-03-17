package main

// @APITitle Waitron
// @APIDescription Templates for server provisioning
// @License BSD
// @LicenseUrl http://opensource.org/licenses/BSD-2-Clause
import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	config "waitron/config"
	"waitron/waitron"

	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
)

type result struct {
	Token string `json:",omitempty"`
	Error string `json:",omitempty"`
	State string `json:",omitempty"`
}

// @Title definitionHandler
// @Description Return the waitron configuration details for a machine
// @Param hostname    path    string    true    "Hostname"
// @Success 200    {object} string "Machine config in JSON format."
// @Failure 404    {object} string "Machine not found"
// @Router /definition/{hostname} [GET]
func definitionHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	hostname := ps.ByName("hostname")

	m, err := w.GetMachines([]string{hostname})
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Unable to find host definition for %s. %s", hostname, err.Error()), 404)
		return
	}

	result, _ := json.Marshal(m)

	fmt.Fprintf(response, string(result))
}

// @Title templateHandler
// @Description Render either the finish or the preseed template
// @Param hostname    path    string    true    "Hostname"
// @Param template    path    string    true    "The template to be rendered"
// @Param token        path    string    true    "Token"
// @Success 200    {object} string "Rendered template"
// @Failure 400    {object} string "Unable to render template"
// @Router /template/{template}/{hostname}/{token} [GET]
func templateHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	renderedTemplate, err := w.RenderStageTemplate(ps.ByName("token"), ps.ByName("template"))
	if err != nil {
		log.Println(err)
		http.Error(response, "Unable to render template", 400)
		return
	}

	fmt.Fprintf(response, renderedTemplate)
}

// @Title buildHandler
// @Description Put the server in build mode
// @Param hostname    path    string    true    "Hostname"
// @Param type        path    string    true    "Type"
// @Success 200    {object} string "{"State": "OK", "Token": <UUID of the build>}"
// @Failure 500    {object} string "Failed to set build mode on hostname"
// @Router build/{hostname}/{type} [PUT]
func buildHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	hostname := ps.ByName("hostname")
	btype := ps.ByName("type")

	if btype == "" {
		btype = "default"
	}

	token, err := w.Build(hostname, btype)
	if err != nil {
		log.Println(err)
		http.Error(response, fmt.Sprintf("Failed to set build mode for %s - %s: %s", hostname, btype, err.Error()), 500)
		return
	}

	result, _ := json.Marshal(&result{State: "OK", Token: token})

	fmt.Fprintf(response, string(result))
}

// @Title doneHandler
// @Description Remove the server from build mode
// @Param hostname    path    string    true    "Hostname"
// @Param token        path    string    true    "Token"
// @Success 200    {object} string "{"State": "OK"}"
// @Failure 500    {object} string "Failed to finish build mode"
// @Failure 400    {object} string "Not in build mode or definition does not exist"
// @Failure 401    {object} string "Invalid token"
// @Router /done/{hostname}/{token} [GET]
func doneHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	err := w.FinishBuild(ps.ByName("hostname"), ps.ByName("token"))

	if err != nil {
		log.Println(err)
		http.Error(response, "Failed to finish build.", 500)
		return
	}

	result, _ := json.Marshal(&result{State: "OK"})

	fmt.Fprintf(response, string(result))
}

// @Title cancelHandler
// @Description Remove the server from build mode
// @Param hostname    path    string    true    "Hostname"
// @Param token        path    string    true    "Token"
// @Success 200    {object} string "{"State": "OK"}"
// @Failure 500    {object} string "Failed to cancel build mode"
// @Failure 400    {object} string "Not in build mode or definition does not exist"
// @Failure 401    {object} string "Invalid token"
// @Router /cancel/{hostname}/{token} [GET]
func cancelHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	err := w.FinishBuild(ps.ByName("hostname"), ps.ByName("token"))

	if err != nil {
		log.Println(err)
		http.Error(response, "Failed to cancel build mode", 500)
		return
	}

	result, _ := json.Marshal(&result{State: "OK"})

	fmt.Fprintf(response, string(result))
}

// @Title hostStatus
// @Description Build status of the server
// @Param hostname    path    string    true    "Hostname"
// @Success 200    {object} string "The status: (installing or installed)"
// @Failure 500    {object} string "Unknown state"
// @Router /status/{hostname} [GET]
func hostStatus(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {
	s, err := w.GetMachineStatus(ps.ByName("hostname"))

	if err != nil {
		http.Error(response, s, 500)
		return
	}
	fmt.Fprintf(response, s)
}

// @Title listMachinesHandler
// @Description List machines handled by waitron
// @Success 200    {array} string "List of machines"
// @Failure 500    {object} string "Unable to list machines"
// @Router /list [GET]
func listMachinesHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {
	machines, err := w.GetMachines([]string{})

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
// @Success 200    {object} string "Dictionary with machines and its status"
// @Router /status [GET]
func status(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {
	result, _ := w.GetJobsHistoryBlob()
	response.Write(result)
}

// @Title pixieHandler
// @Description Dictionary with kernel, intrd(s) and commandline for pixiecore
// @Param macaddr    path    string    true    "MacAddress"
// @Success 200    {object} string "Dictionary with kernel, intrd(s) and commandline for pixiecore"
// @Failure 404    {object} string "Not in build mode"
// @Failure 500    {object} string "Unable to find host definition for hostname"
// @Router /v1/boot/{macaddr} [GET]
func pixieHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	r := strings.NewReplacer(":", "", "-", "", ".", "")

	pxeconfig, err := w.GetPxeConfig(strings.ToLower(r.Replace(ps.ByName("macaddr"))))

	if err != nil {
		log.Println(err)
		http.Error(response, "failed to get pxe config", 500)
		return
	}

	result, _ := json.Marshal(pxeconfig)
	response.Write(result)
}

// @Title healthHandler
// @Description Check that Waitron is running
// @Success 200    {object} string "{"State": "OK"}"
// @Router /health [GET]
func healthHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	result, _ := json.Marshal(&result{State: "OK"})

	fmt.Fprintf(response, string(result))
}

func main() {

	configPath := flag.String("config", "", "Path to config file.")
	address := flag.String("address", "", "Address to listen for requests.")
	port := flag.String("port", "9090", "Port to listen for requests.")
	flag.Parse()

	configFile := *configPath

	if configFile == "" {
		if configFile = os.Getenv("CONFIG_FILE"); configFile == "" {
			log.Fatal("environment variables CONFIG_FILE must be set or use --config")
		}
	}

	configuration, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	w := waitron.New(configuration)

	r := httprouter.New()
	r.GET("/list",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			listMachinesHandler(response, request, ps, w)
		})
	r.PUT("/build/:hostname/:type",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			buildHandler(response, request, ps, w)
		})
	r.GET("/status/:hostname",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			hostStatus(response, request, ps, w)
		})
	r.GET("/status",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			status(response, request, ps, w)
		})
	r.GET("/definition/:hostname",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			definitionHandler(response, request, ps, w)
		})
	r.GET("/done/:hostname/:token",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			doneHandler(response, request, ps, w)
		})
	r.GET("/cancel/:hostname/:token",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			cancelHandler(response, request, ps, w)
		})
	r.GET("/template/:template/:hostname/:token",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			templateHandler(response, request, ps, w)
		})
	r.GET("/v1/boot/:macaddr",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			pixieHandler(response, request, ps, w)
		})
	r.GET("/health",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			healthHandler(response, request, ps, w)
		})

	if configuration.StaticFilesPath != "" {
		fs := http.FileServer(http.Dir(configuration.StaticFilesPath))
		r.Handler("GET", "/files/:filename", http.StripPrefix("/files/", fs))
		log.Println("Serving static files from " + configuration.StaticFilesPath)
	}

	if err := w.Run(); err != nil {
		log.Fatal("waitron instance failed to run: %w", err)
	}

	log.Println("Starting Server on " + *address + ":" + *port)
	log.Fatal(http.ListenAndServe(*address+":"+*port, handlers.LoggingHandler(os.Stdout, r)))

	w.Stop()
}
