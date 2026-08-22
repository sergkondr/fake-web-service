package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sergkondr/fake-web-service/internal/config"
)

func TestWebSocketEchoReturnsValidJSON(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t, config.Config{
		Hostname: "test-backend",
		Endpoints: []config.Endpoint{
			{Type: config.EndpointTypeWSEcho, Path: "/echo"},
		},
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/echo"
	connection, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer func() {
		_ = connection.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "test completed"),
		)
		_ = connection.Close()
	}()

	wantMessage := "quoted \"value\" with newline\nand unicode: Привет"
	if err = connection.WriteMessage(websocket.TextMessage, []byte(wantMessage)); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}

	messageType, payload, err := connection.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}
	if messageType != websocket.TextMessage {
		t.Fatalf("message type = %d, want %d", messageType, websocket.TextMessage)
	}

	var response struct {
		Backend  string `json:"backend"`
		Host     string `json:"host"`
		Endpoint string `json:"endpoint"`
		Sender   string `json:"sender"`
		Message  string `json:"message"`
	}
	if err = json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal websocket response %q: %v", payload, err)
	}

	if response.Backend != "test-backend" {
		t.Errorf("backend = %q, want %q", response.Backend, "test-backend")
	}
	if response.Endpoint != "/echo" {
		t.Errorf("endpoint = %q, want %q", response.Endpoint, "/echo")
	}
	if response.Message != wantMessage {
		t.Errorf("message = %q, want %q", response.Message, wantMessage)
	}
	if response.Host == "" {
		t.Error("host is empty")
	}
	if response.Sender == "" {
		t.Error("sender is empty")
	}
}

func TestWebSocketEchoRejectsBinaryMessages(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t, config.Config{
		Endpoints: []config.Endpoint{{Type: config.EndpointTypeWSEcho, Path: "/echo"}},
	}))
	defer server.Close()

	connection := dialWebSocket(t, server.URL, "/echo")
	defer connection.Close()

	if err := connection.WriteMessage(websocket.BinaryMessage, []byte{0xff, 0x00}); err != nil {
		t.Fatalf("write binary websocket message: %v", err)
	}

	_, _, err := connection.ReadMessage()
	closeError, ok := err.(*websocket.CloseError)
	if !ok {
		t.Fatalf("read error = %T %v, want websocket close error", err, err)
	}
	if closeError.Code != websocket.CloseUnsupportedData {
		t.Errorf("close code = %d, want %d", closeError.Code, websocket.CloseUnsupportedData)
	}
}

func TestWebSocketStreamSendsJSONMessages(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t, config.Config{
		Hostname: "test-backend",
		Endpoints: []config.Endpoint{{
			Type:   config.EndpointTypeWSStream,
			Path:   "/time",
			Stream: &config.StreamConfig{Interval: 10 * time.Millisecond},
		}},
	}))
	defer server.Close()

	connection := dialWebSocket(t, server.URL, "/time")
	defer connection.Close()
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))

	first := readStreamMessage(t, connection)
	second := readStreamMessage(t, connection)
	if first.Backend != "test-backend" {
		t.Errorf("backend = %q, want %q", first.Backend, "test-backend")
	}
	if first.Endpoint != "/time" {
		t.Errorf("endpoint = %q, want %q", first.Endpoint, "/time")
	}
	if first.Sequence != 1 || second.Sequence != 2 {
		t.Errorf("sequences = %d, %d; want 1, 2", first.Sequence, second.Sequence)
	}
	if _, err := time.Parse(time.RFC3339Nano, first.Timestamp); err != nil {
		t.Errorf("first timestamp %q is not RFC3339Nano: %v", first.Timestamp, err)
	}
}

func TestMultipleWebSocketEndpoints(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t, config.Config{
		Endpoints: []config.Endpoint{
			{Type: config.EndpointTypeWSEcho, Path: "/echo-a"},
			{Type: config.EndpointTypeWSEcho, Path: "/echo-b"},
			{Type: config.EndpointTypeWSStream, Path: "/time", Stream: &config.StreamConfig{Interval: time.Second}},
		},
	}))
	defer server.Close()

	for _, path := range []string{"/echo-a", "/echo-b", "/time"} {
		connection := dialWebSocket(t, server.URL, path)
		closeWebSocket(connection)
	}
}

func TestWebSocketRejectsCrossOriginHandshake(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t, config.Config{
		Endpoints: []config.Endpoint{{Type: config.EndpointTypeWSEcho, Path: "/echo"}},
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/echo"
	connection, response, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": {"https://other.example"}})
	if connection != nil {
		defer connection.Close()
	}
	if err == nil {
		t.Fatal("cross-origin websocket handshake unexpectedly succeeded")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}
		t.Fatalf("cross-origin handshake status = %d, want %d", statusCode, http.StatusForbidden)
	}
}

func TestWebSocketMetrics(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t, config.Config{
		Metrics: config.Metrics{Enabled: true, Path: "/metrics"},
		Endpoints: []config.Endpoint{{
			Type: config.EndpointTypeWSEcho,
			Path: "/echo",
		}},
	}))
	defer server.Close()

	connection := dialWebSocket(t, server.URL, "/echo")
	if err := connection.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}
	if _, _, err := connection.ReadMessage(); err != nil {
		t.Fatalf("read websocket response: %v", err)
	}
	closeWebSocket(connection)

	response, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read metrics: %v", err)
	}
	metrics := string(body)
	for _, want := range []string{
		`fakesvc_websocket_connections_active{endpoint="/echo"} 0`,
		`fakesvc_websocket_connections_total{endpoint="/echo",state="opened"} 1`,
		`fakesvc_websocket_connections_total{endpoint="/echo",state="closed"} 1`,
		`fakesvc_websocket_messages_total{direction="received",endpoint="/echo"} 1`,
		`fakesvc_websocket_messages_total{direction="sent",endpoint="/echo"} 1`,
	} {
		if !strings.Contains(metrics, want) {
			t.Errorf("metrics do not contain %q:\n%s", want, metrics)
		}
	}
	if strings.Contains(metrics, "fakesvc_http_requests_total{endpoint=\"/echo\"") {
		t.Errorf("WebSocket handshake must not be recorded as HTTP request duration metric:\n%s", metrics)
	}
}

func dialWebSocket(t *testing.T, serverURL, path string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + path
	connection, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket %q: %v", path, err)
	}
	return connection
}

func closeWebSocket(connection *websocket.Conn) {
	_ = connection.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "test completed"),
		time.Now().Add(time.Second),
	)
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	for {
		if _, _, err := connection.ReadMessage(); err != nil {
			break
		}
	}
	_ = connection.Close()
}

type decodedStreamMessage struct {
	Backend   string `json:"backend"`
	Endpoint  string `json:"endpoint"`
	Sequence  uint64 `json:"sequence"`
	Timestamp string `json:"timestamp"`
}

func readStreamMessage(t *testing.T, connection *websocket.Conn) decodedStreamMessage {
	t.Helper()
	messageType, payload, err := connection.ReadMessage()
	if err != nil {
		t.Fatalf("read stream message: %v", err)
	}
	if messageType != websocket.TextMessage {
		t.Fatalf("message type = %d, want %d", messageType, websocket.TextMessage)
	}

	var message decodedStreamMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatalf("unmarshal stream message %q: %v", payload, err)
	}
	return message
}
