// Package ws contains WebSocket endpoint handlers and connection lifecycle management.
package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
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

var upgrader = websocket.Upgrader{CheckOrigin: sameOrigin}

type connection struct {
	websocket *websocket.Conn
	endpoint  string
	observer  Observer
	done      chan struct{}
	stopOnce  sync.Once
	writeMu   sync.Mutex
}

// Observer receives bounded WebSocket endpoint activity events.
type Observer interface {
	ConnectionOpened(endpoint string)
	ConnectionClosed(endpoint string)
	MessageReceived(endpoint string)
	MessageSent(endpoint string)
	ReadError(endpoint string)
	WriteError(endpoint string)
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
		connection, err := upgrade(w, r, endpoint, observer)
		if err != nil {
			slog.Error("websocket echo handler upgrade", "error", err)
			return
		}
		defer connection.stop()
		connection.connectionOpened()
		defer connection.connectionClosed()

		for {
			messageType, message, err := connection.websocket.ReadMessage()
			if err != nil {
				connection.logReadError("websocket echo handler", err)
				return
			}
			connection.messageReceived()
			if messageType != websocket.TextMessage {
				connection.closeWith(websocket.CloseUnsupportedData, "only text messages are supported")
				return
			}
			if !utf8.Valid(message) {
				connection.closeWith(websocket.CloseInvalidFramePayloadData, "message must be valid UTF-8")
				return
			}

			slog.Debug("websocket message received",
				slog.String("endpoint", r.URL.Path),
				slog.String("message", string(message)),
				slog.String("sender", r.RemoteAddr))

			payload, err := json.Marshal(echoMessage{
				Backend:  hostname,
				Host:     r.Host,
				Endpoint: r.URL.Path,
				Sender:   r.RemoteAddr,
				Message:  string(message),
			})
			if err != nil {
				slog.Error("encode websocket echo response", "error", err)
				return
			}
			if err = connection.writeMessage(websocket.TextMessage, payload); err != nil {
				slog.Error("write websocket echo response", "error", err)
				return
			}
		}
	}
}

// Stream writes a JSON time message immediately and then once per interval.
func Stream(hostname, endpoint string, interval time.Duration, observer Observer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrade(w, r, endpoint, observer)
		if err != nil {
			slog.Error("websocket stream handler upgrade", "error", err)
			return
		}
		defer connection.stop()
		connection.connectionOpened()
		defer connection.connectionClosed()

		readerDone := make(chan struct{})
		go func() {
			defer close(readerDone)
			for {
				_, _, err := connection.websocket.ReadMessage()
				if err != nil {
					connection.logReadError("websocket stream handler", err)
					return
				}
				connection.messageReceived()
			}
		}()

		var sequence uint64
		send := func() error {
			sequence++
			payload, err := json.Marshal(streamMessage{
				Backend:   hostname,
				Endpoint:  r.URL.Path,
				Sequence:  sequence,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			})
			if err != nil {
				return err
			}
			return connection.writeMessage(websocket.TextMessage, payload)
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

func upgrade(w http.ResponseWriter, r *http.Request, endpoint string, observer Observer) (*connection, error) {
	websocketConnection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	websocketConnection.SetReadLimit(maxMessageSize)
	_ = websocketConnection.SetReadDeadline(time.Now().Add(pongWait))
	websocketConnection.SetPongHandler(func(string) error {
		return websocketConnection.SetReadDeadline(time.Now().Add(pongWait))
	})

	managed := &connection{
		websocket: websocketConnection,
		endpoint:  endpoint,
		observer:  observer,
		done:      make(chan struct{}),
	}
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
				slog.Debug("write websocket ping", "error", err)
				connection.stop()
				return
			}
		}
	}
}

func (connection *connection) writeMessage(messageType int, payload []byte) error {
	connection.writeMu.Lock()
	defer connection.writeMu.Unlock()

	if err := connection.websocket.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		connection.writeError()
		return err
	}
	err := connection.websocket.WriteMessage(messageType, payload)
	if err != nil {
		connection.writeError()
		return err
	}
	connection.messageSent()
	return nil
}

func (connection *connection) writeControl(messageType int, payload []byte) error {
	connection.writeMu.Lock()
	defer connection.writeMu.Unlock()

	return connection.websocket.WriteControl(messageType, payload, time.Now().Add(writeWait))
}

func (connection *connection) closeWith(code int, message string) {
	if err := connection.writeControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, message)); err != nil {
		slog.Debug("write websocket close message", "error", err)
	}
}

func (connection *connection) stop() {
	connection.stopOnce.Do(func() {
		close(connection.done)
		_ = connection.websocket.Close()
	})
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsedOrigin, err := url.Parse(origin)
	return err == nil && parsedOrigin.Host == r.Host
}

func (connection *connection) logReadError(handler string, err error) {
	if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		connection.readError()
		slog.Error(handler+" read", "error", err)
		return
	}
	slog.Debug(handler + " closed")
}

func (connection *connection) connectionOpened() {
	if connection.observer != nil {
		connection.observer.ConnectionOpened(connection.endpoint)
	}
}

func (connection *connection) connectionClosed() {
	if connection.observer != nil {
		connection.observer.ConnectionClosed(connection.endpoint)
	}
}

func (connection *connection) messageReceived() {
	if connection.observer != nil {
		connection.observer.MessageReceived(connection.endpoint)
	}
}

func (connection *connection) messageSent() {
	if connection.observer != nil {
		connection.observer.MessageSent(connection.endpoint)
	}
}

func (connection *connection) readError() {
	if connection.observer != nil {
		connection.observer.ReadError(connection.endpoint)
	}
}

func (connection *connection) writeError() {
	if connection.observer != nil {
		connection.observer.WriteError(connection.endpoint)
	}
}
