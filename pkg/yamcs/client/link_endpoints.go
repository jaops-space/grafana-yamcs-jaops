package client

import (
	"fmt"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"google.golang.org/protobuf/types/known/structpb"
)

// ListLinks retrieves all links for a given Yamcs instance.
func (c *YamcsClient) ListLinks(instance Instance) ([]*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s", instance.GetName())

	response := &links.ListLinksResponse{}
	err := c.HTTP.GetProto(url, response)
	if err != nil {
		return nil, err
	}

	return response.GetLinks(), nil
}

// GetLink retrieves a specific link by name for a given Yamcs instance.
func (c *YamcsClient) GetLink(instance Instance, linkName string) (*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s/%s", instance.GetName(), linkName)

	response := &links.LinkInfo{}
	err := c.HTTP.GetProto(url, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// EnableLink enables a link in a given Yamcs instance.
func (c *YamcsClient) EnableLink(instance Instance, linkName string) (*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s/%s:enable", instance.GetName(), linkName)

	request := &links.EnableLinkRequest{}
	response := &links.LinkInfo{}
	err := c.HTTP.PostProto(url, request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// DisableLink disables a link in a given Yamcs instance.
func (c *YamcsClient) DisableLink(instance Instance, linkName string) (*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s/%s:disable", instance.GetName(), linkName)

	request := &links.DisableLinkRequest{}
	response := &links.LinkInfo{}
	err := c.HTTP.PostProto(url, request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// ResetLinkCounters resets the data in/out counters for a link.
func (c *YamcsClient) ResetLinkCounters(instance Instance, linkName string) (*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s/%s:resetCounters", instance.GetName(), linkName)

	request := &links.ResetLinkCountersRequest{}
	response := &links.LinkInfo{}
	err := c.HTTP.PostProto(url, request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// RunLinkAction runs a link-specific action with an optional message payload.
func (c *YamcsClient) RunLinkAction(instance Instance, linkName string, actionID string, message map[string]any) (*structpb.Struct, error) {
	url := fmt.Sprintf("/links/%s/%s/actions/%s", instance.GetName(), linkName, actionID)

	// Convert message map to structpb.Struct
	var messageStruct *structpb.Struct
	if message != nil {
		var err error
		messageStruct, err = structpb.NewStruct(message)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message to struct: %w", err)
		}
	}

	request := &links.RunActionRequest{
		Message: messageStruct,
	}

	response := &structpb.Struct{}
	err := c.HTTP.PostProto(url, request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
