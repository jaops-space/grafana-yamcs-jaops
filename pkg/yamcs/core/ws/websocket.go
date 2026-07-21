package ws

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// WebSocketHandler manages the WebSocket connection and message flow.
type WebSocketHandler struct {
	Credentials      corehttp.Credentials
	connection       *websocket.Conn
	isConnected      atomic.Int32
	useProtobuf      bool
	serverRoot       string
	messageListeners map[ListenerID]MessageListener
	messageCallbacks map[int]MessageCallback
	currentPacketID  int
	mu               sync.Mutex
	nmu              sync.Mutex // network mutex
	disconnectFunc   func()
	handshakeTimeout int
	once             sync.Once // Ensures only one connection attempt
}

type MessageListener func(*api.ServerMessage)
type MessageCallback func(call int, seq int, reply *api.Reply)

func NewWebSocketHandler(serverRoot string, useProtobuf bool) *WebSocketHandler {
	return &WebSocketHandler{
		serverRoot:       serverRoot,
		useProtobuf:      useProtobuf,
		currentPacketID:  0,
		messageListeners: make(map[ListenerID]MessageListener),
		messageCallbacks: make(map[int]MessageCallback),
		handshakeTimeout: 5,
		once:             sync.Once{},
	}
}

func (ws *WebSocketHandler) SetHandshakeTimeout(seconds int) {
	ws.handshakeTimeout = seconds
}

// Connect establishes the WebSocket connection, ensuring it happens only once.
func (ws *WebSocketHandler) Connect(ctx context.Context) error {
	var err error

	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.isConnected.Load() == 1 {
		return nil
	}

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = time.Duration(ws.handshakeTimeout) * time.Second
	if ws.useProtobuf {
		dialer.Subprotocols = []string{"protobuf"}
	} else {
		dialer.Subprotocols = []string{"json"}
	}

	// Prepare headers
	headers := http.Header{}
	if ws.Credentials != nil {
		// Apply credentials headers
		// Here we fake a request so BeforeRequest can set headers normally
		req := &http.Request{Header: headers}
		ws.Credentials.BeforeRequest(req)
	}

	conn, _, dialErr := dialer.DialContext(ctx, ws.serverRoot, headers)
	if dialErr != nil {
		return dialErr
	}
	backend.Logger.Debug("Websocket: Connected to WebSocket.")

	ws.connection = conn
	ws.isConnected.Store(1)

	return err
}

func (ws *WebSocketHandler) Listen() {

	defer ws.ForceDisconnect()
	backend.Logger.Debug("Websocket: Listening for WebSocket messages.")
	defer backend.Logger.Debug("Websocket: Stopped listening for WebSocket messages.")

	for {
		messageType, data, err := ws.connection.ReadMessage()

		if messageType == websocket.CloseMessage {
			backend.Logger.Debug("Websocket: Received close message.")
			return
		}

		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				backend.Logger.Debug("WebSocket connection closed normally.")
			} else {
				backend.Logger.Error("Websocket: WebSocket closed with error: ", err)
			}
			return
		}

		message := &api.ServerMessage{}
		if ws.useProtobuf {
			if err = proto.Unmarshal(data, message); err != nil {
				backend.Logger.Error("Error unmarshalling message: ", err)
				continue
			}
		} else {
			if err = protojson.Unmarshal(data, message); err != nil {
				backend.Logger.Error("Error unmarshalling message: ", err)
				continue
			}
		}

		if message.GetType() == "reply" {
			reply := api.Reply{}
			if err = message.Data.UnmarshalTo(&reply); err != nil {
				backend.Logger.Error("Error unmarshalling reply: ", err)
				continue
			}
			ws.mu.Lock()
			callback, found := ws.messageCallbacks[int(reply.GetReplyTo())]
			ws.mu.Unlock()

			if found {
				callback(int(message.GetCall()), int(message.GetSeq()), &reply)
			}
		}

		for _, listener := range ws.messageListeners {
			listener(message)
		}
	}
}

func (ws *WebSocketHandler) IsConnected() bool {
	return ws.isConnected.Load() == 1
}

// Disconnect properly closes the WebSocket and resets connection state.
func (ws *WebSocketHandler) Disconnect() error {
	if !ws.IsConnected() {
		return exception.New("WebSocket is not connected.", "WS_NOT_CONNECTED")
	}
	if ws.connection == nil {
		ws.ForceDisconnect()
		return exception.New("WebSocket connection is not initialized.", "WS_CONNECTION_NOT_INITIALIZED")
	}
	ws.nmu.Lock()
	err := ws.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	ws.nmu.Unlock()

	ws.ForceDisconnect()
	return err
}

func (ws *WebSocketHandler) ForceDisconnect() {
	ws.isConnected.Store(0)
	if ws.connection != nil {
		ws.connection.Close()
		ws.connection = nil
	}
	ws.once = sync.Once{} // Reset Once so connection can be retried.
	if ws.disconnectFunc != nil {
		ws.disconnectFunc()
	}
}

type syncResponse struct {
	reply *api.Reply
	call  int
	seq   int
}

func (ws *WebSocketHandler) SendSync(
	ctx context.Context,
	message *api.ClientMessage,
) (*api.Reply, int, int, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	responseCh := make(chan syncResponse, 1)

	// Allocate the ID and register the callback before sending.
	ws.mu.Lock()
	currentID := ws.currentPacketID
	ws.currentPacketID++

	message.Id = int32(currentID)

	ws.messageCallbacks[currentID] = func(
		call int,
		seq int,
		reply *api.Reply,
	) {
		// Buffered channel prevents the callback from blocking if timeout and
		// reply arrival happen at nearly the same time.
		select {
		case responseCh <- syncResponse{
			reply: reply,
			call:  call,
			seq:   seq,
		}:
		default:
		}
	}
	ws.mu.Unlock()

	defer func() {
		ws.mu.Lock()
		delete(ws.messageCallbacks, currentID)
		ws.mu.Unlock()
	}()

	data, err := ws.marshalClientMessage(message)
	if err != nil {
		return nil, 0, 0, err
	}
	if !ws.IsConnected() || ws.connection == nil {
		return nil, 0, 0, exception.New("WebSocket is not connected.", "WS_NOT_CONNECTED")
	}

	ws.nmu.Lock()
	err = ws.connection.WriteMessage(websocket.BinaryMessage, data)
	ws.nmu.Unlock()

	if err != nil {
		return nil, 0, 0, err
	}

	select {
	case response := <-responseCh:
		return response.reply, response.call, response.seq, nil

	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, 0, 0, exception.New(
				"Timeout waiting for reply.",
				"WS_TIMEOUT",
			)
		}

		return nil, 0, 0, exception.Wrap(
			"Context canceled waiting for reply",
			"WS_CONTEXT_CANCELED",
			ctx.Err(),
		)
	}
}

func (ws *WebSocketHandler) marshalClientMessage(
	message *api.ClientMessage,
) ([]byte, error) {
	if ws.useProtobuf {
		return proto.Marshal(message)
	}

	return protojson.Marshal(message)
}

func (ws *WebSocketHandler) Send(message *api.ClientMessage) error {
	if !ws.IsConnected() || ws.connection == nil {
		return exception.New("WebSocket is not connected.", "WS_NOT_CONNECTED")
	}

	var data []byte
	var err error
	if ws.useProtobuf {
		data, err = proto.Marshal(message)
	} else {
		data, err = protojson.Marshal(message)
	}
	if err != nil {
		return err
	}

	ws.nmu.Lock()
	defer ws.nmu.Unlock()

	return ws.connection.WriteMessage(websocket.BinaryMessage, data)
}

// AddListener registers a listener for a specific message type.
func (ws *WebSocketHandler) AddListener(name ListenerID, listener MessageListener) {
	ws.messageListeners[name] = listener
}

// RemoveListener removes a listener by name.
func (ws *WebSocketHandler) RemoveListener(name ListenerID) {
	delete(ws.messageListeners, name)
}

// SetDisconnectHandler sets the callback function to be called on disconnection.
func (ws *WebSocketHandler) SetDisconnectHandler(handler func()) {
	ws.disconnectFunc = handler
}
