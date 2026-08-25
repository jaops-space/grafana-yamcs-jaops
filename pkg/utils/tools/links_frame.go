package tools

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
)

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

// ConvertLinkInfo converts a protobuf LinkInfo to a JSON-friendly LinkInfoResult.
func ConvertLinkInfo(link *links.LinkInfo) LinkInfoResult {
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

	if link.GetExtra() != nil {
		result.Extra = link.GetExtra().AsMap()
	}

	return result
}

// ConvertLinksToFrame converts Yamcs links into a columnar Grafana data frame.
func ConvertLinksToFrame(items []*links.LinkInfo) (*data.Frame, error) {
	instances := make([]string, 0, len(items))
	names := make([]string, 0, len(items))
	types := make([]string, 0, len(items))
	disabled := make([]bool, 0, len(items))
	statuses := make([]string, 0, len(items))
	dataInCounts := make([]int64, 0, len(items))
	dataOutCounts := make([]int64, 0, len(items))
	detailedStatuses := make([]string, 0, len(items))
	parentNames := make([]string, 0, len(items))
	actions := make([]json.RawMessage, 0, len(items))
	extras := make([]json.RawMessage, 0, len(items))

	for _, link := range items {
		result := ConvertLinkInfo(link)
		instances = append(instances, result.Instance)
		names = append(names, result.Name)
		types = append(types, result.Type)
		disabled = append(disabled, result.Disabled)
		statuses = append(statuses, result.Status)
		dataInCounts = append(dataInCounts, result.DataInCount)
		dataOutCounts = append(dataOutCounts, result.DataOutCount)
		detailedStatuses = append(detailedStatuses, result.DetailedStatus)
		parentNames = append(parentNames, result.ParentName)

		actionsJSON, err := json.Marshal(result.Actions)
		if err != nil {
			return nil, err
		}
		extraJSON, err := json.Marshal(result.Extra)
		if err != nil {
			return nil, err
		}
		actions = append(actions, json.RawMessage(actionsJSON))
		extras = append(extras, json.RawMessage(extraJSON))
	}

	return data.NewFrame(
		"links",
		data.NewField("instance", nil, instances),
		data.NewField("name", nil, names),
		data.NewField("type", nil, types),
		data.NewField("disabled", nil, disabled),
		data.NewField("status", nil, statuses),
		data.NewField("dataInCount", nil, dataInCounts),
		data.NewField("dataOutCount", nil, dataOutCounts),
		data.NewField("detailedStatus", nil, detailedStatuses),
		data.NewField("parentName", nil, parentNames),
		data.NewField("actions", nil, actions),
		data.NewField("extra", nil, extras),
	), nil
}
