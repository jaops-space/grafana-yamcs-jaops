package plugin

type PluginQuery struct {
	Type          PluginQueryType `json:"type"`
	EndpointID    string          `json:"endpoint"`
	Parameter     string          `json:"parameter"`
	Command       string          `json:"command"`
	From          int             `json:"from"`
	To            int             `json:"to"`
	Fields        []string        `json:"fields"`
	Realtime      bool            `json:"realtime"`
	MaxPoints     int             `json:"points"`
	AggregatePath string          `json:"aggregatePath"`

    // user-chosen split time from Grafana
    SplitAt int `json:"splitAt,omitempty"`

    // YAMCS parameter filter configuration
    YamcsFilter *YamcsFilterConfig `json:"yamcsFilter,omitempty"`
}

// YamcsFilterConfig defines client-side YAMCS parameter filtering
type YamcsFilterConfig struct {
	Enabled   bool   `json:"enabled"`
	Parameter string `json:"parameter"` // Name of parameter to filter by
	Operator  string `json:"operator"`  // "equals" only for now
	Value     string `json:"value"`     // Expected value for comparison
}

type PluginQueryType string

const (
	Graph          PluginQueryType = "plot"
	SingleValue    PluginQueryType = "single"
	DiscreteValue  PluginQueryType = "discrete"
	Events         PluginQueryType = "events"
	Time           PluginQueryType = "time"
	Image          PluginQueryType = "image"
	Commanding     PluginQueryType = "commanding"
	CommandHistory PluginQueryType = "command-history"
	Demands        PluginQueryType = "demands"
	Subscriptions  PluginQueryType = "subscriptions"
)
