package ws

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// WebSocketHandler manages the WebSocket connection and message flow.
type WebSocketHandler struct {
	connection       *websocket.Conn
	isConnected      int32
	useProtobuf      bool
	serverRoot       string
	messageListeners map[ListenerID]MessageListener
	messageCallbacks map[int]MessageCallback
	currentPacketID  int
	mutex            sync.Mutex
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
		messageListeners: make(map[ListenerID]MessageListener),
		messageCallbacks: make(map[int]MessageCallback),
		handshakeTimeout: 5,
		once:             sync.Once{},
	}
}

func (websocketHandler *WebSocketHandler) SetHandshakeTimeout(seconds int) {
	websocketHandler.handshakeTimeout = seconds
}

// Connect establishes the WebSocket connection, ensuring it happens only once.
func (websocketHandler *WebSocketHandler) Connect() error {
	var err error

	websocketHandler.mutex.Lock()
	defer websocketHandler.mutex.Unlock()

	if atomic.LoadInt32(&websocketHandler.isConnected) == 1 {
		return nil
	}

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = time.Duration(websocketHandler.handshakeTimeout) * time.Second
	if websocketHandler.useProtobuf {
		dialer.Subprotocols = []string{"protobuf"}
	} else {
		dialer.Subprotocols = []string{"json"}
	}

	conn, _, dialErr := dialer.Dial(websocketHandler.serverRoot, nil)
	if dialErr != nil {
		return dialErr
	}
	backend.Logger.Debug("Websocket: Connected to WebSocket.")

	websocketHandler.connection = conn
	atomic.StoreInt32(&websocketHandler.isConnected, 1)

	return err
}

func (websocketHandler *WebSocketHandler) Listen() {

	defer websocketHandler.ForceDisconnect()
	backend.Logger.Debug("Websocket: Listening for WebSocket messages.")
	defer backend.Logger.Debug("Websocket: Stopped listening for WebSocket messages.")

	for {
		messageType, data, err := websocketHandler.connection.ReadMessage()

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
		if websocketHandler.useProtobuf {
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
			if callback, found := websocketHandler.messageCallbacks[int(reply.GetReplyTo())]; found {
				callback(int(message.GetCall()), int(message.GetSeq()), &reply)
			}
		}

		for _, listener := range websocketHandler.messageListeners {
			listener(message)
		}
	}
}

func (websocketHandler *WebSocketHandler) IsConnected() bool {
	return atomic.LoadInt32(&websocketHandler.isConnected) == 1
}

// Disconnect properly closes the WebSocket and resets connection state.
func (websocketHandler *WebSocketHandler) Disconnect() error {
	if !websocketHandler.IsConnected() {
		return exception.New("WebSocket is not connected.", "WS_NOT_CONNECTED")
	}
	err := websocketHandler.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	websocketHandler.ForceDisconnect()
	return err
}

func (websocketHandler *WebSocketHandler) ForceDisconnect() {
	atomic.StoreInt32(&websocketHandler.isConnected, 0)
	websocketHandler.connection.Close()
	websocketHandler.once = sync.Once{} // Reset Once so connection can be retried.
	if websocketHandler.disconnectFunc != nil {
		websocketHandler.disconnectFunc()
	}
}

func (websocketHandler *WebSocketHandler) SendSync(message *api.ClientMessage) (*api.Reply, int, int, error) {
	websocketHandler.mutex.Lock()
	message.Id = int32(websocketHandler.currentPacketID)
	currentID := websocketHandler.currentPacketID
	websocketHandler.currentPacketID++
	websocketHandler.mutex.Unlock()

	var data []byte
	var err error
	if websocketHandler.useProtobuf {
		data, err = proto.Marshal(message)
	} else {
		data, err = protojson.Marshal(message)
	}

	if err != nil {
		return nil, 0, 0, err
	}

	websocketHandler.mutex.Lock()
	if err = websocketHandler.connection.WriteMessage(websocket.BinaryMessage, data); err != nil {
		websocketHandler.mutex.Unlock()
		return nil, 0, 0, err
	}
	websocketHandler.mutex.Unlock()

	done := make(chan struct{})
	var reply *api.Reply
	var call, seq int

	websocketHandler.mutex.Lock()
	websocketHandler.messageCallbacks[currentID] = func(returnedCall int, returnedSeq int, returnedReply *api.Reply) {
		call = returnedCall
		seq = returnedSeq
		reply = returnedReply
		close(done)
	}
	websocketHandler.mutex.Unlock()

	select {
	case <-done:
		websocketHandler.mutex.Lock()
		delete(websocketHandler.messageCallbacks, currentID)
		websocketHandler.mutex.Unlock()
		return reply, call, seq, nil
	case <-time.After(10 * time.Second):
		websocketHandler.mutex.Lock()
		delete(websocketHandler.messageCallbacks, currentID)
		websocketHandler.mutex.Unlock()
		return nil, 0, 0, exception.New("Timeout waiting for reply.", "WS_TIMEOUT")
	}
}

func (websocketHandler *WebSocketHandler) Send(message *api.ClientMessage) error {

	websocketHandler.mutex.Lock()
	defer websocketHandler.mutex.Unlock()

	var data []byte
	var err error
	if websocketHandler.useProtobuf {
		data, err = proto.Marshal(message)
	} else {
		data, err = protojson.Marshal(message)
	}
	if err != nil {
		return err
	}
	return websocketHandler.connection.WriteMessage(websocket.BinaryMessage, data)
}

// AddListener registers a listener for a specific message type.
func (websocketHandler *WebSocketHandler) AddListener(name ListenerID, listener MessageListener) {
	websocketHandler.messageListeners[name] = listener
}

// RemoveListener removes a listener by name.
func (websocketHandler *WebSocketHandler) RemoveListener(name ListenerID) {
	delete(websocketHandler.messageListeners, name)
}

// SetDisconnectHandler sets the callback function to be called on disconnection.
func (websocketHandler *WebSocketHandler) SetDisconnectHandler(handler func()) {
	websocketHandler.disconnectFunc = handler
}
