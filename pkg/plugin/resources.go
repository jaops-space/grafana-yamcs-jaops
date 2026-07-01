package plugin

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
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
		endpoint, err := d.multiplexer.GetEndpoint(req.Context(), endpointID)
		object := map[string]any{}
		object["name"] = endpointConfiguration.Name
		object["description"] = endpointConfiguration.Description
		if err != nil {
			object["online"] = false
			object["error"] = "endpoint unavailable"
			backend.Logger.Error("Failed to retrieve endpoint", "endpointID", endpointID, "error", err)
		} else {
			client, clientErr := endpoint.GetClient()
			if clientErr != nil {
				object["online"] = false
				object["error"] = "endpoint unavailable"
				backend.Logger.Error("Failed to retrieve Yamcs client", "endpointID", endpointID, "error", clientErr)
			} else {
				object["online"] = client.WebSocket.IsConnected()
			}
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

	endpoint, err := d.multiplexer.GetEndpoint(req.Context(), endpointID)
	if err != nil {
		backend.Logger.Error("Failed to retrieve endpoint", "endpointID", endpointID, "error", err)
		http.Error(w, "endpoint unavailable", http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		backend.Logger.Error("Failed to retrieve Yamcs client", "endpointID", endpointID, "error", err)
		http.Error(w, "endpoint unavailable", http.StatusServiceUnavailable)
		return
	}
	reqIterator := client.SearchParameters(endpoint.Instance, query)
	results, err := reqIterator.Next(req.Context())
	if err != nil {
		backend.Logger.Error("Parameter search failed", "endpointID", endpointID, "query", query, "error", err)
		http.Error(w, "parameter search failed", http.StatusBadRequest)
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

	endpoint, err := d.multiplexer.GetEndpoint(req.Context(), endpointID)
	if err != nil {
		backend.Logger.Error("Failed to retrieve endpoint", "endpointID", endpointID, "error", err)
		http.Error(w, "endpoint unavailable", http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		backend.Logger.Error("Failed to retrieve Yamcs client", "endpointID", endpointID, "error", err)
		http.Error(w, "endpoint unavailable", http.StatusServiceUnavailable)
		return
	}
	reqIterator := client.SearchCommandInfo(endpoint.Instance, query)
	results, err := reqIterator.Next(req.Context())
	if err != nil {
		backend.Logger.Error("Command search failed", "endpointID", endpointID, "query", query, "error", err)
		http.Error(w, "command search failed", http.StatusBadRequest)
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
	mux.HandleFunc("/endpoint/{endpointID}/parameters", d.handleSearchParameters)
	mux.HandleFunc("/endpoint/{endpointID}/commands", d.handleSearchCommands)
	mux.HandleFunc("/endpoint/{endpointID}/command/issue", d.handleExecuteCommand)
}

type CommandIssueBody struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Comment   string         `json:"comment"`
}

func (d *Datasource) handleExecuteCommand(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(req)
	endpointID := vars["endpointID"]

	body := &CommandIssueBody{}
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		backend.Logger.Error("Invalid command request body", "endpointID", endpointID, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	endpoint, err := d.multiplexer.GetEndpoint(req.Context(), endpointID)
	if err != nil {
		backend.Logger.Error("Failed to retrieve endpoint", "endpointID", endpointID, "error", err)
		http.Error(w, "endpoint unavailable", http.StatusInternalServerError)
		return
	}
	client, err := endpoint.GetClient()
	if err != nil {
		backend.Logger.Error("Failed to retrieve Yamcs client", "endpointID", endpointID, "error", err)
		http.Error(w, "endpoint unavailable", http.StatusServiceUnavailable)
		return
	}
	response, err := client.IssueCommandWithComment(req.Context(), endpoint.Instance, endpoint.Processor, body.Name, body.Arguments, body.Comment)
	if err != nil {
		backend.Logger.Error("Command issue failed", "endpointID", endpointID, "command", body.Name, "error", err)
		http.Error(w, "command execution failed", http.StatusBadRequest)
		return
	}
	marshalled, err := protojson.Marshal(response)
	if err != nil {
		backend.Logger.Error("Failed to marshal command response", "endpointID", endpointID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	responseJSON := json.RawMessage(marshalled)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseJSON)
}
