package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	wsURLPreffix = "/ws"
)

type wsTemplateData struct {
	EchoEndpoint string
}

func newWSRoot(endpoint string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		data := &wsTemplateData{EchoEndpoint: wsURLPreffix + endpoint}
		wsIndexTemplate.Execute(w, data)
	}
}

func wsHandlerEcho(hostname string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("ws echo handler upgrade: " + err.Error())
			return
		}
		defer c.Close()

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				if strings.Contains(err.Error(), "websocket: close ") {
					slog.Info("ws echo handler closed")
					break
				}

				slog.Error("error while reading message",
					slog.String("error", err.Error()),
					slog.String("message", string(message)),
					slog.Int("mt", mt))
				break
			}

			slog.Debug("WS message received",
				slog.String("endpoint", r.URL.Path),
				slog.String("message", string(message)),
				slog.String("sender", r.RemoteAddr))

			err = c.WriteMessage(mt, []byte(
				fmt.Sprintf(`{"backend":"%s", "host":"%s", "endpoint":"%s", "sender":"%s", "message":"%s"}`,
					hostname, r.Host, r.URL.Path, r.RemoteAddr, message,
				)))
			if err != nil {
				slog.Error("error while writing message:" + err.Error())
				break
			}
		}
	}
}
