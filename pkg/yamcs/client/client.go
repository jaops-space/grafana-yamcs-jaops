package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/ws"
)

// YamcsClientOption defines a function type for configuring a YamcsClient.
type YamcsClientOption func(p *YamcsClient)

// YamcsClient represents a client for connecting to a Yamcs server.
type YamcsClient struct {

	// The address of the Yamcs server in the format 'hostname:port'
	ServerAddress string

	// TLS configuration for secure communication
	TLSConfig corehttp.TLS

	// User authentication credentials (username/password or bearer token)
	Credentials corehttp.Credentials

	// Optionally override the default user agent string
	UserAgent string

	// Whether the client should keep the session alive automatically (default: true)
	KeepAlive bool

	// Whether the client should use Protobuf protocol (default: true)
	UseProtobuf bool

	// The context associated with the client connection
	HTTP *corehttp.HTTPManager

	// Pre-built HTTP client (e.g. from Grafana SDK) for connection reuse
	HTTPClient *http.Client

	// WebSocket handler for managing real-time data streams
	WebSocket *ws.WebSocketHandler

	// Various subscriptions for data streams
	ParameterSubscriptions         map[int32]*ParameterSubscription
	CommandHistorySubscriptions    map[int32]*CommandHistorySubscription
	EventSubscriptions             map[int32]*EventSubscription
	AlarmSubscriptions             map[int32]*AlarmSubscription
	GlobalAlarmStatusSubscriptions map[int32]*GlobalStatusSubscription
	TimeSubscriptions              map[int32]*TimeSubscription
	LinkSubscriptions              map[int32]*LinkSubscription
	ProcessorSubscriptions         map[int32]*ProcessorSubscription

	// Sample Point Count for Sample endpoints
	SamplePointCount *types.Optional[int]
}

// NewYamcsClient constructs a new YamcsClient.
func NewYamcsClient(
	address string,
	tlsConfig corehttp.TLS,
	credentials corehttp.Credentials,
	options ...YamcsClientOption,
) (*YamcsClient, error) {

	// Initialize the YamcsClient with default values
	client := &YamcsClient{
		ServerAddress:                  address,
		TLSConfig:                      tlsConfig,
		Credentials:                    credentials,
		UseProtobuf:                    true,
		KeepAlive:                      true,
		ParameterSubscriptions:         make(map[int32]*ParameterSubscription),
		CommandHistorySubscriptions:    make(map[int32]*CommandHistorySubscription),
		EventSubscriptions:             make(map[int32]*EventSubscription),
		AlarmSubscriptions:             make(map[int32]*AlarmSubscription),
		GlobalAlarmStatusSubscriptions: make(map[int32]*GlobalStatusSubscription),
		TimeSubscriptions:              make(map[int32]*TimeSubscription),
		LinkSubscriptions:              make(map[int32]*LinkSubscription),
		ProcessorSubscriptions:         make(map[int32]*ProcessorSubscription),
		SamplePointCount:               types.OptionalOfNil[int](),
	}

	// WebSocket URL based on whether TLS is enabled
	wsURL := fmt.Sprintf("%s://%s/api/websocket", getProtocolPrefix(tlsConfig.Enabled), address)

	// Apply any custom client options
	for _, option := range options {
		option(client)
	}

	// Create a new context for the client
	httpManager, err := corehttp.NewHTTPManager(address, tlsConfig, credentials, client.UserAgent, client.KeepAlive, client.UseProtobuf, client.HTTPClient)
	if err != nil {
		return nil, err
	}
	client.HTTP = httpManager

	// Initialize WebSocket handler
	client.WebSocket = ws.NewWebSocketHandler(wsURL, client.UseProtobuf)
	client.WebSocket.Credentials = credentials

	client.WebSocket.AddListener(ws.ParameterListenerID, client.HandleParameterMessage)
	client.WebSocket.AddListener(ws.EventListenerID, client.HandleEventMessage)
	client.WebSocket.AddListener(ws.AlarmListenerID, client.HandleAlarmMessage)
	client.WebSocket.AddListener(ws.GlobalStatusListenerID, client.HandleGlobalStatusMessage)
	client.WebSocket.AddListener(ws.CommandHistoryLisernerID, client.HandleCommandMessage)
	client.WebSocket.AddListener(ws.TimeListenerID, client.HandleTimeMessage)
	client.WebSocket.AddListener(ws.LinksListenerID, client.HandleLinkMessage)
	client.WebSocket.AddListener(ws.ProcessorListenerID, client.HandleProcessorMessage)

	// Handle WebSocket disconnections
	client.WebSocket.SetDisconnectHandler(func() {
		client.clearAllSubscriptions()
	})

	return client, nil
}

func (client *YamcsClient) EstablishWebSocketConnection(ctx context.Context) error {
	if client.IsWebSocketConnected() {
		return nil
	}
	err := client.WebSocket.Connect(ctx)
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

// OptionSetHTTPClient allows injecting a pre-built *http.Client (e.g. from the
// Grafana plugin SDK) so that connections are reused across queries.
func OptionSetHTTPClient(httpClient *http.Client) YamcsClientOption {
	return func(client *YamcsClient) {
		client.HTTPClient = httpClient
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
	client.ParameterSubscriptions = make(map[int32]*ParameterSubscription)
	client.EventSubscriptions = make(map[int32]*EventSubscription)
	client.CommandHistorySubscriptions = make(map[int32]*CommandHistorySubscription)
	client.AlarmSubscriptions = make(map[int32]*AlarmSubscription)
	client.GlobalAlarmStatusSubscriptions = make(map[int32]*GlobalStatusSubscription)
	client.TimeSubscriptions = make(map[int32]*TimeSubscription)
	client.LinkSubscriptions = make(map[int32]*LinkSubscription)
	client.ProcessorSubscriptions = make(map[int32]*ProcessorSubscription)
}
