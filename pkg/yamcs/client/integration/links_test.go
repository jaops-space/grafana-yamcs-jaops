//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
)

func TestIntegrationYamcs_LinkSubscriptionReceivesFullSnapshots(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := connectWebSocket(t, client)
	instanceName, _ := integrationInstanceAndProcessor(t, client)

	listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
	defer listCancel()
	initialLinks, err := client.ListLinks(listCtx, instanceName)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(initialLinks) == 0 {
		t.Skipf("unknown: Yamcs instance %s has no links", instanceName)
	}

	events := make(chan *links.LinkEvent, 8)
	sub, err := client.CreateLinkSubscription(ctx, instanceName, func(event *links.LinkEvent) error {
		events <- event
		return nil
	})
	if err != nil {
		t.Fatalf("create link subscription: %v", err)
	}
	defer sub.Halt()

	first := waitForLinkSnapshot(t, events, len(initialLinks), 20*time.Second)
	second := waitForLinkSnapshot(t, events, len(initialLinks), 5*time.Second)

	assertLinkSnapshotContainsListedLinks(t, first, initialLinks)
	assertLinkSnapshotContainsListedLinks(t, second, initialLinks)
}

func TestIntegrationYamcs_LinkSubscriptionReflectsDisableEnable(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := connectWebSocket(t, client)
	instanceName, _ := integrationInstanceAndProcessor(t, client)

	listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
	defer listCancel()
	initialLinks, err := client.ListLinks(listCtx, instanceName)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	target := chooseIntegrationLink(initialLinks)
	if target == nil {
		t.Skipf("unknown: Yamcs instance %s has no links", instanceName)
	}

	events := make(chan *links.LinkEvent, 16)
	sub, err := client.CreateLinkSubscription(ctx, instanceName, func(event *links.LinkEvent) error {
		events <- event
		return nil
	})
	if err != nil {
		t.Fatalf("create link subscription: %v", err)
	}
	defer sub.Halt()

	originalDisabled := target.GetDisabled()
	targetDisabled := !originalDisabled
	t.Cleanup(func() {
		restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer restoreCancel()
		if originalDisabled {
			_, _ = client.DisableLink(restoreCtx, instanceName, target.GetName())
		} else {
			_, _ = client.EnableLink(restoreCtx, instanceName, target.GetName())
		}
	})

	actionCtx, actionCancel := context.WithTimeout(ctx, 10*time.Second)
	defer actionCancel()
	if targetDisabled {
		if _, err := client.DisableLink(actionCtx, instanceName, target.GetName()); err != nil {
			t.Skipf("unknown: link %s could not be disabled in this Yamcs instance: %v", target.GetName(), err)
		}
	} else {
		if _, err := client.EnableLink(actionCtx, instanceName, target.GetName()); err != nil {
			t.Skipf("unknown: link %s could not be enabled in this Yamcs instance: %v", target.GetName(), err)
		}
	}

	updated := waitForLinkState(t, events, target.GetName(), targetDisabled, 20*time.Second)
	if updated.GetInstance() != instanceName {
		t.Fatalf("expected updated link instance %q, got %q", instanceName, updated.GetInstance())
	}
}

func TestIntegrationYamcs_EndpointLinksStreamFanoutBuildsColumnarFrame(t *testing.T) {
	client := newIntegrationClient(t)
	instanceName, processorName := integrationInstanceAndProcessor(t, client)

	streamCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mux, err := source.NewMultiplexerWithContext(streamCtx, &config.YamcsPluginConfiguration{
		Hosts: map[string]*config.YamcsHostConfiguration{
			"quickstart": {
				ID:   "quickstart",
				Path: yamcsAddressForIntegration(),
			},
		},
		Endpoints: map[string]*config.YamcsEndpointConfiguration{
			"quickstart_realtime": {
				ID:        "quickstart_realtime",
				Host:      "quickstart",
				Instance:  instanceName,
				Processor: processorName,
			},
		},
	}, &config.YamcsSecureConfiguration{Hosts: map[string]*config.YamcsSecureHost{}})
	if err != nil {
		t.Fatalf("create multiplexer: %v", err)
	}

	hostErrors, endpointErrors := mux.ConnectSync(streamCtx, false)
	if len(hostErrors) != 0 || len(endpointErrors) != 0 {
		t.Fatalf("connect multiplexer hostErrors=%v endpointErrors=%v", hostErrors, endpointErrors)
	}
	t.Cleanup(func() {
		if host := mux.Hosts["quickstart"]; host != nil && host.GetClient() != nil {
			_ = host.GetClient().CloseWebSocketConnection()
		}
	})

	endpoint, err := mux.GetEndpoint("quickstart_realtime")
	if err != nil {
		t.Fatalf("get endpoint: %v", err)
	}

	const path = "quickstart_realtime/links"
	if err := endpoint.RequestLinksStream(streamCtx, path); err != nil {
		t.Fatalf("request links stream: %v", err)
	}
	defer endpoint.WithdrawLinksStreamRequest(path)

	signal := endpoint.GetLinksSignal(path)
	if signal == nil {
		t.Fatalf("expected links stream signal to be registered")
	}

	event := waitForEndpointLinkEvent(t, signal, 20*time.Second)
	frame, err := tools.ConvertLinksToFrame(event.GetLinks())
	if err != nil {
		t.Fatalf("convert links to frame: %v", err)
	}

	if frame.Name != "links" {
		t.Fatalf("expected frame name links, got %q", frame.Name)
	}
	if frame.Rows() != len(event.GetLinks()) {
		t.Fatalf("expected frame row count %d, got %d", len(event.GetLinks()), frame.Rows())
	}

	for _, fieldName := range []string{"instance", "name", "type", "disabled", "status", "dataInCount", "dataOutCount", "detailedStatus", "parentName", "actions", "extra"} {
		field, _ := frame.FieldByName(fieldName)
		if field == nil {
			t.Fatalf("expected columnar links frame to include field %q", fieldName)
		}
	}
}

func waitForLinkSnapshot(t *testing.T, events <-chan *links.LinkEvent, minLinks int, timeout time.Duration) *links.LinkEvent {
	t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case event := <-events:
			if len(event.GetLinks()) >= minLinks {
				return event
			}
		case <-deadline:
			t.Fatalf("timed out waiting for links snapshot with at least %d links", minLinks)
		}
	}
}

func waitForLinkState(t *testing.T, events <-chan *links.LinkEvent, linkName string, disabled bool, timeout time.Duration) *links.LinkInfo {
	t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case event := <-events:
			for _, link := range event.GetLinks() {
				if link.GetName() == linkName && link.GetDisabled() == disabled {
					return link
				}
			}
		case <-deadline:
			t.Fatalf("timed out waiting for link %s disabled=%t update", linkName, disabled)
		}
	}
}

func waitForEndpointLinkEvent(t *testing.T, signal <-chan *links.LinkEvent, timeout time.Duration) *links.LinkEvent {
	t.Helper()

	select {
	case event := <-signal:
		if len(event.GetLinks()) == 0 {
			t.Fatalf("expected endpoint link event to include links")
		}
		return event
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for endpoint links stream event")
		return nil
	}
}

func assertLinkSnapshotContainsListedLinks(t *testing.T, event *links.LinkEvent, listed []*links.LinkInfo) {
	t.Helper()

	seen := make(map[string]*links.LinkInfo, len(event.GetLinks()))
	for _, link := range event.GetLinks() {
		if link.GetName() == "" {
			t.Fatalf("expected subscription link update to include link name")
		}
		if link.GetInstance() == "" {
			t.Fatalf("expected subscription link %s to include instance", link.GetName())
		}
		seen[link.GetName()] = link
	}

	for _, link := range listed {
		if seen[link.GetName()] == nil {
			t.Fatalf("expected subscription snapshot to include listed link %q", link.GetName())
		}
	}
}

func chooseIntegrationLink(listed []*links.LinkInfo) *links.LinkInfo {
	preferredNames := []string{"tc_sim", "tm_realtime", "tm2_realtime", "tm_dump", "TSE"}
	for _, name := range preferredNames {
		for _, link := range listed {
			if link.GetName() == name {
				return link
			}
		}
	}
	if len(listed) == 0 {
		return nil
	}
	return listed[0]
}
