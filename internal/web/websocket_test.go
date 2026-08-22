package web

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

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
