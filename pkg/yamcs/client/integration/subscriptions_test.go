//go:build integration
// +build integration

package integration_test

import "testing"

func TestIntegrationYamcs_ParameterSubscriptionLifecycle(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := connectWebSocket(t, client)

	instanceName, processorName := integrationInstanceAndProcessor(t, client)

	sub, err := client.CreateParameterSubscriptionByNames(ctx, instanceName, processorName)
	if err != nil {
		t.Fatalf("create parameter subscription: %v", err)
	}
	defer sub.Halt()

	before := len(client.ParameterSubscriptions)
	target := "/myproject/Battery1_Voltage"

	if err := sub.Add(target); err != nil {
		t.Fatalf("add parameter subscription: %v", err)
	}
	if !sub.Has(target) {
		t.Fatalf("expected active subscription snapshot to include %s", target)
	}

	if err := sub.Remove(target); err != nil {
		t.Fatalf("remove parameter subscription: %v", err)
	}
	if sub.Has(target) {
		t.Fatalf("expected active subscription snapshot to exclude %s", target)
	}

	after := len(client.ParameterSubscriptions)
	if after != before {
		t.Fatalf("unexpected subscription registry change before=%d after=%d", before, after)
	}
}

func TestIntegrationYamcs_SubscriptionSnapshotCardinality(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := connectWebSocket(t, client)

	instanceName, processorName := integrationInstanceAndProcessor(t, client)

	sub, err := client.CreateParameterSubscriptionByNames(ctx, instanceName, processorName)
	if err != nil {
		t.Fatalf("create parameter subscription: %v", err)
	}
	defer sub.Halt()

	params := []string{
		"/myproject/Battery1_Voltage",
		"/myproject/Battery2_Voltage",
		"/myproject/Detector_Temp",
	}

	if err := sub.Add(params...); err != nil {
		t.Fatalf("add parameters: %v", err)
	}

	snapshotCount := 0
	for _, p := range params {
		if sub.Has(p) {
			snapshotCount++
		}
	}
	if snapshotCount != len(params) {
		t.Fatalf("unexpected active subscription snapshot size: want=%d got=%d", len(params), snapshotCount)
	}

	if err := sub.Remove(params[0], params[1]); err != nil {
		t.Fatalf("remove parameters: %v", err)
	}

	remaining := 0
	for _, p := range params {
		if sub.Has(p) {
			remaining++
		}
	}
	if remaining != 1 {
		t.Fatalf("unexpected active subscription snapshot remaining size: want=1 got=%d", remaining)
	}
}

func TestIntegrationYamcs_WebSocketStateVerifiesAllSubscriptionKinds(t *testing.T) {
	// TODO: unskip when Yamcs state snapshots reliably exclude canceled subscription calls after Halt.
	t.Skip("skipping WebSocket state/halt verification until the upstream Yamcs state snapshot issue is fixed")

	client := newIntegrationClient(t)
	ctx := connectWebSocket(t, client)

	instanceName, processorName := integrationInstanceAndProcessor(t, client)

	parameterSub, err := client.CreateParameterSubscriptionByNames(ctx, instanceName, processorName, "/myproject/Battery1_Voltage")
	if err != nil {
		t.Fatalf("create parameter subscription: %v", err)
	}
	parameterCall := onlyKey(t, client.ParameterSubscriptions)
	eventSub, err := client.CreateEventSubscription(ctx, instanceName)
	if err != nil {
		t.Fatalf("create event subscription: %v", err)
	}
	eventCall := onlyKey(t, client.EventSubscriptions)
	commandSub, err := client.CreateCommandHistorySubscription(instanceName, processorName)
	if err != nil {
		t.Fatalf("create command history subscription: %v", err)
	}
	commandCall := onlyKey(t, client.CommandHistorySubscriptions)
	alarmSub, err := client.CreateAlarmSubscription(ctx, instanceName, processorName)
	if err != nil {
		t.Fatalf("create alarm subscription: %v", err)
	}
	alarmCall := onlyKey(t, client.AlarmSubscriptions)
	globalAlarmSub, err := client.CreateGlobalAlarmStatusSubscription(ctx, instanceName, processorName)
	if err != nil {
		t.Fatalf("create global alarm status subscription: %v", err)
	}
	globalAlarmCall := onlyKey(t, client.GlobalAlarmStatusSubscriptions)
	timeSub, err := client.CreateTimeSubscription(instanceName, processorName)
	if err != nil {
		t.Fatalf("create time subscription: %v", err)
	}
	timeCall := onlyKey(t, client.TimeSubscriptions)
	linkSub, err := client.CreateLinkSubscription(ctx, instanceName)
	if err != nil {
		t.Fatalf("create link subscription: %v", err)
	}
	linkCall := onlyKey(t, client.LinkSubscriptions)
	processorSub, err := client.CreateProcessorSubscriptionByNames(instanceName, processorName)
	if err != nil {
		t.Fatalf("create processor subscription: %v", err)
	}
	processorCall := onlyKey(t, client.ProcessorSubscriptions)

	expected := []struct {
		name     string
		callID   int32
		callType string
		halt     func()
	}{
		{name: "parameters", callID: parameterCall, callType: "parameters", halt: parameterSub.Halt},
		{name: "events", callID: eventCall, callType: "events", halt: eventSub.Halt},
		{name: "commands", callID: commandCall, callType: "commands", halt: commandSub.Halt},
		{name: "alarms", callID: alarmCall, callType: "alarms", halt: alarmSub.Halt},
		{name: "global alarm status", callID: globalAlarmCall, callType: "global-alarm-status", halt: globalAlarmSub.Halt},
		{name: "time", callID: timeCall, callType: "time", halt: timeSub.Halt},
		{name: "links", callID: linkCall, callType: "links", halt: linkSub.Halt},
		{name: "processors", callID: processorCall, callType: "processors", halt: processorSub.Halt},
	}

	for _, item := range expected {
		t.Run(item.name+" appears in state", func(t *testing.T) {
			eventuallyStateHasCall(t, ctx, client, item.callID, item.callType)
		})
	}

	for i := len(expected) - 1; i >= 0; i-- {
		item := expected[i]
		t.Run(item.name+" disappears from state after halt", func(t *testing.T) {
			item.halt()
			waitThenAssertStateMissingCall(t, ctx, client, item.callID)
		})
	}
}
