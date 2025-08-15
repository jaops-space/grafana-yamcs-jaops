package ws

type ListenerID string

const (
	ParameterListenerID      ListenerID = "PARAMETER_LISTENER"
	EventListenerID          ListenerID = "EVENT_LISTENER"
	AlarmListenerID          ListenerID = "ALARM_LISTENER"
	GlobalStatusListenerID   ListenerID = "GLOBAL_STATUS_LISTENER"
	CommandHistoryLisernerID ListenerID = "COMMAND_HISTORY_LISTENER"
	TimeListenerID           ListenerID = "TIME_LISTENER"
)
