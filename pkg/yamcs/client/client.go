package client

import (
	"fmt"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/auth"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/manager/http"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/manager/ws"
)

// YamcsClientOption defines a function type for configuring a YamcsClient.
type YamcsClientOption func(p *YamcsClient)

// YamcsClient represents a client for connecting to a Yamcs server.
type YamcsClient struct {

	// The address of the Yamcs server in the format 'hostname:port'
	ServerAddress string

	// TLS configuration for secure communication
	TLSConfig auth.TLS

	// User authentication credentials (username/password or bearer token)
	Credentials auth.AccountCredentials

	// Optionally override the default user agent string
	UserAgent string

	// Whether the client should keep the session alive automatically (default: true)
	KeepAlive bool

	// Whether the client should use Protobuf protocol (default: true)
	UseProtobuf bool

	// The context associated with the client connection
	HTTP *http.HTTPManager

	// WebSocket handler for managing real-time data streams
	WebSocket *ws.WebSocketHandler

	// Various subscriptions for data streams
	ParameterSubscriptions         map[int]*ParameterSubscription
	CommandHistorySubscriptions    map[int]*CommandHistorySubscription
	EventSubscriptions             map[int]*EventSubscription
	AlarmSubscriptions             map[int]*AlarmSubscription
	GlobalAlarmStatusSubscriptions map[int]*GlobalStatusSubscription

	// Sample Point Count for Sample endpoints
	SamplePointCount *types.Optional[int]
}

// NewYamcsClient constructs a new YamcsClient.
func NewYamcsClient(
	address string,
	tlsConfig auth.TLS,
	credentials auth.AccountCredentials,
	options ...YamcsClientOption,
) (*YamcsClient, error) {

	// Initialize the YamcsClient with default values
	client := &YamcsClient{
		ServerAddress:                  address,
		TLSConfig:                      tlsConfig,
		Credentials:                    credentials,
		UseProtobuf:                    true,
		KeepAlive:                      true,
		ParameterSubscriptions:         make(map[int]*ParameterSubscription),
		CommandHistorySubscriptions:    make(map[int]*CommandHistorySubscription),
		EventSubscriptions:             make(map[int]*EventSubscription),
		AlarmSubscriptions:             make(map[int]*AlarmSubscription),
		GlobalAlarmStatusSubscriptions: make(map[int]*GlobalStatusSubscription),
		SamplePointCount:               types.OptionalOfNil[int](),
	}

	// WebSocket URL based on whether TLS is enabled
	wsURL := fmt.Sprintf("%s://%s/api/websocket", getProtocolPrefix(tlsConfig.Enabled), address)

	// Apply any custom client options
	for _, option := range options {
		option(client)
	}

	// Create a new context for the client
	httpManager, err := http.NewHTTPManager(address, tlsConfig, credentials, client.UserAgent, client.KeepAlive, client.UseProtobuf)
	if err != nil {
		return nil, err
	}
	client.HTTP = httpManager

	// Initialize WebSocket handler
	client.WebSocket = ws.NewWebSocketHandler(wsURL, client.UseProtobuf)

	client.WebSocket.AddListener(ws.ParameterListenerID, client.HandleParameterMessage)
	client.WebSocket.AddListener(ws.EventListenerID, client.HandleEventMessage)
	client.WebSocket.AddListener(ws.AlarmListenerID, client.HandleAlarmMessage)
	client.WebSocket.AddListener(ws.GlobalStatusListenerID, client.HandleGlobalStatusMessage)
	client.WebSocket.AddListener(ws.CommandHistoryLisernerID, client.HandleCommandMessage)

	// Handle WebSocket disconnections
	client.WebSocket.SetDisconnectHandler(func() {
		client.clearAllSubscriptions()
	})

	return client, nil
}

func (client *YamcsClient) EstablishWebSocketConnection() error {
	if client.IsWebSocketConnected() {
		return nil
	}
	err := client.WebSocket.Connect()
	if err == nil {
		client.clearAllSubscriptions()
		go client.WebSocket.Listen()
	}
	return err
}

func (client *YamcsClient) CloseWebSocketConnection() error {
	return client.WebSocket.Disconnect()
}

func (client *YamcsClient) IsWebSocketConnected() bool {
	return client.WebSocket.IsConnected()
}

// OptionSetUserAgent allows overriding the default User-Agent.
func OptionSetUserAgent(userAgent string) YamcsClientOption {
	return func(client *YamcsClient) {
		client.UserAgent = userAgent
	}
}

// OptionSetKeepAlive allows enabling or disabling session keep-alive.
func OptionSetKeepAlive(keepAlive bool) YamcsClientOption {
	return func(client *YamcsClient) {
		client.KeepAlive = keepAlive
	}
}

// OptionSetProtocol allows choosing between Protobuf or JSON protocols.
func OptionSetProtocol(useProtobuf bool) YamcsClientOption {
	return func(client *YamcsClient) {
		client.UseProtobuf = useProtobuf
	}
}

// getProtocolPrefix returns the appropriate protocol prefix based on TLS configuration.
func getProtocolPrefix(isTLS bool) string {
	if isTLS {
		return "wss"
	}
	return "ws"
}

// clearAllSubscriptions clears all subscriptions for the client.
func (client *YamcsClient) clearAllSubscriptions() {
	// Clear subscriptions
	client.ParameterSubscriptions = make(map[int]*ParameterSubscription)
	client.EventSubscriptions = make(map[int]*EventSubscription)
	client.AlarmSubscriptions = make(map[int]*AlarmSubscription)
	client.GlobalAlarmStatusSubscriptions = make(map[int]*GlobalStatusSubscription)
}
