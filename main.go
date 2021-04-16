package main

// @Title Waitron
// @Version 2
// @Description Endpoints for server provisioning
// @License BSD
// @LicenseUrl http://opensource.org/licenses/BSD-2-Clause
import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"waitron/config"
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
// @Summary Return the waitron configuration details for a machine.  Note that "build type" is technically not required, depending on your config.
// @Param hostname  path    string    true    "Hostname"
// @Param type    	path    string    true    "Build Type"
// @Success 200    {object} string "Machine config in JSON format."
// @Failure 404    {object} string "Unable to find host definition for '<hostname>' '<build_type>' '<error>'"
// @Failure 500    {object} string "Bad machine data for '<hostname>' '<build_type>' '<error>'"
// @Router /definition/{hostname}/{type} [GET]
func definitionHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	hostname := ps.ByName("hostname")
	btype := ps.ByName("type")

	m, err := w.GetMergedMachine(hostname, "", btype, nil)
	if err != nil || m == nil {
		http.Error(response, fmt.Sprintf("Unable to find host definition for '%s' '%s'. %s", hostname, btype, err.Error()), 404)
		return
	}

	result, err := json.Marshal(m)

	if err != nil {
		http.Error(response, fmt.Sprintf("Bad machine data for '%s' '%s'. %s", hostname, btype, err.Error()), 500)
		return
	}

	fmt.Fprintf(response, string(result))
}

// @Title jobDefinitionHandler
// @Description Return details for the specified job token
// @Summary Return details for the specified job token
// @Param token    path    string    true    "Token"
// @Success 200    {object} string "Job details in JSON format."
// @Failure 404    {object} string "Job not found"
// @Router /job/{token} [GET]
func jobDefinitionHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	token := ps.ByName("token")

	jb, err := w.GetJobBlob(token)
	if err != nil {
		http.Error(response, fmt.Sprintf("Unable to find valid job for %s. %s", token, err.Error()), 404)
		return
	}

	response.Write(jb)
}

// @Title templateHandler
// @Description Render either the finish or the preseed template
// @Summary Render either the finish or the preseed template
// @Param hostname    path    string    true    "Hostname"
// @Param template    path    string    true    "The template to be rendered"
// @Param token        path    string    true    "Token"
// @Success 200    {object} string "Rendered template"
// @Failure 400    {object} string "Unable to render template"
// @Router /template/{template}/{hostname}/{token} [GET]
func templateHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	/* This eventually should to change to a PUT/POST because it causes changes. */

	renderedTemplate, err := w.RenderStageTemplate(ps.ByName("token"), ps.ByName("template"))
	if err != nil {
		http.Error(response, "Unable to render template", 400)
		return
	}

	fmt.Fprintf(response, renderedTemplate)
}

// @Title buildHandler
// @Description Put the server in build mode
// @Summary Put the server in build mode
// @Accept json
// @Produce json
// @Param hostname    path    string    true    "Hostname"
// @Param type        path    string    true    "Build Type"
// @Param {object}     body    string    true    "Machine definition if desired.  Can be used to override nearly all properties of a compiled machine.  See examples directory for machine definition."
// @Success 200    {object} string "{"State": "OK", "Token": <UUID of the build>}"
// @Failure 500    {object} string "Failed to set build mode on hostname"
// @Router /build/{hostname}/{type} [PUT]
func buildHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	hostname := ps.ByName("hostname")
	btype := ps.ByName("type")

	body := http.MaxBytesReader(response, request.Body, 1024*1024) // 1MB limit on posted machine definition.  Even that seems insane.
	machineDefinition, err := ioutil.ReadAll(body)

	if err != nil {
		http.Error(response, fmt.Sprintf("Failed to set build mode for %s - %s while reading request body: %s", hostname, btype, err.Error()), 500)
		return
	}

	token, err := w.Build(hostname, btype, machineDefinition)
	if err != nil {
		http.Error(response, fmt.Sprintf("Failed to set build mode for %s - %s: %s", hostname, btype, err.Error()), 500)
		return
	}

	result, _ := json.Marshal(&result{State: "OK", Token: token})

	fmt.Fprintf(response, string(result))
}

// @Title doneHandler
// @Description Remove the server from build mode
// @Summary Remove the server from build mode
// @Param hostname    path    string    true    "Hostname"
// @Param token        path    string    true    "Token"
// @Success 200    {object} string "{"State": "OK"}"
// @Failure 500    {object} string "Failed to finish build mode"
// @Router /done/{hostname}/{token} [GET]
func doneHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	/* This eventually should to change to a PUT/POST because it causes changes. */

	err := w.FinishBuild(ps.ByName("hostname"), ps.ByName("token"))

	if err != nil {
		http.Error(response, "Failed to finish build.", 500)
		return
	}

	result, _ := json.Marshal(&result{State: "OK"})

	fmt.Fprintf(response, string(result))
}

// @Title cancelHandler
// @Description Remove the server from build mode
// @Summary Remove the server from build mode
// @Accept json
// @Produce json
// @Param hostname    path    string    true    "Hostname"
// @Param token       path    string    true    "Token"
// @Param machine     body    string    true    "Machine definition if desired.  Can be used to override nearly all properties of a compiled machine.  See examples directory for machine definition."
// @Success 200    {object} string "{"State": "OK"}"
// @Failure 500    {object} string "Failed to cancel build mode"
// @Router /cancel/{hostname}/{token} [PUT]
func cancelHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	/* This eventually should to change to a PUT/POST because it causes changes. */

	err := w.CancelBuild(ps.ByName("hostname"), ps.ByName("token"))

	if err != nil {
		http.Error(response, "Failed to cancel build mode", 500)
		return
	}

	result, _ := json.Marshal(&result{State: "OK"})

	fmt.Fprintf(response, string(result))
}

// @Title hostStatus
// @Description Build status of the server
// @Summary Build status of the server
// @Param hostname    path    string    true    "Hostname"
// @Success 200    {object} string "The status: (installing or installed)"
// @Failure 404    {object} string "Failed to find active job for host"
// @Router /status/{hostname} [GET]
func hostStatus(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	hostname := ps.ByName("hostname")
	s, err := w.GetMachineStatus(hostname)

	if err != nil {
		http.Error(response, fmt.Sprintf("Failed to find active job for %s. Try search by job ID. %s", hostname, err.Error()), 404)
		return
	}
	fmt.Fprintf(response, s)
}

// @Title status
// @Description Dictionary with jobs and status
// @Summary Dictionary with jobs and status
// @Success 200    {object} string "Dictionary with jobs and status"
// @Success 500    {object} string "The error encountered"
// @Router /status [GET]
func status(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {
	result, err := w.GetJobsHistoryBlob()
	if err != nil {
		http.Error(response, err.Error(), 500)
		return
	}
	response.Write(result)
}

// @Title cleanHistory
// @Description Clear all completed jobs from the in-memory history of Waitron
// @Summary Clear all completed jobs from the in-memory history of Waitron
// @Success 200    {object} string "{"State": "OK"}"
// @Failure 500    {object} string "Failed to clean history"
// @Router /cleanhistory [PUT]
func cleanHistory(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {
	err := w.CleanHistory()
	if err != nil {
		http.Error(response, "Failed to clean history", 500)
		return
	}
	result, _ := json.Marshal(&result{State: "OK"})

	response.Write(result)
}

// @Title pixieHandler
// @Description Dictionary with kernel, intrd(s) and commandline for pixiecore
// @Summary Dictionary with kernel, intrd(s) and commandline for pixiecore
// @Param macaddr    path    string    true    "MacAddress"
// @Success 200    {object} string "Dictionary with kernel, intrd(s) and commandline for pixiecore"
// @Failure 500    {object} string "failed to get pxe config: <error>"
// @Router /v1/boot/{macaddr} [GET]
func pixieHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params, w *waitron.Waitron) {

	pxeconfig, err := w.GetPxeConfig(ps.ByName("macaddr"))

	if err != nil {
		http.Error(response, "failed to get pxe config: "+err.Error(), 500)
		return
	}

	result, _ := json.Marshal(pxeconfig)
	response.Write(result)
}

// @Title healthHandler
// @Description Check that Waitron is running
// @Summary Check that Waitron is running
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
	if err := w.Init(); err != nil {
		log.Fatal(err)
	}

	r := httprouter.New()
	r.PUT("/build/:hostname",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			buildHandler(response, request, ps, w)
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
	r.PUT("/cleanhistory",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			cleanHistory(response, request, ps, w)
		})
	r.GET("/definition/:hostname",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			definitionHandler(response, request, ps, w)
		})
	r.GET("/definition/:hostname/:type",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			definitionHandler(response, request, ps, w)
		})
	r.GET("/job/:token",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			jobDefinitionHandler(response, request, ps, w)
		})

	r.GET("/done/:hostname/:token",
		func(response http.ResponseWriter, request *http.Request, ps httprouter.Params) {
			doneHandler(response, request, ps, w)
		})
	r.PUT("/cancel/:hostname/:token",
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
		log.Fatal(fmt.Sprintf("waitron instance failed to run: %v", err))
	}

	log.Println("Starting Server on " + *address + ":" + *port)
	log.Fatal(http.ListenAndServe(*address+":"+*port, handlers.LoggingHandler(w.GetLogger(), r)))

	// This is practically a lie since nothing is properly catching signals AFAIK, but maybe in
	w.Stop()
}
