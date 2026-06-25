package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"google.golang.org/protobuf/encoding/protojson"
)

// handleFetchSources handles incoming requests to check endpoint statuses.
func (d *Datasource) handleFetchSources(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	response := make(map[string]any)
	for endpointID, endpointConfiguration := range d.multiplexer.Configuration.Endpoints {
		endpoint, err := d.multiplexer.GetEndpoint(endpointID)
		object := map[string]any{}
		object["name"] = endpointConfiguration.Name
		object["description"] = endpointConfiguration.Description
		if err != nil {
			object["online"] = false
			object["error"] = err.Error()
		} else {
			object["online"] = endpoint.GetClient().WebSocket.IsConnected()
		}
		response[endpointID] = object
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSearchParameters handles incoming requests to search for parameters.
func (d *Datasource) handleSearchParameters(w http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]
	query := req.URL.Query().Get("q")

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := endpoint.GetClient()
	reqIterator := client.SearchParameters(endpoint.Instance, query)
	results, err := reqIterator.Next()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	parameters := []string{}
	for _, result := range results {
		parameters = append(parameters, result.GetQualifiedName())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(parameters)

}

type CommandInfoResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (d *Datasource) handleSearchCommands(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]
	query := req.URL.Query().Get("q")

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := endpoint.GetClient()
	reqIterator := client.SearchCommandInfo(endpoint.Instance, query)
	results, err := reqIterator.Next()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	commands := []CommandInfoResult{}
	for _, result := range results {
		desc := result.GetShortDescription()
		if result.GetShortDescription() == "" {
			desc = result.GetLongDescription()
		}
		commandInfo := CommandInfoResult{
			Name:        result.GetQualifiedName(),
			Description: desc,
		}
		commands = append(commands, commandInfo)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commands)
}

// registerRoutes takes a *http.ServeMux and registers some HTTP handlers.
func (d *Datasource) registerRoutes(mux *mux.Router) {
	mux.HandleFunc("/fetch/endpoints", d.handleFetchSources)

	mux.HandleFunc("/fetch/health-details", d.handleGetLastHealthDetails)

	mux.HandleFunc("/endpoint/{endpointID}/parameters", d.handleSearchParameters)
	mux.HandleFunc("/endpoint/{endpointID}/time", d.handleEndpointTime)
	mux.HandleFunc("/endpoint/{endpointID}/commands", d.handleSearchCommands)
	mux.HandleFunc("/endpoint/{endpointID}/command/info", d.handleGetCommandInfo)
	mux.HandleFunc("/endpoint/{endpointID}/command/issue", d.handleExecuteCommand)
	mux.HandleFunc("/endpoint/{endpointID}/alarm/acknowledge", d.handleAcknowledgeAlarm)
	mux.HandleFunc("/endpoint/{endpointID}/alarm/clear", d.handleClearAlarm)
	mux.HandleFunc("/endpoint/{endpointID}/alarm/shelve", d.handleShelveAlarm)
	mux.HandleFunc("/endpoint/{endpointID}/alarm/unshelve", d.handleUnshelveAlarm)

	// Link management routes
	mux.HandleFunc("/endpoint/{endpointID}/links", d.handleListLinks)
	mux.HandleFunc("/endpoint/{endpointID}/links/{linkName}", d.handleGetLink)
	mux.HandleFunc("/endpoint/{endpointID}/links/{linkName}/enable", d.handleEnableLink)
	mux.HandleFunc("/endpoint/{endpointID}/links/{linkName}/disable", d.handleDisableLink)
	mux.HandleFunc("/endpoint/{endpointID}/links/{linkName}/reset", d.handleResetLinkCounters)
	mux.HandleFunc("/endpoint/{endpointID}/links/{linkName}/action/{actionID}", d.handleRunLinkAction)
}

func (d *Datasource) handleEndpointTime(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure we are subscribed to processor time updates.
	endpoint.RequestTime()

	currentTime := endpoint.CurrentTime
	currentTimeUpdatedAt := endpoint.CurrentTimeUpdatedAt
	maxTimeAge := 10 * time.Second

	if currentTime.IsZero() || currentTimeUpdatedAt.IsZero() || time.Since(currentTimeUpdatedAt) > maxTimeAge {
		http.Error(w, "processor time unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"currentTime": currentTime.UnixMilli(),
	})
}

// endpoint to get latest health details and whether they are available (non-nil)
func (d *Datasource) handleGetLastHealthDetails(w http.ResponseWriter, req *http.Request) {
	d.healthMutex.RLock()
	defer d.healthMutex.RUnlock()

	if d.lastHealthDetails != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(d.lastHealthDetails)
	} else {
		http.Error(w, "No health details available", http.StatusNotFound)
	}
}

type CommandIssueBody struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Comment   string         `json:"comment"`
}

const maxJSONBodyBytes int64 = 1 << 20

func decodeJSONBody(w http.ResponseWriter, req *http.Request, dst any) error {
	req.Body = http.MaxBytesReader(w, req.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON object")
	}

	return nil
}

func decodeOptionalJSONBody(w http.ResponseWriter, req *http.Request, dst any) error {
	req.Body = http.MaxBytesReader(w, req.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON object")
	}

	return nil
}

func (d *Datasource) handleGetCommandInfo(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]
	commandName := req.URL.Query().Get("name")
	if commandName == "" {
		http.Error(w, "missing required query parameter: name", http.StatusBadRequest)
		return
	}

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := endpoint.GetClient()
	commandInfo, err := client.GetCommandInfo(endpoint.Instance, commandName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	marshalled, err := protojson.Marshal(commandInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(marshalled)
}

func (d *Datasource) handleExecuteCommand(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]

	body := &CommandIssueBody{}
	err := decodeJSONBody(w, req, &body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := endpoint.GetClient()
	response, err := client.IssueCommandWithComment(endpoint.Instance, endpoint.Processor, body.Name, body.Arguments, body.Comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	marshalled, err := protojson.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	responseJSON := json.RawMessage(marshalled)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseJSON)
}

// AlarmActionBody represents the request body for alarm actions.
type AlarmActionBody struct {
	Name           string `json:"name"`
	SeqNum         uint32 `json:"seqNum"`
	Comment        string `json:"comment"`
	ShelveDuration uint64 `json:"shelveDuration,omitempty"` // Duration in milliseconds for shelving
}

func (d *Datasource) handleAcknowledgeAlarm(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]

	var body AlarmActionBody
	err := decodeJSONBody(w, req, &body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "missing required field: name", http.StatusBadRequest)
		return
	}

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := endpoint.GetClient()
	err = client.AcknowledgeAlarm(endpoint.Instance, endpoint.Processor, body.Name, body.SeqNum, body.Comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged"})
}

func (d *Datasource) handleClearAlarm(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]

	body := AlarmActionBody{}
	err := decodeJSONBody(w, req, &body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "missing required field: name", http.StatusBadRequest)
		return
	}

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := endpoint.GetClient()
	err = client.ClearAlarm(endpoint.Instance, endpoint.Processor, body.Name, body.SeqNum, body.Comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
}

func (d *Datasource) handleShelveAlarm(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]

	body := AlarmActionBody{}
	err := decodeJSONBody(w, req, &body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "missing required field: name", http.StatusBadRequest)
		return
	}

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := endpoint.GetClient()
	err = client.ShelveAlarm(endpoint.Instance, endpoint.Processor, body.Name, body.SeqNum, body.Comment, body.ShelveDuration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "shelved"})
}

func (d *Datasource) handleUnshelveAlarm(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]

	body := AlarmActionBody{}
	err := decodeJSONBody(w, req, &body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "missing required field: name", http.StatusBadRequest)
		return
	}

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := endpoint.GetClient()
	err = client.UnshelveAlarm(endpoint.Instance, endpoint.Processor, body.Name, body.SeqNum)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unshelved"})
}

// LinkInfoResult is a JSON-friendly representation of a link.
type LinkInfoResult struct {
	Instance       string         `json:"instance"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Disabled       bool           `json:"disabled"`
	Status         string         `json:"status"`
	DataInCount    int64          `json:"dataInCount"`
	DataOutCount   int64          `json:"dataOutCount"`
	DetailedStatus string         `json:"detailedStatus"`
	ParentName     string         `json:"parentName,omitempty"`
	Actions        []ActionResult `json:"actions,omitempty"`
	Extra          map[string]any `json:"extra,omitempty"`
}

// ActionResult is a JSON-friendly representation of a link action.
type ActionResult struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Style   string `json:"style"`
	Enabled bool   `json:"enabled"`
	Checked bool   `json:"checked"`
}

// handleListLinks handles incoming requests to list all links for an endpoint.
func (d *Datasource) handleListLinks(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := endpoint.GetClient()
	links, err := client.ListLinks(endpoint.Instance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results := make([]LinkInfoResult, 0, len(links))
	for _, link := range links {
		results = append(results, convertLinkInfo(link))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleGetLink handles incoming requests to get a specific link.
func (d *Datasource) handleGetLink(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]
	linkName := vars["linkName"]

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := endpoint.GetClient()
	link, err := client.GetLink(endpoint.Instance, linkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := convertLinkInfo(link)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleEnableLink handles incoming requests to enable a link.
func (d *Datasource) handleEnableLink(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]
	linkName := vars["linkName"]

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := endpoint.GetClient()
	link, err := client.EnableLink(endpoint.Instance, linkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := convertLinkInfo(link)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleDisableLink handles incoming requests to disable a link.
func (d *Datasource) handleDisableLink(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]
	linkName := vars["linkName"]

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := endpoint.GetClient()
	link, err := client.DisableLink(endpoint.Instance, linkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := convertLinkInfo(link)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleResetLinkCounters handles incoming requests to reset link counters.
func (d *Datasource) handleResetLinkCounters(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]
	linkName := vars["linkName"]

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := endpoint.GetClient()
	link, err := client.ResetLinkCounters(endpoint.Instance, linkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := convertLinkInfo(link)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// LinkActionBody represents the request body for running a link action.
type LinkActionBody struct {
	Message map[string]any `json:"message,omitempty"`
}

// handleRunLinkAction handles incoming requests to run a link action.
func (d *Datasource) handleRunLinkAction(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]
	linkName := vars["linkName"]
	actionID := vars["actionID"]

	endpoint, err := d.multiplexer.GetEndpoint(endpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse optional message body
	var body LinkActionBody
	if req.Body != nil {
		if err := decodeOptionalJSONBody(w, req, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	client := endpoint.GetClient()
	response, err := client.RunLinkAction(endpoint.Instance, linkName, actionID, body.Message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert response to JSON
	var result map[string]any
	if response != nil {
		result = response.AsMap()
	} else {
		result = make(map[string]any)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// convertLinkInfo converts a protobuf LinkInfo to a JSON-friendly LinkInfoResult.
func convertLinkInfo(link *links.LinkInfo) LinkInfoResult {
	result := LinkInfoResult{
		Instance:       link.GetInstance(),
		Name:           link.GetName(),
		Type:           link.GetType(),
		Disabled:       link.GetDisabled(),
		Status:         link.GetStatus(),
		DataInCount:    link.GetDataInCount(),
		DataOutCount:   link.GetDataOutCount(),
		DetailedStatus: link.GetDetailedStatus(),
		ParentName:     link.GetParentName(),
	}

	// Convert actions
	if link.GetActions() != nil {
		result.Actions = make([]ActionResult, 0, len(link.GetActions()))
		for _, action := range link.GetActions() {
			result.Actions = append(result.Actions, ActionResult{
				ID:      action.GetId(),
				Label:   action.GetLabel(),
				Style:   action.GetStyle(),
				Enabled: action.GetEnabled(),
				Checked: action.GetChecked(),
			})
		}
	}

	// Convert extra fields
	if link.GetExtra() != nil {
		result.Extra = link.GetExtra().AsMap()
	}

	return result
}
