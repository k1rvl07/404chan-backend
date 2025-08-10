package websocket

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"backend/internal/app/session"
	"backend/internal/utils"

	"go.uber.org/zap"
)

type Client struct {
	hub       *Hub
	conn      ClientConn
	ID        string
	SessionID string
	UserID    uint64
}

type ClientConn interface {
	WriteJSON(v interface{}) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
}

func generateClientID() string {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "xxxxx"
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	logger     *zap.SugaredLogger
	sessionSvc session.Service
	eventBus   *utils.EventBus
}

func NewHub(logger *zap.Logger, sessionSvc session.Service, eventBus *utils.EventBus) *Hub {
	hub := &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		logger:     logger.Sugar(),
		sessionSvc: sessionSvc,
		eventBus:   eventBus,
	}

	hub.eventBus.Subscribe("nickname_updated", func(event utils.Event) {
		hub.logger.Infow("EventBus: nickname_updated triggered")
		hub.handleEvent(event)
	})
	hub.eventBus.Subscribe("stats_updated", func(event utils.Event) {
		hub.logger.Infow("EventBus: stats_updated triggered")
		hub.handleEvent(event)
	})

	return hub
}

func (h *Hub) Run() {
	h.logger.Info("WebSocket Hub started")
	eventCh := h.eventBus.SubscribeCh()

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.logger.Infow("Client connected",
				"client_id", client.ID,
				"user_id", client.UserID,
				"session_key", client.SessionID,
				"clients_count", len(h.clients),
			)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				h.logger.Infow("Client disconnected",
					"client_id", client.ID,
					"user_id", client.UserID,
					"clients_count", len(h.clients),
				)
			}

		case event := <-eventCh:
			h.logger.Infow("EventBus: Received event", "event", event.Event, "data", event.Data)
			h.handleEvent(event)
		}
	}
}

func (h *Hub) handleEvent(event utils.Event) {
	h.logger.Infow("Handling event", "event", event.Event)
	switch event.Event {
	case "nickname_updated":
		h.handleNicknameUpdated(event)
	case "stats_updated":
		h.handleStatsUpdated(event)
	default:
		h.logger.Warnw("Unknown event type", "event", event.Event)
	}
}

func (h *Hub) handleNicknameUpdated(event utils.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		h.logger.Errorw("handleNicknameUpdated: invalid data type",
			"data_type", fmt.Sprintf("%T", event.Data),
			"data", event.Data)
		return
	}

	userIDRaw, exists := data["user_id"]
	if !exists {
		h.logger.Errorw("handleNicknameUpdated: missing user_id in event")
		return
	}

	var userID uint64
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint64(v)
	case int:
		userID = uint64(v)
	case int64:
		userID = uint64(v)
	case uint64:
		userID = v
	default:
		h.logger.Errorw("handleNicknameUpdated: unsupported user_id type",
			"user_id_value", v,
			"user_id_type", fmt.Sprintf("%T", v))
		return
	}

	nickname, _ := data["nickname"].(string)

	msg := map[string]interface{}{
		"event":     "nickname_updated",
		"user_id":   userID,
		"nickname":  nickname,
		"timestamp": data["timestamp"],
	}

	sent := 0
	for client := range h.clients {
		if client.UserID == userID {
			if err := client.conn.WriteJSON(msg); err != nil {
				h.logger.Errorw("Failed to send nickname_updated to client",
					"client_id", client.ID,
					"user_id", client.UserID,
					"error", err,
				)
				client.conn.Close()
				h.unregister <- client
			} else {
				h.logger.Debugw("Sent nickname_updated to client",
					"client_id", client.ID,
					"user_id", client.UserID,
					"nickname", nickname,
				)
				sent++
			}
		}
	}
	h.logger.Infow("nickname_updated broadcast completed", "sent_to_clients", sent)
}

func (h *Hub) handleStatsUpdated(event utils.Event) {
	msg := map[string]interface{}{
		"event": "stats_updated",
		"data":  event.Data,
	}

	sent := 0
	for client := range h.clients {
		if err := client.conn.WriteJSON(msg); err != nil {
			h.logger.Errorw("Failed to send stats_updated", "client_id", client.ID, "error", err)
			client.conn.Close()
			h.unregister <- client
		} else {
			sent++
		}
	}
	h.logger.Infow("stats_updated broadcast completed", "sent_to_clients", sent)
}
