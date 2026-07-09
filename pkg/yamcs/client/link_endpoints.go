package client

import (
	"context"
	"fmt"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"google.golang.org/protobuf/types/known/structpb"
)

// ListLinks retrieves all links for a given Yamcs instance.
func (c *YamcsClient) ListLinks(ctx context.Context, instance string) ([]*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s", instance)

	response := &links.ListLinksResponse{}
	err := c.HTTP.GetProto(ctx, url, response)
	if err != nil {
		return nil, err
	}

	return response.GetLinks(), nil
}

// GetLink retrieves a specific link by name for a given Yamcs instance.
func (c *YamcsClient) GetLink(ctx context.Context, instance string, linkName string) (*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s/%s", instance, linkName)

	response := &links.LinkInfo{}
	err := c.HTTP.GetProto(ctx, url, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// EnableLink enables a link in a given Yamcs instance.
func (c *YamcsClient) EnableLink(ctx context.Context, instance string, linkName string) (*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s/%s:enable", instance, linkName)

	request := &links.EnableLinkRequest{}
	response := &links.LinkInfo{}
	err := c.HTTP.PostProto(ctx, url, request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// DisableLink disables a link in a given Yamcs instance.
func (c *YamcsClient) DisableLink(ctx context.Context, instance string, linkName string) (*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s/%s:disable", instance, linkName)

	request := &links.DisableLinkRequest{}
	response := &links.LinkInfo{}
	err := c.HTTP.PostProto(ctx, url, request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// ResetLinkCounters resets the data in/out counters for a link.
func (c *YamcsClient) ResetLinkCounters(ctx context.Context, instance string, linkName string) (*links.LinkInfo, error) {
	url := fmt.Sprintf("/links/%s/%s:resetCounters", instance, linkName)

	request := &links.ResetLinkCountersRequest{}
	response := &links.LinkInfo{}
	err := c.HTTP.PostProto(ctx, url, request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// RunLinkAction runs a link-specific action with an optional message payload.
func (c *YamcsClient) RunLinkAction(ctx context.Context, instance string, linkName string, actionID string, message map[string]any) (*structpb.Struct, error) {
	url := fmt.Sprintf("/links/%s/%s/actions/%s", instance, linkName, actionID)

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
	err := c.HTTP.PostProto(ctx, url, request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
