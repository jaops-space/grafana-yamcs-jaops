package client

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
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

	// subsMu guards every read, write, range and delete of the eight
	// subscription maps below. They are mutated concurrently from at least
	// three places: the WebSocket read loop (Listen(), dispatching incoming
	// frames and looking subscriptions up by call ID), any number of
	// concurrent RunXStream goroutines (creating/finding/halting
	// subscriptions for their own instance/processor), and
	// clearAllSubscriptions (wiping all of them on connect/reconnect/close).
	// Without this lock, opening a second instance's dashboard while another
	// is already streaming reliably races the read loop against a
	// subscribing goroutine, which for plain Go maps is a fatal
	// "concurrent map read/iteration and map write" crash, not just a data
	// race - this crashes the whole backend process (all instances, not just
	// the new one), which is the root cause of dashboards going silent with
	// timeouts until Grafana restarts the backend.
	subsMu sync.RWMutex

	// Various subscriptions for data streams. Access only through subsMu.
	ParameterSubscriptions         map[int32]*ParameterSubscription
	CommandHistorySubscriptions    map[int32]*CommandHistorySubscription
	EventSubscriptions             map[int32]*EventSubscription
	AlarmSubscriptions             map[int32]*AlarmSubscription
	GlobalAlarmStatusSubscriptions map[int32]*GlobalStatusSubscription
	TimeSubscriptions              map[int32]*TimeSubscription
	LinkSubscriptions              map[int32]*LinkSubscription
	ProcessorSubscriptions         map[int32]*ProcessorSubscription

	// disconnectMu guards disconnectSignal so it can be safely read, closed and
	// replaced concurrently from the WebSocket's disconnect handler (writer)
	// and from any number of RunStream goroutines (readers).
	disconnectMu sync.Mutex

	// disconnectSignal is closed exactly once whenever the underlying
	// WebSocket connection is lost, and replaced with a fresh, open channel on
	// every successful (re)connect. Streams should select on Disconnected()
	// instead of periodically polling IsWebSocketConnected(), so a dropped
	// connection is reacted to immediately instead of up to a polling
	// interval later.
	disconnectSignal chan struct{}
}

// NewYamcsClient constructs a new YamcsClient.
func NewYamcsClient(
	address string,
	tlsConfig corehttp.TLS,
	credentials corehttp.Credentials,
	options ...YamcsClientOption,
) (*YamcsClient, error) {
	return NewYamcsClientWithContext(context.Background(), address, tlsConfig, credentials, options...)
}

func NewYamcsClientWithContext(
	ctx context.Context,
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
		disconnectSignal:               make(chan struct{}),
	}

	// WebSocket URL based on whether TLS is enabled
	wsURL := fmt.Sprintf("%s://%s/api/websocket", getProtocolPrefix(tlsConfig.Enabled), address)

	// Apply any custom client options
	for _, option := range options {
		option(client)
	}

	// Create a new context for the client
	httpManager, err := corehttp.NewHTTPManagerWithContext(ctx, address, tlsConfig, credentials, client.UserAgent, client.KeepAlive, client.UseProtobuf, client.HTTPClient)
	if err != nil {
		return nil, err
	}
	client.HTTP = httpManager

	// Initialize WebSocket handler
	client.WebSocket = ws.NewWebSocketHandler(wsURL, client.UseProtobuf)
	client.WebSocket.Credentials = credentials

	client.WebSocket.SetListener(ws.ParameterListenerID, client.HandleParameterMessage)
	client.WebSocket.SetListener(ws.EventListenerID, client.HandleEventMessage)
	client.WebSocket.SetListener(ws.AlarmListenerID, client.HandleAlarmMessage)
	client.WebSocket.SetListener(ws.GlobalStatusListenerID, client.HandleGlobalStatusMessage)
	client.WebSocket.SetListener(ws.CommandHistoryLisernerID, client.HandleCommandMessage)
	client.WebSocket.SetListener(ws.TimeListenerID, client.HandleTimeMessage)
	client.WebSocket.SetListener(ws.LinksListenerID, client.HandleLinkMessage)
	client.WebSocket.SetListener(ws.ProcessorListenerID, client.HandleProcessorMessage)

	// Handle WebSocket disconnections
	client.WebSocket.SetDisconnectHandler(func() {
		client.clearAllSubscriptions()
		client.signalDisconnected()
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
		client.resetDisconnectSignal()
		go client.WebSocket.Listen()
	}
	return err
}

// Disconnected returns a channel that is closed exactly once the underlying
// WebSocket connection is lost. Streams should select on this channel (rather
// than, or in addition to, periodically polling IsWebSocketConnected()) so a
// dropped connection is reacted to the moment it happens instead of only on
// the next poll tick - which previously left streams blocked on an event
// signal for up to a full polling interval (or indefinitely, for streams that
// didn't poll at all) after the connection was actually already gone.
//
// The returned channel is only valid for the connection that was active when
// Disconnected() was called; after a reconnect, a fresh channel is created,
// so callers that keep running across reconnects should call Disconnected()
// again rather than caching the returned channel.
func (client *YamcsClient) Disconnected() <-chan struct{} {
	client.disconnectMu.Lock()
	defer client.disconnectMu.Unlock()
	if client.disconnectSignal == nil {
		// Defensively lazy-init: guards against a YamcsClient constructed via
		// struct literal (e.g. in tests) rather than NewYamcsClient, which
		// would otherwise leave this nil and make Disconnected() block
		// forever instead of ever firing.
		client.disconnectSignal = make(chan struct{})
	}
	return client.disconnectSignal
}

// signalDisconnected closes the current disconnect signal, waking up any
// stream goroutines blocked on Disconnected(). Safe to call more than once
// for the same disconnect event (e.g. if both an explicit Disconnect() and
// the read loop's own cleanup both observe the same drop).
func (client *YamcsClient) signalDisconnected() {
	client.disconnectMu.Lock()
	defer client.disconnectMu.Unlock()
	if client.disconnectSignal == nil {
		client.disconnectSignal = make(chan struct{})
	}
	select {
	case <-client.disconnectSignal:
		// already closed for this connection attempt
	default:
		close(client.disconnectSignal)
	}
}

// resetDisconnectSignal replaces the disconnect signal with a fresh, open
// channel. Called after every successful (re)connect so a previously closed
// signal doesn't cause newly started streams to immediately think they're
// disconnected.
func (client *YamcsClient) resetDisconnectSignal() {
	client.disconnectMu.Lock()
	defer client.disconnectMu.Unlock()
	client.disconnectSignal = make(chan struct{})
}

func (client *YamcsClient) CloseWebSocketConnection() error {
	return client.WebSocket.Disconnect()
}

func (client *YamcsClient) Close() error {
	var err error
	if client.WebSocket != nil {
		err = client.WebSocket.Disconnect()
	}
	if client.HTTP != nil {
		client.HTTP.Dispose()
	}
	client.clearAllSubscriptions()
	return err
}

func (client *YamcsClient) IsWebSocketConnected() bool {
	return client.WebSocket.IsConnected()
}

func (client *YamcsClient) WebSocketState(ctx context.Context) (*api.State, error) {
	timeout := 10 * time.Second
	if ctx != nil {
		if deadline, ok := ctx.Deadline(); ok {
			timeout = time.Until(deadline)
		}
	}
	return client.WebSocket.GetState(timeout)
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
	client.subsMu.Lock()
	defer client.subsMu.Unlock()
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
