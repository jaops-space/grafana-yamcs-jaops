package plugin

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestSetPreferredVisualization(t *testing.T) {
	tests := []struct {
		name       string
		queryType  PluginQueryType
		wantID     string
		wantLegacy data.VisType
	}{
		{name: "graph uses graph enum", queryType: Graph, wantLegacy: data.VisTypeGraph},
		{name: "single uses stat", queryType: SingleValue, wantID: statVisualizationPluginID},
		{name: "discrete uses state timeline", queryType: DiscreteValue, wantID: stateTimelineVisualizationPluginID},
		{name: "events uses logs enum", queryType: Events, wantLegacy: data.VisTypeLogs},
		{name: "commanding uses custom panel", queryType: Commanding, wantID: commandingPanelPluginID},
		{name: "command history uses custom panel", queryType: CommandHistory, wantID: commandHistoryPanelPluginID},
		{name: "image uses custom panel", queryType: Image, wantID: telemetricImagePanelPluginID},
		{name: "alarms uses custom panel", queryType: Alarms, wantID: alarmsPanelPluginID},
		{name: "links uses custom panel", queryType: Links, wantID: linksPanelPluginID},
		{name: "time uses custom panel", queryType: Time, wantID: timeSyncPanelPluginID},
		{name: "demands uses table enum", queryType: Demands, wantLegacy: data.VisTypeTable},
		{name: "subscriptions uses table enum", queryType: Subscriptions, wantLegacy: data.VisTypeTable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := data.NewFrame("response")

			setPreferredVisualization(frame, tt.queryType)

			if frame.Meta == nil {
				t.Fatalf("expected frame metadata")
			}
			if frame.Meta.PreferredVisualizationPluginID != tt.wantID {
				t.Fatalf("expected visualization plugin ID %q, got %q", tt.wantID, frame.Meta.PreferredVisualizationPluginID)
			}
			if frame.Meta.PreferredVisualization != tt.wantLegacy {
				t.Fatalf("expected legacy preferred visualization %q, got %q", tt.wantLegacy, frame.Meta.PreferredVisualization)
			}
			if tt.wantID != "" && tt.wantLegacy != "" {
				t.Fatalf("test case cannot expect both plugin ID and legacy visualization")
			}
		})
	}
}

func TestSetPreferredVisualizationClearsTheUnusedMetadataField(t *testing.T) {
	tests := []struct {
		name      string
		queryType PluginQueryType
	}{
		{name: "graph enum clears plugin id", queryType: Graph},
		{name: "custom plugin id clears enum", queryType: Alarms},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &data.Frame{Meta: &data.FrameMeta{
				PreferredVisualization:         data.VisTypeTable,
				PreferredVisualizationPluginID: "previous-panel",
			}}

			setPreferredVisualization(frame, tt.queryType)

			if frame.Meta.PreferredVisualization != "" && frame.Meta.PreferredVisualizationPluginID != "" {
				t.Fatalf("expected only one preferred visualization metadata field, got enum %q and plugin ID %q",
					frame.Meta.PreferredVisualization,
					frame.Meta.PreferredVisualizationPluginID)
			}
		})
	}
}

func TestSetPreferredVisualizationPreservesCustomMetadata(t *testing.T) {
	custom := map[string]interface{}{"status": "ok"}
	frame := &data.Frame{Meta: &data.FrameMeta{Custom: custom}}

	setPreferredVisualization(frame, Alarms)

	gotCustom, ok := frame.Meta.Custom.(map[string]interface{})
	if !ok {
		t.Fatalf("expected custom metadata to be preserved")
	}
	if gotCustom["status"] != custom["status"] {
		t.Fatalf("expected custom status %q, got %q", custom["status"], gotCustom["status"])
	}
	if frame.Meta.PreferredVisualizationPluginID != alarmsPanelPluginID {
		t.Fatalf("expected alarms panel plugin ID, got %q", frame.Meta.PreferredVisualizationPluginID)
	}
}
