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
}

type PluginQueryType string

const (
	Graph         PluginQueryType = "plot"
	SingleValue   PluginQueryType = "single"
	DiscreteValue PluginQueryType = "discrete"
	Events        PluginQueryType = "events"
	Image         PluginQueryType = "image"
	Commanding    PluginQueryType = "commanding"
	Demands       PluginQueryType = "demands"
	Subscriptions PluginQueryType = "subscriptions"
)
