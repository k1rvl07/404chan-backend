package websocket

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (h *Hub) ServeWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf("Failed to upgrade WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	client := &Client{
		hub:  h,
		conn: conn,
		ID:   generateClientID(),
	}

	h.register <- client
	h.logger.Infow("WebSocket connection established",
		"client_id", client.ID,
		"client_ip", c.ClientIP(),
		"user_agent", c.Request.UserAgent(),
	)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			h.logger.Infow("Client disconnected",
				"client_id", client.ID,
				"reason", err.Error(),
				"client_ip", c.ClientIP(),
			)
			break
		}
	}

	h.unregister <- client
}
