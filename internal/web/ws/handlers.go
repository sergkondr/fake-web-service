// Package ws contains WebSocket endpoint handlers and connection lifecycle management.
package ws

import (
	"log/slog"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 1 << 20 // 1 MiB
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = pongWait * 9 / 10
)

var upgrader = websocket.Upgrader{}

// Event identifies a bounded WebSocket activity metric.
type Event uint8

const (
	ConnectionOpened Event = iota
	ConnectionClosed
	MessageReceived
	MessageSent
	ReadError
	WriteError
)

// Observer receives WebSocket activity events for one configured endpoint.
type Observer func(Event)

type connection struct {
	websocket *websocket.Conn
	observer  Observer
	done      chan struct{}
	stopOnce  sync.Once
}

type echoMessage struct {
	Backend  string `json:"backend"`
	Host     string `json:"host"`
	Endpoint string `json:"endpoint"`
	Sender   string `json:"sender"`
	Message  string `json:"message"`
}

type streamMessage struct {
	Backend   string `json:"backend"`
	Endpoint  string `json:"endpoint"`
	Sequence  uint64 `json:"sequence"`
	Timestamp string `json:"timestamp"`
}

// Echo accepts text frames and returns their metadata and content as JSON text frames.
func Echo(hostname, endpoint string, observer Observer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrade(w, r, observer)
		if err != nil {
			slog.Debug("websocket echo handler upgrade", "error", err)
			return
		}
		connection.observe(ConnectionOpened)
		defer connection.stop()

		for {
			messageType, message, err := connection.websocket.ReadMessage()
			if err != nil {
				connection.logReadError("websocket echo handler", err)
				return
			}
			connection.observe(MessageReceived)
			if messageType != websocket.TextMessage {
				connection.closeWith(websocket.CloseUnsupportedData, "only text messages are supported")
				return
			}
			if !utf8.Valid(message) {
				connection.closeWith(websocket.CloseInvalidFramePayloadData, "message must be valid UTF-8")
				return
			}

			slog.Debug("websocket message received",
				slog.String("endpoint", endpoint),
				slog.String("message", string(message)),
				slog.String("sender", r.RemoteAddr))

			if err = connection.writeJSON(echoMessage{
				Backend:  hostname,
				Host:     r.Host,
				Endpoint: endpoint,
				Sender:   r.RemoteAddr,
				Message:  string(message),
			}); err != nil {
				slog.Error("write websocket echo response", "error", err)
				return
			}
		}
	}
}

// Stream writes a JSON time message immediately and then once per interval.
func Stream(hostname, endpoint string, interval time.Duration, observer Observer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrade(w, r, observer)
		if err != nil {
			slog.Debug("websocket stream handler upgrade", "error", err)
			return
		}
		connection.observe(ConnectionOpened)
		defer connection.stop()

		readerDone := make(chan struct{})
		go func() {
			defer close(readerDone)
			for {
				_, _, err := connection.websocket.ReadMessage()
				if err != nil {
					connection.logReadError("websocket stream handler", err)
					return
				}
				connection.observe(MessageReceived)
			}
		}()

		var sequence uint64
		send := func() error {
			sequence++
			return connection.writeJSON(streamMessage{
				Backend:   hostname,
				Endpoint:  endpoint,
				Sequence:  sequence,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			})
		}

		if err := send(); err != nil {
			slog.Error("write websocket stream message", "error", err)
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-readerDone:
				return
			case <-connection.done:
				return
			case <-ticker.C:
				if err := send(); err != nil {
					slog.Error("write websocket stream message", "error", err)
					return
				}
			}
		}
	}
}

func upgrade(w http.ResponseWriter, r *http.Request, observer Observer) (*connection, error) {
	websocketConnection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	websocketConnection.SetReadLimit(maxMessageSize)
	_ = websocketConnection.SetReadDeadline(time.Now().Add(pongWait))
	websocketConnection.SetPongHandler(func(string) error {
		return websocketConnection.SetReadDeadline(time.Now().Add(pongWait))
	})

	managed := &connection{websocket: websocketConnection, observer: observer, done: make(chan struct{})}
	go managed.pingLoop()
	return managed, nil
}

func (connection *connection) pingLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-connection.done:
			return
		case <-ticker.C:
			if err := connection.writeControl(websocket.PingMessage, nil); err != nil {
				connection.stop()
				return
			}
		}
	}
}

func (connection *connection) writeJSON(value any) error {
	if err := connection.websocket.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		connection.observe(WriteError)
		return err
	}
	if err := connection.websocket.WriteJSON(value); err != nil {
		connection.observe(WriteError)
		return err
	}
	connection.observe(MessageSent)
	return nil
}

func (connection *connection) writeControl(messageType int, payload []byte) error {
	if err := connection.websocket.WriteControl(messageType, payload, time.Now().Add(writeWait)); err != nil {
		connection.observe(WriteError)
		return err
	}
	return nil
}

func (connection *connection) closeWith(code int, message string) {
	if err := connection.writeControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, message)); err != nil {
		slog.Debug("write websocket close message", "error", err)
	}
}

func (connection *connection) stop() {
	connection.stopOnce.Do(func() {
		connection.observe(ConnectionClosed)
		close(connection.done)
		_ = connection.websocket.Close()
	})
}

func (connection *connection) observe(event Event) {
	if connection.observer != nil {
		connection.observer(event)
	}
}

func (connection *connection) logReadError(handler string, err error) {
	if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		connection.observe(ReadError)
		slog.Error(handler+" read", "error", err)
		return
	}
	slog.Debug(handler + " closed")
}
