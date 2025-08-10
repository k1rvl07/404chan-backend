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
	sessionKey := c.Query("session_key")
	if sessionKey == "" {
		h.logger.Warnw("WebSocket connection rejected: session_key missing",
			"client_ip", c.ClientIP(),
			"user_agent", c.GetHeader("User-Agent"),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_key is required"})
		return
	}

	user, err := h.sessionSvc.GetUserBySessionKey(sessionKey)
	if err != nil {
		h.logger.Warnw("WebSocket connection rejected: user not found",
			"session_key", sessionKey,
			"client_ip", c.ClientIP(),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorw("Failed to upgrade connection",
			"session_key", sessionKey,
			"error", err,
		)
		return
	}
	defer conn.Close()

	client := &Client{
		hub:       h,
		conn:      conn,
		ID:        generateClientID(),
		SessionID: sessionKey,
		UserID:    user.ID,
	}

	h.logger.Infow("WebSocket connection established",
		"client_id", client.ID,
		"user_id", client.UserID,
		"session_key", client.SessionID,
		"client_ip", c.ClientIP(),
		"user_agent", c.GetHeader("User-Agent"),
	)

	h.register <- client

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	h.unregister <- client
}
