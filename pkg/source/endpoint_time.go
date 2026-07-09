package source

import (
	"context"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
)

func (ep *YamcsEndpoint) RequestTime(ctx context.Context) error {

	client, err := ep.GetClient()
	if err != nil {
		return err
	}

	subscription, found := client.GetTimeSubscription(ep.GetInstanceName(), ep.GetProcessorName())
	if !found {
		var err error
		subscription, err = client.CreateTimeSubscription(ep.GetInstanceName(), ep.GetProcessorName())
		if err != nil {
			return err
		}
	}

	subscription.AddTimeListener(ep.getTimeHandler())

	return nil
}

func (ep *YamcsEndpoint) getTimeHandler() func(t time.Time) {
	return func(currentTime time.Time) {
		ep.SetCurrentTime(currentTime)
	}
}

func (ep *YamcsEndpoint) SetCurrentTime(currentTime time.Time) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.CurrentTime = currentTime
	ep.CurrentTimeUpdatedAt = time.Now()
}

func (ep *YamcsEndpoint) GetCurrentTime() (time.Time, time.Time, bool) {

	ep.mu.RLock()
	defer ep.mu.RUnlock()

	if ep.CurrentTime.IsZero() || ep.CurrentTimeUpdatedAt.IsZero() {
		return time.Time{}, time.Time{}, false
	}

	return ep.CurrentTime, ep.CurrentTimeUpdatedAt, true
}

func (ep *YamcsEndpoint) GetCurrentTimeIfFresh(maxAge time.Duration) (time.Time, bool) {

	currentTime, updatedAt, ok := ep.GetCurrentTime()
	if !ok {
		return time.Time{}, false
	}

	if maxAge > 0 && time.Since(updatedAt) > maxAge {
		return time.Time{}, false
	}

	return currentTime, true
}

// GetReplaySpeedMultiplier returns the processor replay speed multiplier for this endpoint.
func (ep *YamcsEndpoint) GetReplaySpeedMultiplier() (float64, error) {

	ep.mu.RLock()
	processor, err := ep.GetProcessor()
	ep.mu.RUnlock()

	if err != nil {
		return 1, err
	}

	if !processor.GetReplay() {
		return 1, nil
	}

	replayRequest := processor.GetReplayRequest()
	if replayRequest == nil || replayRequest.GetSpeed() == nil {
		return 1, nil
	}

	speed := replayRequest.GetSpeed()
	if speed.GetType() != protobuf.ReplaySpeed_REALTIME {
		return 1, nil
	}

	return float64(speed.GetParam()), nil

}
