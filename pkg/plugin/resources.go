package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"google.golang.org/protobuf/encoding/protojson"
)

// handleFetchSources handles incoming requests to check endpoint statuses.
func (d *Datasource) handleFetchSources(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	response := make(map[string]any)
	for endpointID, endpoint := range d.multiplexer.Endpoints {
		object := map[string]any{}
		object["name"] = endpoint.Configuration.Name
		object["description"] = endpoint.Configuration.Description
		cli, err := endpoint.GetClient()
		if err != nil {
			object["online"] = false
			object["error"] = err.Error()
		} else {
			d.healthMutex.Lock()
			status := d.lastHealthDetails.Endpoints[endpointID]
			object["online"] = cli.WebSocket.IsConnected() && status.Status == "ok"
			if status.Status != "ok" {
				object["error"] = status.Message
			}
			d.healthMutex.Unlock()
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
		http.Error(w, exception.Wrap("endpoint not found", "SEARCH_PARAMETERS_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "SEARCH_PARAMETERS_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	reqIterator := client.SearchParameters(req.Context(), endpoint.GetInstanceName(), query)
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
		http.Error(w, exception.Wrap("endpoint not found", "SEARCH_COMMANDS_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "SEARCH_COMMANDS_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}

	reqIterator := client.SearchCommandInfo(req.Context(), endpoint.GetInstanceName(), query)
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

// endpoint to get latest health details and whether they are available (non-nil)
func (d *Datasource) handleGetLastHealthDetails(w http.ResponseWriter, req *http.Request) {
	d.healthMutex.RLock()
	defer d.healthMutex.RUnlock()

	if d.lastHealthDetails != nil {
		w.Header().Set("Content-Type", "application/json")
		jsonBytes, err := json.Marshal(d.lastHealthDetails)
		if err != nil {
			http.Error(w, "marshal error", http.StatusInternalServerError)
			return
		}
		w.Write(jsonBytes)
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
		http.Error(w, exception.Wrap("endpoint not found", "COMMAND_INFO_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}

	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "COMMAND_INFO_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}

	commandInfo, err := client.GetCommandInfo(req.Context(), endpoint.GetInstanceName(), commandName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	marshalled, err := protojson.Marshal(commandInfo)
	if err != nil {
		http.Error(w, exception.Wrap("could not marshal command info", "COMMAND_INFO_MARSHAL", err).Error(), http.StatusInternalServerError)
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
		http.Error(w, exception.Wrap("endpoint not found", "EXECUTE_COMMAND_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "EXECUTE_COMMAND_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	response, err := client.IssueCommandWithComment(req.Context(), endpoint.GetInstanceName(), endpoint.GetProcessorName(), body.Name, body.Arguments, body.Comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	marshalled, err := protojson.Marshal(response)
	if err != nil {
		http.Error(w, exception.Wrap("could not marshal command response", "EXECUTE_COMMAND_MARSHAL", err).Error(), http.StatusInternalServerError)
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
		http.Error(w, exception.Wrap("endpoint not found", "ACK_ALARM_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "ACK_ALARM_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	err = client.AcknowledgeAlarm(req.Context(), endpoint.GetInstanceName(), endpoint.GetProcessorName(), body.Name, body.SeqNum, body.Comment)
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
		http.Error(w, exception.Wrap("endpoint not found", "CLEAR_ALARM_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "CLEAR_ALARM_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	err = client.ClearAlarm(req.Context(), endpoint.GetInstanceName(), endpoint.GetProcessorName(), body.Name, body.SeqNum, body.Comment)
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
		http.Error(w, exception.Wrap("endpoint not found", "SHELVE_ALARM_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "SHELVE_ALARM_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	err = client.ShelveAlarm(req.Context(), endpoint.GetInstanceName(), endpoint.GetProcessorName(), body.Name, body.SeqNum, body.Comment, body.ShelveDuration)
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
		http.Error(w, exception.Wrap("endpoint not found", "UNSHELVE_ALARM_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "UNSHELVE_ALARM_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	err = client.UnshelveAlarm(req.Context(), endpoint.GetInstanceName(), endpoint.GetProcessorName(), body.Name, body.SeqNum)
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
		http.Error(w, exception.Wrap("endpoint not found", "LIST_LINKS_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}

	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "LIST_LINKS_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	links, err := client.ListLinks(req.Context(), endpoint.GetInstanceName())
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
		http.Error(w, exception.Wrap("endpoint not found", "GET_LINK_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}

	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "GET_LINK_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	link, err := client.GetLink(req.Context(), endpoint.GetInstanceName(), linkName)
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
		http.Error(w, exception.Wrap("endpoint not found", "ENABLE_LINK_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}

	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "ENABLE_LINK_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	link, err := client.EnableLink(req.Context(), endpoint.GetInstanceName(), linkName)
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
		http.Error(w, exception.Wrap("endpoint not found", "DISABLE_LINK_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}

	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "DISABLE_LINK_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	link, err := client.DisableLink(req.Context(), endpoint.GetInstanceName(), linkName)
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
		http.Error(w, exception.Wrap("endpoint not found", "RESET_LINK_COUNTERS_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
		return
	}

	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "RESET_LINK_COUNTERS_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	link, err := client.ResetLinkCounters(req.Context(), endpoint.GetInstanceName(), linkName)
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
		http.Error(w, exception.Wrap("endpoint not found", "RUN_LINK_ACTION_NO_ENDPOINT", err).Error(), http.StatusInternalServerError)
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

	client, err := endpoint.GetClient()
	if err != nil {
		http.Error(w, exception.Wrap("client not found", "RUN_LINK_ACTION_NO_CLIENT", err).Error(), http.StatusInternalServerError)
		return
	}
	response, err := client.RunLinkAction(req.Context(), endpoint.GetInstanceName(), linkName, actionID, body.Message)
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
