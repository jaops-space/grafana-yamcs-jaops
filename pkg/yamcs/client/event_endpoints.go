package client

import (
	"context"
	"fmt"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
)

// ListEvents returns a paginated iterator for fetching events from a specified instance.
func (c *YamcsClient) ListEvents(ctx context.Context, instance string) *types.PaginatedRequestIterator[[]*events.Event] {
	return types.NewPaginatedRequestIterator(c.HTTP, c.fetchEventBatch(ctx, instance))
}

// ListEventsWithinTimeRange returns a paginated iterator for fetching events within a given time range.
func (c *YamcsClient) ListEventsWithinTimeRange(ctx context.Context, instance string, startTime, endTime time.Time) *types.PaginatedRequestIterator[[]*events.Event] {
	iterator := types.NewPaginatedRequestIterator(c.HTTP, c.fetchEventBatch(ctx, instance))
	iterator.SetQuery(map[string]string{
		"start": startTime.Format(time.RFC3339),
		"stop":  endTime.Format(time.RFC3339),
	})
	return iterator
}

// fetchEventBatch retrieves a batch of events from the given instance.
func (c *YamcsClient) fetchEventBatch(ctx context.Context, instance string) types.FetchFunction[[]*events.Event] {
	return func() ([]*events.Event, string, error) {
		response := &events.ListEventsResponse{}
		err := c.HTTP.GetProto(ctx, fmt.Sprintf("/archive/%s/events", instance), response)
		if err != nil {
			return nil, "", err
		}
		return response.GetEvents(), response.GetContinuationToken(), nil
	}
}
