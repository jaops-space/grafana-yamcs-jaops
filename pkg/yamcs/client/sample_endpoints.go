package client

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
)

// SetSamplePointCount sets the sample point count in the client.
func (client *YamcsClient) SetSamplePointCount(count int) {
	client.SamplePointCount.Set(count)
}

// ClearSamplePointCount clears the sample point count in the client.
func (client *YamcsClient) ClearSamplePointCount() {
	client.SamplePointCount.Clear()
}

// setTime sets the start and end times in the HTTP query parameters.
func (client *YamcsClient) setTime(start time.Time, end time.Time) {
	client.HTTP.Query["start"] = start.Format(time.RFC3339)
	client.HTTP.Query["stop"] = end.Format(time.RFC3339)
}

// setTimeAndSampleCount sets both the time range and the sample point count in the HTTP query parameters.
func (client *YamcsClient) setTimeAndSampleCount(start time.Time, end time.Time) {
	client.setTime(start, end)
	if client.SamplePointCount.IsPresent() {
		client.HTTP.Query["count"] = strconv.FormatInt(int64(client.SamplePointCount.Get()), 10)
	}
}

// setFilter sets the filter parameter and value in the HTTP query parameters.
// This allows filtering parameter samples by another parameter's value (e.g., filter Temperature where vcid=1)
func (client *YamcsClient) setFilter(parameterFqn string, value string) {
	if parameterFqn != "" && value != "" {
		// Use dot notation for nested filter structure: filter.parameter and filter.value
		client.HTTP.Query["filter.parameter"] = parameterFqn
		client.HTTP.Query["filter.operator"] = "EQUALS"
		client.HTTP.Query["filter.values"] = value
	}
}

// clearFilter removes filter parameters from the HTTP query parameters.
func (client *YamcsClient) clearFilter() {
	delete(client.HTTP.Query, "filter.parameter")
	delete(client.HTTP.Query, "filter.operator")
	delete(client.HTTP.Query, "filter.values")
}

// GetParameterSamples retrieves parameter samples for a given instance and parameter within a time range.
func (client *YamcsClient) GetParameterSamples(instance Instance, parameter Parameter, start time.Time, end time.Time) ([]Sample, error) {
	client.setTimeAndSampleCount(start, end)

	result := &pvalue.TimeSeries{}
	err := client.HTTP.GetProto(fmt.Sprintf("/archive/%s/parameters/%s/samples", instance.GetName(), parameter.GetName()), result)
	if err != nil {
		return nil, err
	}

	return result.GetSample(), nil
}

// GetParameterSamplesByNames retrieves parameter samples for a given instance and parameter (by name) within a time range.
func (client *YamcsClient) GetParameterSamplesByNames(instance Instance, parameter string, start time.Time, end time.Time) ([]Sample, error) {
	client.setTimeAndSampleCount(start, end)

	result := &pvalue.TimeSeries{}
	err := client.HTTP.GetProto(fmt.Sprintf("/archive/%s/parameters/%s/samples", instance.GetName(), parameter), result)
	if err != nil {
		return nil, err
	}

	return result.GetSample(), nil
}

// GetParameterSamplesInProcessor retrieves parameter samples within a specified processor, instance, and parameter within a time range.
func (client *YamcsClient) GetParameterSamplesInProcessor(instance Instance, processor Processor, parameter Parameter, start time.Time, end time.Time) ([]Sample, error) {
	client.setTimeAndSampleCount(start, end)

	result := &pvalue.TimeSeries{}
	err := client.HTTP.GetProto(fmt.Sprintf("/archive/%s/parameters/%s/samples", instance.GetName(), parameter.GetName()), result)
	if err != nil {
		return nil, err
	}

	return result.GetSample(), nil
}

// GetParameterSamplesInProcessorByNames retrieves parameter samples within a specified processor, instance, and parameter (by name) within a time range.
func (client *YamcsClient) GetParameterSamplesInProcessorByNames(instanceName string, processorName string, parameterName string, start time.Time, end time.Time) ([]Sample, error) {
	client.setTimeAndSampleCount(start, end)

	result := &pvalue.TimeSeries{}
	err := client.HTTP.GetProto(fmt.Sprintf("/archive/%s/parameters/%s/samples", instanceName, parameterName), result)
	if err != nil {
		return nil, err
	}

	return result.GetSample(), nil
}
