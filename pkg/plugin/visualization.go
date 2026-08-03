package plugin

import "github.com/grafana/grafana-plugin-sdk-go/data"

const (
	statVisualizationPluginID          = "stat"
	stateTimelineVisualizationPluginID = "state-timeline"

	commandingPanelPluginID      = "jaops-commanding-panel"
	commandHistoryPanelPluginID  = "jaops-commandhistory-panel"
	telemetricImagePanelPluginID = "jaops-telemetricimage-panel"
	alarmsPanelPluginID          = "jaops-alarms-panel"
	linksPanelPluginID           = "jaops-links-panel"
	timeSyncPanelPluginID        = "jaops-timesync-panel"
)

func setPreferredVisualization(frame *data.Frame, queryType PluginQueryType) {
	if frame == nil {
		return
	}
	if frame.Meta == nil {
		frame.Meta = &data.FrameMeta{}
	}
	frame.Meta.PreferredVisualization = ""
	frame.Meta.PreferredVisualizationPluginID = ""

	switch queryType {
	case Graph:
		frame.Meta.PreferredVisualization = data.VisTypeGraph
	case SingleValue:
		frame.Meta.PreferredVisualizationPluginID = statVisualizationPluginID
	case DiscreteValue:
		frame.Meta.PreferredVisualizationPluginID = stateTimelineVisualizationPluginID
	case Events:
		frame.Meta.PreferredVisualization = data.VisTypeLogs
	case Commanding:
		frame.Meta.PreferredVisualizationPluginID = commandingPanelPluginID
	case CommandHistory:
		frame.Meta.PreferredVisualizationPluginID = commandHistoryPanelPluginID
	case Image:
		frame.Meta.PreferredVisualizationPluginID = telemetricImagePanelPluginID
	case Alarms:
		frame.Meta.PreferredVisualizationPluginID = alarmsPanelPluginID
	case Links:
		frame.Meta.PreferredVisualizationPluginID = linksPanelPluginID
	case Time:
		frame.Meta.PreferredVisualizationPluginID = timeSyncPanelPluginID
	case Demands, Subscriptions:
		frame.Meta.PreferredVisualization = data.VisTypeTable
	}
}
