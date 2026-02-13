package plugin

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
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
	mux.HandleFunc("/endpoint/{endpointID}/parameters", d.handleSearchParameters)
	mux.HandleFunc("/endpoint/{endpointID}/commands", d.handleSearchCommands)
	mux.HandleFunc("/endpoint/{endpointID}/command/issue", d.handleExecuteCommand)
	mux.HandleFunc("/endpoint/{endpointID}/alarm/acknowledge", d.handleAcknowledgeAlarm)
	mux.HandleFunc("/endpoint/{endpointID}/alarm/clear", d.handleClearAlarm)
	mux.HandleFunc("/endpoint/{endpointID}/alarm/shelve", d.handleShelveAlarm)
	mux.HandleFunc("/endpoint/{endpointID}/alarm/unshelve", d.handleUnshelveAlarm)
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

	body := &AlarmActionBody{}
	err := json.NewDecoder(req.Body).Decode(&body)
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

	body := &AlarmActionBody{}
	err := json.NewDecoder(req.Body).Decode(&body)
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

	body := &AlarmActionBody{}
	err := json.NewDecoder(req.Body).Decode(&body)
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

	body := &AlarmActionBody{}
	err := json.NewDecoder(req.Body).Decode(&body)
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
	err = client.UnshelveAlarm(endpoint.Instance, endpoint.Processor, body.Name, body.SeqNum)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unshelved"})
}

