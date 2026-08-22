package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

func wsHandlerEcho(hostname string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("websocket echo handler upgrade: " + err.Error())
			return
		}
		defer c.Close()

		type messageData struct {
			Backend  string `json:"backend"`
			Host     string `json:"host"`
			Endpoint string `json:"endpoint"`
			Sender   string `json:"sender"`
			Message  string `json:"message"`
		}

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				if strings.Contains(err.Error(), "websocket: close ") {
					slog.Info("websocket echo handler closed")
					break
				}

				slog.Error("error while reading websocket message",
					slog.String("error", err.Error()),
					slog.String("message", string(message)),
					slog.Int("mt", mt))
				break
			}

			slog.Debug("websocket message received",
				slog.String("endpoint", r.URL.Path),
				slog.String("message", string(message)),
				slog.String("sender", r.RemoteAddr))

			payload, err := json.Marshal(messageData{
				Backend:  hostname,
				Host:     r.Host,
				Endpoint: r.URL.Path,
				Sender:   r.RemoteAddr,
				Message:  string(message),
			})
			if err != nil {
				slog.Error("error encoding websocket message", "error", err)
				break
			}

			if err = c.WriteMessage(mt, payload); err != nil {
				slog.Error("error while writing message", "error", err)
				break
			}
		}
	}
}
