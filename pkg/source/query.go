package source

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/database"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
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

// Querier orchestrates queries across the archive database and Yamcs live data.
type Querier struct {
	endpoints    map[string]*config.YamcsEndpointConfiguration
	yamcsConnMgr *ConnectionManager
	archiveDbMgr *DbConnectionManager
}

// New creates a new Querier instance.
func New(endpoints map[string]*config.YamcsEndpointConfiguration, yamcsConnMgr *ConnectionManager, dbMgr *DbConnectionManager) *Querier {
	return &Querier{
		endpoints:    endpoints,
		yamcsConnMgr: yamcsConnMgr,
		archiveDbMgr: dbMgr,
	}
}

// QueryTelemetryRange queries telemetry data from the archive database for [-∞, T0)
// and from Yamcs for [T0, now], with automatic boundary handling.
//
// If no archive database is configured, queries only from Yamcs.
// If splitAt is not provided, defaults to (now - liveWindow).
// dbSqlFilter is an optional SQL WHERE clause fragment for database queries.
// yamcsFilter is an optional parameter filter configuration for YAMCS queries.
func (q *Querier) QueryTelemetryRange(ctx context.Context, endpointID string, from, to time.Time, telemetryIDs []string, splitAt *time.Time, dbSqlFilter string, yamcsFilter *YamcsFilterConfig) (map[string][]TelemetryPoint, error) {
	// Get the database client for this endpoint (if configured)
	archiveClient := q.archiveDbMgr.GetConnection(endpointID)
	archiveConfigured := archiveClient != nil

	// Log entry
	log("QueryTelemetryRange called", map[string]interface{}{
		"endpoint":          endpointID,
		"from":              from,
		"to":                to,
		"paramCount":        len(telemetryIDs),
		"archiveConfigured": archiveConfigured,
		"splitAtProvided":   splitAt != nil,
	})

	endpointConfig := q.endpoints[endpointID]
	if endpointConfig == nil {
		log("Failed to get endpoint config", map[string]interface{}{"error": "endpoint not found"})
		return nil, exception.New("Endpoint configuration not found", "ENDPOINT_CONFIG_NOT_FOUND")
	}

	yamcsClient, err := q.yamcsConnMgr.GetClient(endpointConfig.Host)
	if err != nil {
		log("Failed to get Yamcs client", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	instance, err := yamcsClient.GetInstanceByName(endpointConfig.Instance)
	if err != nil {
		log("Failed to get instance", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	processor, err := yamcsClient.GetProcessor(instance, endpointConfig.Processor)
	if err != nil {
		processor = yamcsClient.GetInstanceDefaultProcessor(instance)
		if processor == nil {
			log("Failed to get processor", map[string]interface{}{"error": err.Error()})
			return nil, err
		}
	}

	out := make(map[string][]TelemetryPoint, len(telemetryIDs))

	// If no archive database is configured for this endpoint, query Yamcs only
	if !archiveConfigured {
		log("No archive database for endpoint, querying Yamcs only", nil)
		return q.queryYamcsOnly(yamcsClient, instance, processor, from, to, telemetryIDs, yamcsFilter)
	}

	// Decide T0
	now := time.Now()
	liveWin := time.Duration(endpointConfig.Database.LiveWindowSec) * time.Second

	var T0 time.Time
	if splitAt != nil && !splitAt.IsZero() {
		// user provided a split from Grafana
		T0 = *splitAt
	} else {
		// fallback to old behavior
		T0 = now.Add(-liveWin)
	}
	log("T0 decided", map[string]interface{}{
		"T0":      T0,
		"now":     now,
		"liveWin": liveWin,
		"from":    from,
		"to":      to,
	})

	// per-ID logic with 3 cases
	for _, id := range telemetryIDs {
		// CASE 1: whole query before T0 -> DB only (fallback to Yamcs if DB empty)
		if to.Before(T0) || to.Equal(T0) {
			log("CASE 1: Query before T0 (DB only)", map[string]interface{}{"param": id})
			dbSeries, err := q.queryDatabaseRange(ctx, archiveClient, from, to, []string{id}, dbSqlFilter)
			if err == nil && len(dbSeries[id]) > 0 {
				log("  Using DB data", map[string]interface{}{"param": id, "pointCount": len(dbSeries[id])})
				out[id] = dbSeries[id]
				continue
			}
			log("  DB empty, falling back to Yamcs", map[string]interface{}{"param": id, "dbErr": err})
			var samples []client.Sample
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
			log("  Got Yamcs data", map[string]interface{}{"param": id, "pointCount": len(samples)})
			out[id] = samplesToPoints(samples)
			continue
		}

		// CASE 2: whole query after T0 -> Yamcs only
		if from.After(T0) || from.Equal(T0) {
			log("CASE 2: Query after T0 (Yamcs only)", map[string]interface{}{"param": id})
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
			log("  Got Yamcs data", map[string]interface{}{"param": id, "pointCount": len(samples)})
			out[id] = samplesToPoints(samples)
			continue
		}

		// CASE 3: spans T0 -> DB(before T0) + Yamcs(after T0)
		log("CASE 3: Query spans T0 (DB before + Yamcs after)", map[string]interface{}{"param": id, "T0": T0})
		dbSeries, err := q.queryDatabaseRange(ctx, archiveClient, from, T0, []string{id}, dbSqlFilter)
		merged := make([]TelemetryPoint, 0)

		// Add DB points strictly before T0
		dbCount := 0
		if err == nil {
			for _, pt := range dbSeries[id] {
				if pt.Time.Before(T0) {
					merged = append(merged, pt)
					dbCount++
				}
			}
		}
		log("  Added DB data", map[string]interface{}{"param": id, "count": dbCount, "dbErr": err})

		// then append Yamcs points from T0 .. to
		var samples []client.Sample
		// Use filtered query if yamcsFilter is provided
		if yamcsFilter != nil && yamcsFilter.Enabled && yamcsFilter.Parameter != "" && yamcsFilter.Value != "" {
			samples, err = yamcsClient.GetParameterSamplesInProcessorByNamesWithFilter(
				instance.GetName(),
				processor.GetName(),
				id,
				T0,
				to,
				yamcsFilter.Parameter,
				yamcsFilter.Value,
			)
		} else {
			samples, err = yamcsClient.GetParameterSamplesInProcessorByNames(
				instance.GetName(),
				processor.GetName(),
				id,
				T0,
				to,
			)
		}
		if err != nil {
			return nil, err
		}
		yamcsPoints := samplesToPoints(samples)
		merged = append(merged, yamcsPoints...)
		log("  Added Yamcs data", map[string]interface{}{"param": id, "count": len(yamcsPoints), "total": len(merged)})

		out[id] = merged
	}

	return out, nil
}

// queryDatabaseRange queries the archive database for a specific time range, only if configured.
// Returns data keyed by parameter name.
// sqlFilter is an optional SQL WHERE clause fragment to further filter results.
func (q *Querier) queryDatabaseRange(ctx context.Context, archive *database.Client, from, to time.Time, telemetryIDs []string, sqlFilter string) (map[string][]TelemetryPoint, error) {
	out := make(map[string][]TelemetryPoint)

	// If no archive database, return empty results
	if archive == nil {
		for _, id := range telemetryIDs {
			out[id] = []TelemetryPoint{}
		}
		return out, nil
	}

	// Look up telemetry IDs by name
	idMap, err := archive.LookupTelemetryID(ctx, telemetryIDs)
	if err != nil {
		return out, err
	}

	// Query database using the looked-up IDs
	dbDataByID, err := archive.QueryTelemetryByID(ctx, from, to, idMap, sqlFilter)
	if err != nil {
		return out, err
	}

	// Map results back to parameter names and convert to TelemetryPoint
	for paramName, id := range idMap {
		if pts, ok := dbDataByID[id]; ok {
			points := make([]TelemetryPoint, 0, len(pts))
			for _, p := range pts {
				if p.Value == nil {
					continue
				}
				points = append(points, TelemetryPoint{Time: p.Time, Value: p.Value, Status: p.Status})
			}
			out[paramName] = points
		} else {
			out[paramName] = []TelemetryPoint{}
		}
	}

	return out, nil
}

// queryYamcsOnly queries data directly from Yamcs without database splitting.
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
