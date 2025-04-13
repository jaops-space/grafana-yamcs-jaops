package client

import (
	"fmt"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/processing"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
)

// GetProcessor retrieves a processor by its name from the specified instance.
// If the processor is not found, an error is returned.
func (client *YamcsClient) GetProcessor(instance Instance, name string) (Processor, error) {
	for _, processor := range instance.GetProcessors() {
		if processor.GetName() == name {
			return processor, nil
		}
	}

	return nil, exception.New(fmt.Sprintf("Processor %s or Instance %s not found", name, instance.GetName()), "PROCESSOR_NOT_FOUND")
}

// ListInstanceProcessorsByName retrieves all processors by their name for the specified instance.
// It performs a GET request to fetch processor data and returns a list of processors.
func (client *YamcsClient) ListInstanceProcessorsByName(instanceName string) ([]Processor, error) {
	processorsResponse := &processing.ListProcessorsResponse{}

	// Set the query parameter for the instance
	client.HTTP.Query["instance"] = instanceName
	err := client.HTTP.GetProto("/processors", processorsResponse)
	if err != nil {
		return nil, err
	}

	return processorsResponse.GetProcessors(), nil
}

// ListInstanceProcessors retrieves all processors for a given instance.
// It performs a GET request to fetch processor data and returns a list of processors.
func (client *YamcsClient) ListInstanceProcessors(instance Instance) ([]Processor, error) {
	processorsResponse := &processing.ListProcessorsResponse{}

	// Set the query parameter for the instance
	client.HTTP.Query["instance"] = instance.GetName()
	err := client.HTTP.GetProto("/processors", processorsResponse)
	if err != nil {
		return nil, err
	}

	return processorsResponse.GetProcessors(), nil
}

// GetInstanceDefaultProcessor returns the first processor from the specified instance's processor list.
// It assumes the instance has at least one processor.
func (client *YamcsClient) GetInstanceDefaultProcessor(instance Instance) Processor {
	if len(instance.Processors) > 0 {
		return instance.Processors[0]
	}
	return nil
}

// GetInstanceDefaultProcessorByName retrieves the default processor by its name for a given instance.
// It fetches the processors for the instance and returns the first one.
func (client *YamcsClient) GetInstanceDefaultProcessorByName(instanceName string) (Processor, error) {
	processors, err := client.ListInstanceProcessorsByName(instanceName)
	if err != nil {
		return nil, err
	}
	if len(processors) > 0 {
		return processors[0], nil
	}
	return nil, exception.New(fmt.Sprintf("No processors found for instance %s", instanceName), "PROCESSOR_NOT_FOUND")
}
