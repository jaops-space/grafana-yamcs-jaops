//go:build integration
// +build integration

package integration_test

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	yamcsclient "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

func requireReachableYamcs(t *testing.T) {
	t.Helper()

	address := yamcsAddressForIntegration()
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		t.Skipf("unknown: Yamcs is unreachable at %s: %v", address, err)
		return
	}
	_ = conn.Close()
}

func newIntegrationClient(t *testing.T) *yamcsclient.YamcsClient {
	t.Helper()
	requireReachableYamcs(t)

	c, err := yamcsclient.NewYamcsClient(
		yamcsAddressForIntegration(),
		corehttp.GetNoTLSConfiguration(),
		&corehttp.NoCredentials{},
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return c
}

func integrationInstanceAndProcessor(t *testing.T, c *yamcsclient.YamcsClient) (string, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	instances, err := c.ListInstances(ctx)
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(instances) == 0 {
		t.Skip("unknown: no Yamcs instances returned")
	}

	instance := instances[0]
	instanceName := instance.GetName()
	if instanceName == "" {
		t.Skip("unknown: first Yamcs instance has empty name")
	}

	if len(instance.Processors) == 0 {
		t.Skipf("unknown: instance %s has no processors", instanceName)
	}
	processorName := instance.Processors[0].GetName()
	if processorName == "" {
		t.Skipf("unknown: instance %s first processor has empty name", instanceName)
	}

	return instanceName, processorName
}

func yamcsAddressForIntegration() string {
	address := os.Getenv("YAMCS_ADDRESS")
	if address == "" {
		address = "localhost:8090"
	}
	address = strings.TrimPrefix(address, "http://")
	address = strings.TrimPrefix(address, "https://")
	return strings.TrimSuffix(address, "/")
}

func connectWebSocket(t *testing.T, c *yamcsclient.YamcsClient) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	if err := c.EstablishWebSocketConnection(ctx); err != nil {
		t.Fatalf("establish websocket: %v", err)
	}
	t.Cleanup(func() {
		_ = c.CloseWebSocketConnection()
	})
	return ctx
}

func eventuallyStateHasCall(t *testing.T, ctx context.Context, c *yamcsclient.YamcsClient, callID int32, callType string) {
	t.Helper()

	state := requestState(t, ctx, c)
	call := findStateCall(state, callID)
	if call == nil {
		t.Fatalf("expected websocket state to include active %q call %d", callType, callID)
	}
	if call.GetType() != callType {
		t.Fatalf("expected websocket state call %d type %q, got %q", callID, callType, call.GetType())
	}
}

func waitThenAssertStateMissingCall(t *testing.T, ctx context.Context, c *yamcsclient.YamcsClient, callID int32) {
	t.Helper()

	time.Sleep(5 * time.Second)
	state := requestState(t, ctx, c)
	if findStateCall(state, callID) != nil {
		t.Fatalf("expected websocket state to remove call %d", callID)
	}
}

func requestState(t *testing.T, ctx context.Context, c *yamcsclient.YamcsClient) *api.State {
	t.Helper()

	stateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	state, err := c.WebSocketState(stateCtx)
	if err != nil {
		t.Fatalf("request websocket state: %v", err)
	}
	return state
}

func findStateCall(state *api.State, callID int32) *api.State_CallInfo {
	for _, call := range state.GetCalls() {
		if call.GetCall() == callID {
			return call
		}
	}
	return nil
}

func onlyKey[T any](t *testing.T, values map[int32]T) int32 {
	t.Helper()

	if len(values) != 1 {
		t.Fatalf("expected exactly one subscription registry entry, got %d", len(values))
	}
	for key := range values {
		return key
	}
	t.Fatalf("expected a subscription registry entry")
	return 0
}
