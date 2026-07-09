package client

import (
	"context"
	"fmt"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/archive"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/mdb"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
)

// getParametersFetchMethod returns a fetch function for paginated parameter results.
func (client *YamcsClient) getParametersFetchMethod(ctx context.Context, instance string) types.FetchFunction[[]Parameter] {
	return func() ([]Parameter, string, error) {
		response := &mdb.ListParametersResponse{}
		err := client.HTTP.GetProto(ctx, fmt.Sprintf("/mdb/%s/parameters", instance), response)
		if err != nil {
			return nil, "", err
		}
		return response.GetParameters(), response.GetContinuationToken(), nil
	}
}

// ListParameters retrieves a list of parameters for a given instance.
func (client *YamcsClient) ListParameters(ctx context.Context, instance Instance) *types.PaginatedRequestIterator[[]Parameter] {
	iterator := types.NewPaginatedRequestIterator(client.HTTP, client.getParametersFetchMethod(ctx, instance.GetName()))
	return iterator
}

// ListParametersByInstanceName retrieves a list of parameters by instance name.
func (client *YamcsClient) ListParametersByInstanceName(ctx context.Context, instance string) *types.PaginatedRequestIterator[[]Parameter] {
	iterator := types.NewPaginatedRequestIterator(client.HTTP, client.getParametersFetchMethod(ctx, instance))
	return iterator
}

// SearchParameters searches for parameters matching the search query in a specific instance.
func (client *YamcsClient) SearchParameters(ctx context.Context, instance string, query string) *types.PaginatedRequestIterator[[]Parameter] {
	iterator := types.NewPaginatedRequestIterator(client.HTTP, client.getParametersFetchMethod(ctx, instance))
	iterator.SetQuery(map[string]string{"q": query})
	return iterator
}

// GetParameter retrieves a specific parameter's info for an instance.
func (client *YamcsClient) GetParameter(ctx context.Context, instance string, parameter string) (Parameter, error) {
	response := &mdb.ParameterInfo{}
	err := client.HTTP.GetProto(ctx, fmt.Sprintf("/mdb/%s/parameters/%s", instance, parameter), response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetParameterRanges retrieves the ranges of a specific parameter in a given instance.
func (client *YamcsClient) GetParameterRanges(ctx context.Context, instance Instance, parameter Parameter) (*pvalue.Ranges, error) {
	url := fmt.Sprintf("/archive/%s/parameters/%s/ranges", instance.GetName(), parameter.GetQualifiedName())
	ranges := &pvalue.Ranges{}
	err := client.HTTP.GetProto(ctx, url, ranges)
	if err != nil {
		return nil, err
	}
	return ranges, nil
}

// GetParameterRangesWithTime retrieves parameter ranges within a specific time range.
func (client *YamcsClient) GetParameterRangesByQueryWithTimeByNames(ctx context.Context, instance, parameter string, query map[string]string, start, end time.Time) (*pvalue.Ranges, error) {
	url := fmt.Sprintf("/archive/%s/parameters/%s/ranges", instance, parameter)
	ranges := &pvalue.Ranges{}
	client.HTTP.Query = query
	client.setTime(start, end)
	err := client.HTTP.GetProto(ctx, url, ranges)
	if err != nil {
		return nil, err
	}
	return ranges, nil
}

// ListParameterHistory retrieves the history of a parameter in a given instance.
func (client *YamcsClient) ListParameterHistory(ctx context.Context, instance Instance, parameter Parameter) *types.PaginatedRequestIterator[[]*pvalue.ParameterValue] {
	iterator := types.NewPaginatedRequestIterator(client.HTTP, client.getParameterHistoryFetchMethod(ctx, instance.GetName(), parameter.GetQualifiedName()))
	return iterator
}

// getParameterHistoryFetchMethod returns a fetch function for paginated parameter history results.
func (client *YamcsClient) getParameterHistoryFetchMethod(ctx context.Context, instance string, parameter string) types.FetchFunction[[]*pvalue.ParameterValue] {
	return func() ([]*pvalue.ParameterValue, string, error) {
		response := &archive.ListParameterHistoryResponse{}
		err := client.HTTP.GetProto(ctx, fmt.Sprintf("/archive/%s/parameters/%s", instance, parameter), response)
		if err != nil {
			return nil, "", err
		}
		return response.GetParameter(), response.GetContinuationToken(), nil
	}
}

// GetParameter retrieves a specific parameter's value for a processor in a given instance.
func (client *YamcsClient) GetParameterValue(ctx context.Context, instance Instance, processor Processor, parameter Parameter) (ParameterValue, error) {
	response := &pvalue.ParameterValue{}
	err := client.HTTP.GetProto(ctx, fmt.Sprintf("/processors/%s/%s/parameters/%s", instance.GetName(), processor.GetName(), parameter.GetQualifiedName()), response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetParameter retrieves a specific parameter's value for a processor in a given instance.
func (client *YamcsClient) GetParameterValueByName(ctx context.Context, instance string, processor string, parameter string) (ParameterValue, error) {
	response := &pvalue.ParameterValue{}
	err := client.HTTP.GetProto(ctx, fmt.Sprintf("/processors/%s/%s/parameters/%s", instance, processor, parameter), response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
