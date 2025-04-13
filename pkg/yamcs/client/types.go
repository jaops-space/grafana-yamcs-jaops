package client

import (
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/instances"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/mdb"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/yamcsManagement"
)

// Instance represents a Yamcs instance.
type Instance = *instances.YamcsInstance

// Parameter represents parameter information.
type Parameter = *mdb.ParameterInfo

// ParameterValue represents the value of a parameter.
type ParameterValue = *pvalue.ParameterValue

// Processor represents a processor within the Yamcs management system.
type Processor = *yamcsManagement.ProcessorInfo

// Sample represents a sample in a time series.
type Sample = *pvalue.TimeSeries_Sample

// ParameterTypeInfo holds information about the type of a parameter.
type ParameterTypeInfo = *mdb.ParameterTypeInfo

// CommandInfo holds information about a command.
type CommandInfo = *mdb.CommandInfo
