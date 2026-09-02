package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/yamcsManagement"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// mockTransport implements http.RoundTripper to mock HTTP requests.
type mockTransport struct{}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{}")), // Set body to "{}"
		Header:     make(http.Header),
	}, nil
}

type closeTrackingClientTransport struct {
	closedIdleConnections bool
}

func (m *closeTrackingClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

func (m *closeTrackingClientTransport) CloseIdleConnections() {
	m.closedIdleConnections = true
}

func TestClient(t *testing.T) {

	client, err := NewYamcsClient(
		"somepath",
		corehttp.GetNoTLSConfiguration(),
		&corehttp.NoCredentials{},
		OptionSetProtocol(false),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Assign mockTransport to the client's HTTP transport
	client.HTTP.Client.Transport = &mockTransport{}
	ctx := context.Background()

	instance, err := client.GetInstanceByName(ctx, "someinstance")
	if err != nil {
		t.Fatalf("Failed to get instance: %v", err)
	}

	someProcessor := &yamcsManagement.ProcessorInfo{}
	someProcessor.Name = new("someprocessor")
	instance.Processors = []Processor{someProcessor}

	_, err = client.GetProcessor(instance, "someprocessor")
	if err != nil {
		t.Fatalf("Failed to get processor: %v", err)
	}

	parameter, err := client.GetParameter(ctx, instance.GetName(), "someparameter")
	if err != nil {
		t.Fatalf("Failed to get parameter: %v", err)
	}

	_, err = client.GetParameterValue(ctx, instance, someProcessor, parameter)
	if err != nil {
		t.Fatalf("Failed to get parameter value: %v", err)
	}

	_, err = client.GetParameterRanges(ctx, instance, parameter)
	if err != nil {
		t.Fatalf("Failed to get parameter ranges: %v", err)
	}

	_, err = client.GetParameterSamples(ctx, instance, parameter, time.Now(), time.Now(), 100)
	if err != nil {
		t.Fatalf("Failed to get parameter samples: %v", err)
	}

	_, err = client.GetCommand(ctx, instance.GetName(), "somecommand")
	if err != nil {
		t.Fatalf("Failed to get command: %v", err)
	}

	_, err = client.IssueCommand(ctx, instance.GetName(), someProcessor.GetName(), "somecommand", make(map[string]any))
	if err != nil {
		t.Fatalf("Failed to issue command: %v", err)
	}

}

func TestClientCloseDisposesHTTPAndClearsSubscriptions(t *testing.T) {
	transport := &closeTrackingClientTransport{}
	client, err := NewYamcsClient(
		"somepath",
		corehttp.GetNoTLSConfiguration(),
		&corehttp.BearerCredentials{
			AccessToken: "access-token",
			Expiry:      time.Now().Add(time.Hour),
		},
		OptionSetProtocol(false),
		OptionSetHTTPClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.ParameterSubscriptions[1] = nil
	client.EventSubscriptions[2] = nil
	client.CommandHistorySubscriptions[3] = nil
	client.AlarmSubscriptions[4] = nil
	client.GlobalAlarmStatusSubscriptions[5] = nil
	client.TimeSubscriptions[6] = nil
	client.LinkSubscriptions[7] = nil
	client.ProcessorSubscriptions[8] = nil
	client.HTTP.StartAutoRefresh()

	if client.HTTP.RefreshStop == nil {
		t.Fatal("expected refresh loop to start")
	}

	_ = client.Close()

	if client.HTTP.RefreshStop != nil {
		t.Fatal("expected refresh loop to stop")
	}
	if !transport.closedIdleConnections {
		t.Fatal("expected HTTP idle connections to close")
	}
	if len(client.ParameterSubscriptions) != 0 ||
		len(client.EventSubscriptions) != 0 ||
		len(client.CommandHistorySubscriptions) != 0 ||
		len(client.AlarmSubscriptions) != 0 ||
		len(client.GlobalAlarmStatusSubscriptions) != 0 ||
		len(client.TimeSubscriptions) != 0 ||
		len(client.LinkSubscriptions) != 0 ||
		len(client.ProcessorSubscriptions) != 0 {
		t.Fatal("expected all subscriptions to be cleared on close")
	}
}

// TestClientDisconnectedSignalClosesOnDisconnect verifies that the channel
// returned by Disconnected() is closed the moment the underlying WebSocket
// connection drops, without requiring any caller to poll
// IsWebSocketConnected(). This is what lets RunStream handlers (events, alarms,
// command history, links, ...) react to a lost connection immediately instead
// of blocking forever (or until the next poll tick) on a signal channel that
// will never receive anything again.
func TestClientDisconnectedSignalClosesOnDisconnect(t *testing.T) {
	client, err := NewYamcsClient(
		"somepath",
		corehttp.GetNoTLSConfiguration(),
		&corehttp.NoCredentials{},
		OptionSetProtocol(false),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	disconnected := client.Disconnected()
	select {
	case <-disconnected:
		t.Fatal("expected disconnect signal to be open before any disconnect")
	default:
	}

	// Simulate a dropped connection the same way the WebSocket read loop does
	// internally (ws.WebSocketHandler.Listen defers ForceDisconnect on any
	// read error/close), without needing a real server.
	client.WebSocket.ForceDisconnect()

	select {
	case <-disconnected:
		// expected: signal closed immediately on disconnect
	case <-time.After(time.Second):
		t.Fatal("expected disconnect signal to be closed after ForceDisconnect")
	}

	// A second disconnect notification for the same (already-disconnected)
	// connection must not panic from a double-close.
	client.signalDisconnected()

	// After a fresh connect, a NEW open signal should be handed out so newly
	// started streams don't immediately think they're disconnected.
	client.resetDisconnectSignal()
	fresh := client.Disconnected()
	select {
	case <-fresh:
		t.Fatal("expected a fresh, open disconnect signal after reconnect")
	default:
	}
	if fresh == disconnected {
		t.Fatal("expected resetDisconnectSignal to hand out a new channel instance")
	}
}

func TestSubscribeCooldownAllowsFirstAttemptAndClearsOnSuccess(t *testing.T) {
	client := &YamcsClient{subscribeCooldowns: make(map[string]time.Time)}
	key := subscribeCooldownKey("parameters", "myinstance", "realtime")

	if err := client.checkSubscribeCooldown(key); err != nil {
		t.Fatalf("expected no cooldown before any failure, got: %v", err)
	}

	client.recordSubscribeOutcome(context.Background(), key, nil)
	if err := client.checkSubscribeCooldown(key); err != nil {
		t.Fatalf("expected no cooldown after a successful attempt, got: %v", err)
	}
}

func TestSubscribeCooldownBlocksImmediateRetryAfterFailure(t *testing.T) {
	client := &YamcsClient{subscribeCooldowns: make(map[string]time.Time)}
	key := subscribeCooldownKey("commandHistory", "myinstance", "realtime")

	client.recordSubscribeOutcome(context.Background(), key, errors.New("boom"))

	start := time.Now()
	err := client.checkSubscribeCooldown(key)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a cooldown error immediately after a recorded failure")
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("checkSubscribeCooldown took %v - expected an instant, non-blocking, in-memory-only check", elapsed)
	}

	// A different key must be unaffected.
	otherKey := subscribeCooldownKey("commandHistory", "otherinstance", "realtime")
	if err := client.checkSubscribeCooldown(otherKey); err != nil {
		t.Fatalf("expected no cooldown for an unrelated key, got: %v", err)
	}
}

func TestSubscribeCooldownIgnoresCancelledContextFailures(t *testing.T) {
	client := &YamcsClient{subscribeCooldowns: make(map[string]time.Time)}
	key := subscribeCooldownKey("events", "myinstance", "")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// A failure caused by the caller's own context being cancelled (e.g.
	// normal shutdown) must not start a cooldown - it isn't evidence the next
	// attempt will fail too.
	client.recordSubscribeOutcome(ctx, key, errors.New("context canceled"))
	if err := client.checkSubscribeCooldown(key); err != nil {
		t.Fatalf("expected no cooldown after a context-cancelled failure, got: %v", err)
	}
}

func TestSubscribeCooldownExpiresAfterConfiguredDuration(t *testing.T) {
	client := &YamcsClient{subscribeCooldowns: make(map[string]time.Time)}
	key := subscribeCooldownKey("links", "myinstance", "")

	// Simulate a failure that happened just outside the cooldown window.
	client.subscribeCooldowns[key] = time.Now().Add(-time.Millisecond)

	if err := client.checkSubscribeCooldown(key); err != nil {
		t.Fatalf("expected cooldown to have expired, got: %v", err)
	}
}

func TestClearAllSubscriptionsResetsCooldowns(t *testing.T) {
	client := &YamcsClient{subscribeCooldowns: make(map[string]time.Time)}
	key := subscribeCooldownKey("time", "myinstance", "realtime")
	client.recordSubscribeOutcome(context.Background(), key, errors.New("boom"))

	if err := client.checkSubscribeCooldown(key); err == nil {
		t.Fatal("expected a cooldown to be active before clearAllSubscriptions")
	}

	client.clearAllSubscriptions()

	if err := client.checkSubscribeCooldown(key); err != nil {
		t.Fatalf("expected clearAllSubscriptions to reset cooldowns for a fresh connection, got: %v", err)
	}
}
