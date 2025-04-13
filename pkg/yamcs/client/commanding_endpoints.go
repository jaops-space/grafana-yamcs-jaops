package client

import (
	"fmt"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/mdb"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"google.golang.org/protobuf/types/known/structpb"
)

// IssueCommand sends a command to the specified Yamcs processor with given arguments.
func (c *YamcsClient) IssueCommand(instance Instance, processor Processor, commandName string, args map[string]any) (*commanding.IssueCommandResponse, error) {
	return c.issueCommand(instance, processor, commandName, args, "", nil, nil, false, nil, nil, nil, nil, nil)
}

// IssueCommandWithComment sends a command with an attached comment.
func (c *YamcsClient) IssueCommandWithComment(instance Instance, processor Processor, commandName string, args map[string]any, comment string) (*commanding.IssueCommandResponse, error) {
	return c.issueCommand(instance, processor, commandName, args, comment, nil, nil, false, nil, nil, nil, nil, nil)
}

// IssueCommandWithOptions sends a command with additional options but without elevated privileges.
func (c *YamcsClient) IssueCommandWithOptions(instance Instance, processor Processor, commandName string, args map[string]any, origin string, sequenceNumber int32, dryRun bool, comment string, extra map[string]*protobuf.Value) (*commanding.IssueCommandResponse, error) {
	return c.issueCommand(instance, processor, commandName, args, comment, &origin, &sequenceNumber, dryRun, nil, nil, nil, nil, extra)
}

// IssueCommandWithElevatedPrivileges sends a command with all options, including those requiring elevated privileges.
func (c *YamcsClient) IssueCommandWithElevatedPrivileges(instance Instance, processor Processor, commandName string, args map[string]any, origin string, sequenceNumber int32, dryRun bool, comment string, stream string, disableTransmissionConstraints, disableVerifiers bool, verifierConfig map[string]*commanding.VerifierConfig, extra map[string]*protobuf.Value) (*commanding.IssueCommandResponse, error) {
	return c.issueCommand(instance, processor, commandName, args, comment, &origin, &sequenceNumber, dryRun, &stream, &disableTransmissionConstraints, &disableVerifiers, verifierConfig, extra)
}

// issueCommand handles command execution with optional parameters.
func (c *YamcsClient) issueCommand(instance Instance, processor Processor, commandName string, args map[string]any, comment string, origin *string, sequenceNumber *int32, dryRun bool, stream *string, disableTransmissionConstraints, disableVerifiers *bool, verifierConfig map[string]*commanding.VerifierConfig, extra map[string]*protobuf.Value) (*commanding.IssueCommandResponse, error) {
	url := fmt.Sprintf("/processors/%s/%s/commands/%s", instance.GetName(), processor.GetName(), commandName)

	argsProto, err := convertMap(args)
	if err != nil {
		return nil, err
	}

	commandRequest := &commanding.IssueCommandRequest{
		Args:                           &structpb.Struct{Fields: argsProto},
		Comment:                        &comment,
		Origin:                         origin,
		DryRun:                         &dryRun,
		Stream:                         stream,
		DisableTransmissionConstraints: disableTransmissionConstraints,
		DisableVerifiers:               disableVerifiers,
		VerifierConfig:                 verifierConfig,
		Extra:                          extra,
		SequenceNumber:                 sequenceNumber,
	}

	commandResponse := &commanding.IssueCommandResponse{}
	err = c.HTTP.PostProto(url, commandRequest, commandResponse)
	if err != nil {
		return nil, err
	}

	return commandResponse, nil
}

// GetCommand retrieves command history entry by instance and ID.
func (c *YamcsClient) GetCommand(instance, id string) (*commanding.CommandHistoryEntry, error) {
	url := fmt.Sprintf("/archive/%s/commands/%s", instance, id)
	command := &commanding.CommandHistoryEntry{}
	if err := c.HTTP.GetProto(url, command); err != nil {
		return nil, err
	}
	return command, nil
}

// ListCommandsHistory returns an iterator over command history entries.
func (c *YamcsClient) ListCommandsHistory(instance Instance) *types.PaginatedRequestIterator[[]*commanding.CommandHistoryEntry] {
	return types.NewPaginatedRequestIterator(c.HTTP, c.getCommandsHistoryFetcher(instance.GetName()))
}

func (c *YamcsClient) getCommandsHistoryFetcher(instance string) types.FetchFunction[[]*commanding.CommandHistoryEntry] {
	return func() ([]*commanding.CommandHistoryEntry, string, error) {
		response := &commanding.ListCommandsResponse{}
		if err := c.HTTP.GetProto(fmt.Sprintf("/archive/%s/commands", instance), response); err != nil {
			return nil, "", err
		}
		return response.Commands, response.GetContinuationToken(), nil
	}
}

// Convert map[string]any to map[string]*structpb.Value
func convertMap(m map[string]any) (map[string]*structpb.Value, error) {
	newMap := make(map[string]*structpb.Value)
	for key, val := range m {
		v, err := structpb.NewValue(val)
		if err != nil {
			return nil, err
		}
		newMap[key] = v
	}
	return newMap, nil
}

// ListCommandInfos retrieves an iterator for all command metadata.
func (c *YamcsClient) ListCommandInfos(instance Instance) *types.PaginatedRequestIterator[[]CommandInfo] {
	return c.SearchCommandInfo(instance, "")
}

// SearchCommandInfo retrieves an iterator for commands matching a search query.
func (c *YamcsClient) SearchCommandInfo(instance Instance, query string) *types.PaginatedRequestIterator[[]CommandInfo] {
	iterator := types.NewPaginatedRequestIterator(c.HTTP, c.getCommandInfoFetcher(instance.GetName()))
	iterator.SetQuery(map[string]string{"q": query})
	return iterator
}

func (c *YamcsClient) getCommandInfoFetcher(instance string) types.FetchFunction[[]CommandInfo] {
	return func() ([]CommandInfo, string, error) {
		response := &mdb.ListCommandsResponse{}
		if err := c.HTTP.GetProto(fmt.Sprintf("/mdb/%s/commands", instance), response); err != nil {
			return nil, "", err
		}
		return response.GetCommands(), response.GetContinuationToken(), nil
	}
}

// GetCommandInfo retrieves metadata for a specific command.
func (c *YamcsClient) GetCommandInfo(instance Instance, command string) (CommandInfo, error) {
	url := fmt.Sprintf("/mdb/%s/commands/%s", instance.GetName(), command)
	info := &mdb.CommandInfo{}
	if err := c.HTTP.GetProto(url, info); err != nil {
		return nil, err
	}
	return info, nil
}
