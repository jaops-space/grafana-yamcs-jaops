//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	yamcsclient "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

func TestIntegrationYamcs_ListInstancesAndFetchOne(t *testing.T) {
	client := newIntegrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	instances, err := client.ListInstances(ctx)
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(instances) == 0 {
		t.Fatalf("expected at least one Yamcs instance")
	}

	got, err := client.GetInstanceByName(ctx, instances[0].GetName())
	if err != nil {
		t.Fatalf("get instance by name: %v", err)
	}
	if got.GetName() == "" {
		t.Fatalf("expected non-empty instance name")
	}
}

func TestIntegrationYamcs_WebSocketConnectDisconnect(t *testing.T) {
	client := newIntegrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := client.EstablishWebSocketConnection(ctx); err != nil {
		t.Fatalf("establish websocket: %v", err)
	}

	if !client.IsWebSocketConnected() {
		t.Fatalf("expected websocket to be connected")
	}

	if err := client.CloseWebSocketConnection(); err != nil {
		t.Fatalf("close websocket: %v", err)
	}

	if client.IsWebSocketConnected() {
		t.Fatalf("expected websocket to be disconnected")
	}
}

func TestIntegrationYamcs_CommandMetadataLookup(t *testing.T) {
	client := newIntegrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	instances, err := client.ListInstances(ctx)
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(instances) == 0 {
		t.Fatalf("expected at least one Yamcs instance")
	}

	instanceName := instances[0].GetName()
	commandInfos, err := client.ListCommandInfos(ctx, instanceName).Next()
	if err != nil {
		t.Fatalf("list command infos: %v", err)
	}
	if len(commandInfos) == 0 {
		t.Skip("no commands found in Yamcs quickstart instance")
	}

	cmdName := commandInfos[0].GetQualifiedName()
	if cmdName == "" {
		t.Fatalf("expected non-empty qualified command name")
	}

	info, err := client.GetCommandInfo(ctx, instanceName, cmdName)
	if err != nil {
		t.Fatalf("get command info for %s: %v", cmdName, err)
	}
	if info.GetQualifiedName() == "" {
		t.Fatalf("expected command info qualified name")
	}
}

func TestIntegrationYamcs_WebSocketReconnectClearsSubscriptionRegistry(t *testing.T) {
	client := newIntegrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.EstablishWebSocketConnection(ctx); err != nil {
		t.Fatalf("establish websocket: %v", err)
	}

	instanceName, processorName := integrationInstanceAndProcessor(t, client)

	sub, err := client.CreateParameterSubscriptionByNames(ctx, instanceName, processorName, "/myproject/Battery1_Voltage")
	if err != nil {
		t.Fatalf("create parameter subscription: %v", err)
	}

	if len(client.ParameterSubscriptions) == 0 {
		t.Fatalf("expected parameter subscriptions registry to contain entries")
	}

	if err := client.CloseWebSocketConnection(); err != nil {
		t.Fatalf("close websocket: %v", err)
	}

	if got := len(client.ParameterSubscriptions); got != 0 {
		t.Fatalf("expected parameter subscriptions to be cleared on disconnect, got %d", got)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("subscription halt should not panic after disconnect: %v", r)
		}
	}()
	sub.Halt()
}

func TestIntegrationYamcs_RepeatedConnectDisconnectStability(t *testing.T) {
	client := newIntegrationClient(t)

	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err := client.EstablishWebSocketConnection(ctx)
		cancel()
		if err != nil {
			t.Fatalf("connect #%d failed: %v", i+1, err)
		}
		if !client.IsWebSocketConnected() {
			t.Fatalf("expected connected after connect #%d", i+1)
		}

		if err := client.CloseWebSocketConnection(); err != nil {
			t.Fatalf("disconnect #%d failed: %v", i+1, err)
		}

		if client.IsWebSocketConnected() {
			t.Fatalf("expected disconnected after disconnect #%d", i+1)
		}
	}

	if client.IsWebSocketConnected() {
		t.Fatalf("expected websocket to be disconnected after loop")
	}

	if len(client.ParameterSubscriptions) != 0 {
		t.Fatalf("expected parameter subscriptions to stay empty after reconnect loop: %d", len(client.ParameterSubscriptions))
	}
}

func TestIntegrationYamcs_UnreachableAddressReportsUnknown(t *testing.T) {
	address := "127.0.0.1:65535"
	c, err := yamcsclient.NewYamcsClient(address, corehttp.GetNoTLSConfiguration(), &corehttp.NoCredentials{})
	if err != nil {
		t.Fatalf("create unreachable client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = c.EstablishWebSocketConnection(ctx)
	if err == nil {
		_ = c.CloseWebSocketConnection()
		t.Skipf("unknown: unreachable check could not be verified because %s unexpectedly accepted websocket", address)
		return
	}

	t.Logf("unknown: unreachable Yamcs behaves as expected at %s (%v)", address, err)
	if c.IsWebSocketConnected() {
		t.Fatalf("expected websocket to remain disconnected for unreachable Yamcs")
	}
}
