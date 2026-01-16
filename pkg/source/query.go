package source

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// YamcsFilterConfig defines client-side YAMCS parameter filtering
// This is duplicated from plugin package to avoid circular import
type YamcsFilterConfig struct {
	Enabled   bool
	Parameter string
	Operator  string
	Value     string
}

// log is a helper to log with variable arguments
func log(msg string, fields map[string]interface{}) {
	if fields == nil {
		backend.Logger.Info(msg)
	} else {
		args := make([]interface{}, 0, len(fields)*2)
		for k, v := range fields {
			args = append(args, k, v)
		}
		backend.Logger.Info(msg, args...)
	}
}

// TelemetryPoint represents a single telemetry data point.
type TelemetryPoint struct {
	Time   time.Time
	Value  *float64
	Status string
}

// Querier orchestrates queries across Yamcs live data.
type Querier struct {
	endpoints    map[string]*config.YamcsEndpointConfiguration
}

// New creates a new Querier instance.
func New(endpoints map[string]*config.YamcsEndpointConfiguration) *Querier {
	return &Querier{
		endpoints: endpoints,
	}
}

// queryYamcsOnly queries data directly from Yamcs.
// yamcsFilter is an optional parameter filter configuration for server-side filtering.
func (q *Querier) queryYamcsOnly(yamcsClient *client.YamcsClient, instance client.Instance, processor client.Processor, from, to time.Time, telemetryIDs []string, yamcsFilter *YamcsFilterConfig) (map[string][]TelemetryPoint, error) {
	out := make(map[string][]TelemetryPoint, len(telemetryIDs))

	for _, id := range telemetryIDs {
		var samples []client.Sample
		var err error

		// Use filtered query if yamcsFilter is provided
		if yamcsFilter != nil && yamcsFilter.Enabled && yamcsFilter.Parameter != "" && yamcsFilter.Value != "" {
			samples, err = yamcsClient.GetParameterSamplesInProcessorByNamesWithFilter(
				instance.GetName(),
				processor.GetName(),
				id,
				from,
				to,
				yamcsFilter.Parameter,
				yamcsFilter.Value,
			)
		} else {
			samples, err = yamcsClient.GetParameterSamplesInProcessorByNames(
				instance.GetName(),
				processor.GetName(),
				id,
				from,
				to,
			)
		}

		if err != nil {
			return nil, err
		}
		out[id] = samplesToPoints(samples)
	}

	return out, nil
}

// samplesToPoints converts Yamcs samples to TelemetryPoint format.
func samplesToPoints(samples []client.Sample) []TelemetryPoint {
	series := make([]TelemetryPoint, 0, len(samples))
	for _, s := range samples {
		if s.GetN() == 0 {
			continue
		}
		v := s.GetAvg()
		ts := s.GetTime().AsTime()
		series = append(series, TelemetryPoint{
			Time:   ts,
			Value:  &v,
			Status: "",
		})
	}
	return series
}
