package plugin

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
)

var GlobalMultiplexer *source.Multiplexer = source.NewMultiplexer(nil)

// Make sure App implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. Plugin should not implement all these interfaces - only those which are
// required for a particular task.
var (
	_ backend.CallResourceHandler   = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
	_ backend.StreamHandler         = (*Datasource)(nil)
)

type Datasource struct {
	backend.CallResourceHandler
	backend.StreamHandler
	instancemgmt.InstanceDisposer
	multiplexer *source.Multiplexer
	querier     *source.Querier
}
