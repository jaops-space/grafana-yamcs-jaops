package source

import (
	"context"
	"testing"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// newUnreachableTestHost builds a YamcsHost pointed at an address that
// refuses connections immediately (nothing listens on 127.0.0.1:1, a
// privileged/unused port), so dial attempts fail fast without a long
// timeout, keeping these tests quick.
func newUnreachableTestHost(t *testing.T) *YamcsHost {
	t.Helper()

	cli, err := client.NewYamcsClient("127.0.0.1:1", corehttp.GetNoTLSConfiguration(), &corehttp.NoCredentials{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return &YamcsHost{
		Client:        cli,
		Instances:     map[string]*YamcsHostInstance{},
		Configuration: &config.YamcsHostConfiguration{ID: "test-host"},
	}
}

// TestHostRequestConnectIsNonBlockingWithoutManager verifies that calling
// RequestConnect() before a connection manager has been started (e.g. a
// throwaway health-check Multiplexer, which never starts one) is a harmless,
// instantaneous no-op rather than a block or a panic.
func TestHostRequestConnectIsNonBlockingWithoutManager(t *testing.T) {
	host := newUnreachableTestHost(t)

	done := make(chan struct{})
	go func() {
		for range 10 {
			host.RequestConnect()
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RequestConnect blocked with no manager running")
	}
}

// TestHostRequestConnectCoalescesWithRunningManager verifies that many
// concurrent RequestConnect() calls against a host whose manager is actively
// running never block the caller, even while the manager itself is busy
// retrying a failing dial - this is the core guarantee that lets
// SubscribeStream/RunStream "ask" for a reconnect without ever performing (or
// waiting on) the dial themselves.
func TestHostRequestConnectCoalescesWithRunningManager(t *testing.T) {
	host := newUnreachableTestHost(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host.startConnectionManager(ctx, "test-host", func(context.Context, string, *YamcsHost) {})
	defer host.stopConnectionManager()

	done := make(chan struct{})
	go func() {
		for range 50 {
			host.RequestConnect()
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RequestConnect blocked while manager was running/retrying")
	}

	if host.IsConnected() {
		t.Fatal("expected host to remain disconnected against an unreachable address")
	}
}

// TestEndpointEnsureReadyFailsFastAndRequestsConnectWithoutDialing verifies
// that EnsureReady() never blocks on a network dial itself: against a
// disconnected host it returns immediately with an error and leaves a
// connect request queued for whatever manager is (or later becomes) running,
// rather than attempting to connect inline.
func TestEndpointEnsureReadyFailsFastAndRequestsConnectWithoutDialing(t *testing.T) {
	host := newUnreachableTestHost(t)
	// Simulate a manager having been started (so RequestConnect has somewhere
	// to send to) without actually running one, so we can inspect whether a
	// request was queued.
	host.connectRequest = make(chan struct{}, 1)

	endpoint := &YamcsEndpoint{
		Host: host,
		Configuration: &config.YamcsEndpointConfiguration{
			ID:       "ep",
			Instance: "myinstance",
		},
	}

	start := time.Now()
	err := endpoint.EnsureReady()
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected EnsureReady to fail while host is disconnected")
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("EnsureReady took %v - expected an instant, non-blocking, in-memory-only check", elapsed)
	}

	// A connect request should now be pending for this host (buffered size 1),
	// confirming EnsureReady asked for a connect rather than performing one.
	select {
	case <-host.connectRequest:
		// Expected: EnsureReady queued exactly one pending request.
	default:
		t.Fatal("expected EnsureReady to have queued a connect request")
	}
}
