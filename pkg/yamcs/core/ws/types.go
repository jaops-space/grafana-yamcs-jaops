package ws

type ListenerID string

const (
	ParameterListenerID      ListenerID = "parameters"
	EventListenerID          ListenerID = "events"
	AlarmListenerID          ListenerID = "alarms"
	GlobalStatusListenerID   ListenerID = "global-alarm-status"
	CommandHistoryLisernerID ListenerID = "commands"
	TimeListenerID           ListenerID = "time"
	LinksListenerID          ListenerID = "links"
	ProcessorListenerID      ListenerID = "processors"
)
