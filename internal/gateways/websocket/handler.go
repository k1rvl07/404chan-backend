package websocket

import (
	"net/http"
	"time"

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

	lastChange, err := h.userRepo.GetUserLastNicknameChange(user.ID)
	if err != nil {
		h.logger.Errorw("ServeWS: failed to get last nickname change", "user_id", user.ID, "error", err)
	} else {
		now := time.Now().UTC()
		if lastChange != nil && now.Sub(*lastChange) < time.Minute {
			msg := map[string]interface{}{
				"event":     "nickname_updated",
				"user_id":   user.ID,
				"nickname":  user.Nickname,
				"timestamp": lastChange.Unix(),
			}
			if err := conn.WriteJSON(msg); err != nil {
				h.logger.Errorw("ServeWS: failed to send initial nickname_updated", "user_id", user.ID, "error", err)
			} else {
				h.logger.Debugw("ServeWS: sent initial nickname_updated due to active cooldown",
					"user_id", user.ID,
					"nickname", user.Nickname,
					"time_left_seconds", time.Minute-now.Sub(*lastChange),
				)
			}
		}
	}

	h.register <- client

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	h.unregister <- client
}
