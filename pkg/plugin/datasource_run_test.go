package plugin

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/instances"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/yamcsManagement"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/ws"
)

type testStreamPacketSender struct {
	mu      sync.Mutex
	packets []*backend.StreamPacket
}

func (s *testStreamPacketSender) Send(packet *backend.StreamPacket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.packets = append(s.packets, packet)
	return nil
}

func (s *testStreamPacketSender) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.packets)
}

func buildTestEndpointWithClient(connected bool) *source.YamcsEndpoint {
	instanceName := "sim"
	processorName := "realtime"
	instance := &instances.YamcsInstance{Name: &instanceName}
	processor := &yamcsManagement.ProcessorInfo{Name: &processorName}

	wsHandler := ws.NewWebSocketHandler("ws://example.invalid", true)
	_ = connected // Reserved for future connected-path tests once websocket state is injectable.

	c := &client.YamcsClient{
		WebSocket:                   wsHandler,
		ParameterSubscriptions:      map[int32]*client.ParameterSubscription{},
		EventSubscriptions:          map[int32]*client.EventSubscription{},
		CommandHistorySubscriptions: map[int32]*client.CommandHistorySubscription{},
		AlarmSubscriptions:          map[int32]*client.AlarmSubscription{},
		LinkSubscriptions:           map[int32]*client.LinkSubscription{},
		TimeSubscriptions:           map[int32]*client.TimeSubscription{},
	}

	host := &source.YamcsHost{
		Client: c,
		Instances: map[string]*source.YamcsHostInstance{
			instanceName: {
				Instance: instance,
				Processors: map[string]client.Processor{
					processorName: processor,
				},
			},
		},
	}
	mux := &source.Multiplexer{
		Hosts: map[string]*source.YamcsHost{
			"h1": host,
		},
	}

	return &source.YamcsEndpoint{
		Multiplexer:           mux,
		Host:                  host,
		ID:                    "e1",
		Configuration:         &config.YamcsEndpointConfiguration{Host: "h1", Instance: instanceName, Processor: processorName},
		Parameters:            map[string]*source.ParameterDemand{},
		Events:                map[string]chan *events.Event{},
		CommandHistorySignals: map[string]source.CommandHistorySignal{},
		Alarms:                map[string][]*alarms.AlarmData{},
		AlarmSignals:          map[string]chan struct{}{},
		LinkSignals:           map[string]source.LinkSignal{},
		AlarmCache:            map[string]*alarms.AlarmData{},
	}
}

func TestGetStreamTickerInterval(t *testing.T) {
	tests := []struct {
		name     string
		q        PluginQuery
		fallback time.Duration
		want     time.Duration
	}{
		{
			name:     "invalid max points uses fallback",
			q:        PluginQuery{From: 0, To: 10, MaxPoints: 0},
			fallback: time.Second,
			want:     time.Second,
		},
		{
			name:     "invalid time range uses fallback",
			q:        PluginQuery{From: 10, To: 10, MaxPoints: 5},
			fallback: 2 * time.Second,
			want:     2 * time.Second,
		},
		{
			name:     "clamps low interval",
			q:        PluginQuery{From: 0, To: 1, MaxPoints: 1000},
			fallback: time.Second,
			want:     200 * time.Millisecond,
		},
		{
			name:     "clamps high interval",
			q:        PluginQuery{From: 0, To: 4000, MaxPoints: 1},
			fallback: time.Second,
			want:     30 * time.Second,
		},
		{
			name:     "returns computed interval",
			q:        PluginQuery{From: 0, To: 10, MaxPoints: 5},
			fallback: time.Second,
			want:     2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStreamTickerInterval(tt.q, tt.fallback)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRunDemandsStream_ContextCancelExits(t *testing.T) {
	endpoint := buildTestEndpointWithClient(false)
	packetSender := &testStreamPacketSender{}
	sender := backend.NewStreamSender(packetSender)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RunDemandsStream(ctx, &backend.RunStreamRequest{Path: "req/path"}, sender, endpoint, PluginQuery{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if packetSender.Count() != 0 {
		t.Fatalf("expected no packets after immediate cancellation, got %d", packetSender.Count())
	}
}

func TestRunSubscriptionStream_NoClientFailsFast(t *testing.T) {
	instanceName := "sim"
	processorName := "realtime"
	host := &source.YamcsHost{Client: nil}

	endpoint := &source.YamcsEndpoint{
		Multiplexer: &source.Multiplexer{
			Hosts: map[string]*source.YamcsHost{
				"h1": host,
			},
		},
		Host:          host,
		ID:            "e1",
		Configuration: &config.YamcsEndpointConfiguration{Host: "h1", Instance: instanceName, Processor: processorName},
	}

	packetSender := &testStreamPacketSender{}
	sender := backend.NewStreamSender(packetSender)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- RunSubscriptionStream(ctx, &backend.RunStreamRequest{Path: "req/path"}, sender, endpoint, PluginQuery{})
	}()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "client not found") {
			t.Fatalf("expected no client error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunSubscriptionStream did not fail fast")
	}
}

func TestRunTimeStream_DisconnectedClientErrors(t *testing.T) {
	endpoint := buildTestEndpointWithClient(false)
	packetSender := &testStreamPacketSender{}
	sender := backend.NewStreamSender(packetSender)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := RunTimeStream(ctx, &backend.RunStreamRequest{Path: "req/path"}, sender, endpoint, PluginQuery{})
	if err == nil || !strings.Contains(err.Error(), "yamcs client disconnected") {
		t.Fatalf("expected disconnected error, got %v", err)
	}
}

func TestRunParameterStream_NilClientFailsFast(t *testing.T) {
	instanceName := "sim"
	processorName := "realtime"
	host := &source.YamcsHost{Client: nil}

	endpoint := &source.YamcsEndpoint{
		Multiplexer: &source.Multiplexer{
			Hosts: map[string]*source.YamcsHost{
				"h1": host,
			},
		},
		Host:                  host,
		ID:                    "e1",
		Configuration:         &config.YamcsEndpointConfiguration{Host: "h1", Instance: instanceName, Processor: processorName},
		Parameters:            map[string]*source.ParameterDemand{},
		Events:                map[string]chan *events.Event{},
		CommandHistorySignals: map[string]source.CommandHistorySignal{},
		Alarms:                map[string][]*alarms.AlarmData{},
		AlarmSignals:          map[string]chan struct{}{},
		LinkSignals:           map[string]source.LinkSignal{},
		AlarmCache:            map[string]*alarms.AlarmData{},
	}

	packetSender := &testStreamPacketSender{}
	sender := backend.NewStreamSender(packetSender)

	err := RunParameterStream(context.Background(), &backend.RunStreamRequest{Path: "req/path"}, sender, endpoint, PluginQuery{Parameter: "p", MaxPoints: 100, From: 0, To: 10})
	if err == nil || !strings.Contains(err.Error(), "client not found") {
		t.Fatalf("expected no client error, got %v", err)
	}
}

func TestRunEventStream_DisconnectedClientErrors(t *testing.T) {
	endpoint := buildTestEndpointWithClient(false)
	packetSender := &testStreamPacketSender{}
	sender := backend.NewStreamSender(packetSender)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- RunEventStream(ctx, &backend.RunStreamRequest{Path: "req/path"}, sender, endpoint, PluginQuery{})
	}()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "yamcs client disconnected") {
			t.Fatalf("expected disconnected error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunEventStream did not return on disconnected client")
	}
}

func TestRunLinksStream_DisconnectedClientErrors(t *testing.T) {
	endpoint := buildTestEndpointWithClient(false)
	packetSender := &testStreamPacketSender{}
	sender := backend.NewStreamSender(packetSender)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- RunLinksStream(ctx, &backend.RunStreamRequest{Path: "req/path"}, sender, endpoint, PluginQuery{From: 0, To: 10, MaxPoints: 10})
	}()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "yamcs client disconnected") {
			t.Fatalf("expected disconnected error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunLinksStream did not return on disconnected client")
	}
}
