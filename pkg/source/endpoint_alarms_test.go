package source

import (
	"testing"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
)

func TestAlarmsListenerReplacesCachedAlarm(t *testing.T) {
	ep := testAlarmEndpoint()
	listener := ep.getAlarmsListener()

	id := testAlarmID()
	seqNum := uint32(7)
	triggered := alarms.AlarmNotificationType_TRIGGERED
	acknowledgedNotification := alarms.AlarmNotificationType_ACKNOWLEDGED
	acknowledged := true
	user := "operator"
	message := "checked"

	alarmID := "/YSS/SIMULATOR/BatteryVoltage1/7"
	ep.AlarmCache[alarmID] = &alarms.AlarmData{
		Id:               id,
		SeqNum:           &seqNum,
		NotificationType: &triggered,
	}

	update := &alarms.AlarmData{
		Id:               id,
		SeqNum:           &seqNum,
		NotificationType: &acknowledgedNotification,
		Acknowledged:     &acknowledged,
		AcknowledgeInfo: &alarms.AcknowledgeInfo{
			AcknowledgedBy:     &user,
			AcknowledgeMessage: &message,
		},
	}

	err := listener(update)
	if err != nil {
		t.Fatal(err)
	}

	if ep.AlarmCache[alarmID] != update {
		t.Fatal("expected cache to contain the latest alarm update")
	}
	if !ep.AlarmCache[alarmID].GetAcknowledged() {
		t.Fatal("expected acknowledged state from latest alarm update")
	}
	assertAlarmNotified(t, ep)
}

func TestAlarmsListenerClearRemovesCachedAlarm(t *testing.T) {
	ep := testAlarmEndpoint()
	listener := ep.getAlarmsListener()

	id := testAlarmID()
	seqNum := uint32(7)
	triggered := alarms.AlarmNotificationType_TRIGGERED
	cleared := alarms.AlarmNotificationType_CLEARED
	alarmID := "/YSS/SIMULATOR/BatteryVoltage1/7"

	ep.AlarmCache[alarmID] = &alarms.AlarmData{
		Id:               id,
		SeqNum:           &seqNum,
		NotificationType: &triggered,
	}

	err := listener(&alarms.AlarmData{
		Id:               id,
		SeqNum:           &seqNum,
		NotificationType: &cleared,
		ClearInfo:        &alarms.ClearInfo{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := ep.AlarmCache[alarmID]; ok {
		t.Fatal("expected cleared alarm to be removed from cache")
	}
	assertAlarmNotified(t, ep)
}

func TestAlarmsListenerProcessOKAcknowledgedRemovesCachedAlarm(t *testing.T) {
	ep := testAlarmEndpoint()
	listener := ep.getAlarmsListener()

	id := testAlarmID()
	seqNum := uint32(7)
	triggeredNotification := alarms.AlarmNotificationType_TRIGGERED
	rtnNotification := alarms.AlarmNotificationType_RTN
	processOK := true
	triggered := false
	acknowledged := true
	alarmID := "/YSS/SIMULATOR/BatteryVoltage1/7"

	ep.AlarmCache[alarmID] = &alarms.AlarmData{
		Id:               id,
		SeqNum:           &seqNum,
		NotificationType: &triggeredNotification,
	}

	err := listener(&alarms.AlarmData{
		Id:               id,
		SeqNum:           &seqNum,
		NotificationType: &rtnNotification,
		ProcessOK:        &processOK,
		Triggered:        &triggered,
		Acknowledged:     &acknowledged,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := ep.AlarmCache[alarmID]; ok {
		t.Fatal("expected returned-to-normal acknowledged alarm to be removed from cache")
	}
	assertAlarmNotified(t, ep)
}

func testAlarmEndpoint() *YamcsEndpoint {
	return &YamcsEndpoint{
		Alarms:       map[string][]*alarms.AlarmData{"test": {}},
		AlarmSignals: map[string]chan struct{}{"test": make(chan struct{}, 1)},
		AlarmCache:   map[string]*alarms.AlarmData{},
	}
}

func testAlarmID() *protobuf.NamedObjectId {
	namespace := "/YSS/SIMULATOR"
	name := "BatteryVoltage1"
	return &protobuf.NamedObjectId{
		Namespace: &namespace,
		Name:      &name,
	}
}

func assertAlarmNotified(t *testing.T, ep *YamcsEndpoint) {
	t.Helper()
	select {
	case <-ep.AlarmSignals["test"]:
	default:
		t.Fatal("expected alarm stream notification")
	}
}
