package client

import (
	"context"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/instances"
)

// ListInstances fetches all available instances from the Yamcs server.
func (client *YamcsClient) ListInstances(ctx context.Context) ([]Instance, error) {
	// Create a response object to store the list of instances
	response := &instances.ListInstancesResponse{}

	// Make an HTTP request to retrieve the instances data
	err := client.HTTP.GetProto(ctx, "/instances", response)
	if err != nil {
		return nil, err
	}

	// Return the list of instances
	return response.GetInstances(), nil
}

// GetInstanceByName fetches a specific instance by its name from the Yamcs server.
func (client *YamcsClient) GetInstanceByName(ctx context.Context, name string) (Instance, error) {
	// Create an object to store the specific instance details
	instance := &instances.YamcsInstance{}

	// Make an HTTP request to retrieve the instance data by name
	err := client.HTTP.GetProto(ctx, "/instances/"+name, instance)
	if err != nil {
		return nil, err
	}

	// Return the instance object
	return instance, nil
}
