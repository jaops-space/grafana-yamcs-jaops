package client

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
)

// buildTimeQuery builds the start/stop query parameters for a time range.
func buildTimeQuery(start time.Time, end time.Time) map[string]string {
	return map[string]string{
		"start": start.Format(time.RFC3339),
		"stop":  end.Format(time.RFC3339),
	}
}

// buildTimeAndSampleCountQuery builds the time range query parameters plus
// an optional sample point count.
func buildTimeAndSampleCountQuery(start time.Time, end time.Time, sampleCount int) map[string]string {
	query := buildTimeQuery(start, end)
	if sampleCount > 0 {
		query["count"] = strconv.FormatInt(int64(sampleCount), 10)
	}
	return query
}

// buildFilterQuery builds the filter query parameters that filter parameter
// samples by another parameter's value (e.g., filter Temperature where
// vcid=1). Returns nil if either argument is empty, meaning "no filter".
func buildFilterQuery(parameterFqn string, value string) map[string]string {
	if parameterFqn == "" || value == "" {
		return nil
	}
	// Use dot notation for nested filter structure: filter.parameter and filter.value
	return map[string]string{
		"filter.parameter": parameterFqn,
		"filter.operator":  "EQUALS",
		"filter.values":    value,
	}
}

// GetParameterSamples retrieves parameter samples for a given instance and parameter within a time range.
func (client *YamcsClient) GetParameterSamples(ctx context.Context, instance Instance, parameter Parameter, start time.Time, end time.Time, sampleCount int) ([]Sample, error) {
	query := buildTimeAndSampleCountQuery(start, end, sampleCount)

	result := &pvalue.TimeSeries{}
	err := client.HTTP.GetProtoWithQuery(ctx, fmt.Sprintf("/archive/%s/parameters/%s/samples", instance.GetName(), parameter.GetName()), query, result)
	if err != nil {
		return nil, err
	}

	return result.GetSample(), nil
}

// GetParameterSamplesByNames retrieves parameter samples for a given instance and parameter (by name) within a time range.
func (client *YamcsClient) GetParameterSamplesByNames(ctx context.Context, instance Instance, parameter string, start time.Time, end time.Time, sampleCount int) ([]Sample, error) {
	query := buildTimeAndSampleCountQuery(start, end, sampleCount)

	result := &pvalue.TimeSeries{}
	err := client.HTTP.GetProtoWithQuery(ctx, fmt.Sprintf("/archive/%s/parameters/%s/samples", instance.GetName(), parameter), query, result)
	if err != nil {
		return nil, err
	}

	return result.GetSample(), nil
}

// GetParameterSamplesInProcessor retrieves parameter samples within a specified processor, instance, and parameter within a time range.
func (client *YamcsClient) GetParameterSamplesInProcessor(ctx context.Context, instance Instance, processor Processor, parameter Parameter, start time.Time, end time.Time, sampleCount int) ([]Sample, error) {
	query := buildTimeAndSampleCountQuery(start, end, sampleCount)

	result := &pvalue.TimeSeries{}
	err := client.HTTP.GetProtoWithQuery(ctx, fmt.Sprintf("/archive/%s/parameters/%s/samples", instance.GetName(), parameter.GetName()), query, result)
	if err != nil {
		return nil, err
	}

	return result.GetSample(), nil
}

// GetParameterSamplesInProcessorByNames retrieves parameter samples within a specified processor, instance, and parameter (by name) within a time range.
func (client *YamcsClient) GetParameterSamplesInProcessorByNames(ctx context.Context, instanceName string, processorName string, parameterName string, start time.Time, end time.Time, sampleCount int) ([]Sample, error) {
	query := buildTimeAndSampleCountQuery(start, end, sampleCount)

	result := &pvalue.TimeSeries{}
	err := client.HTTP.GetProtoWithQuery(ctx, fmt.Sprintf("/archive/%s/parameters/%s/samples", instanceName, parameterName), query, result)
	if err != nil {
		return nil, err
	}

	return result.GetSample(), nil
}

// GetParameterSamplesInProcessorByNamesWithFilter retrieves parameter samples with optional filtering.
// If filterParamFqn and filterValue are provided, only returns samples where the filter parameter equals the filter value.
// Example: Get Temperature samples filtered by vcid=1
func (client *YamcsClient) GetParameterSamplesInProcessorByNamesWithFilter(
	ctx context.Context,
	instanceName string,
	processorName string,
	parameterName string,
	start time.Time,
	end time.Time,
	filterParamFqn string,
	filterValue string,
	sampleCount int,
) ([]Sample, error) {
	query := buildTimeAndSampleCountQuery(start, end, sampleCount)
	for k, v := range buildFilterQuery(filterParamFqn, filterValue) {
		query[k] = v
	}

	result := &pvalue.TimeSeries{}
	err := client.HTTP.GetProtoWithQuery(ctx, fmt.Sprintf("/archive/%s/parameters/%s/samples", instanceName, parameterName), query, result)
	if err != nil {
		return nil, err
	}

	return result.GetSample(), nil
}
