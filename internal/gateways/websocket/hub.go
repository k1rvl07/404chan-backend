package websocket

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"backend/internal/app/session"
	"backend/internal/app/user"
	"backend/internal/providers/redis"
	"backend/internal/utils"

	"go.uber.org/zap"
)

type Client struct {
	hub        *Hub
	conn       ClientConn
	ID         string
	SessionID  uint64
	UserID     uint64
	SessionKey string
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
	userRepo   user.Repository
	redisP     *redis.RedisProvider
}

func NewHub(
	logger *zap.Logger,
	sessionSvc session.Service,
	eventBus *utils.EventBus,
	userRepo user.Repository,
	redisP *redis.RedisProvider,
) *Hub {
	hub := &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		logger:     logger.Sugar(),
		sessionSvc: sessionSvc,
		eventBus:   eventBus,
		userRepo:   userRepo,
		redisP:     redisP,
	}

	hub.eventBus.Subscribe("nickname_updated", func(event utils.Event) {
		hub.logger.Infow("EventBus: nickname_updated triggered")
		hub.handleNicknameUpdated(event)
	})

	hub.eventBus.Subscribe("thread_created", func(event utils.Event) {
		hub.logger.Infow("EventBus: thread_created triggered")
		hub.handleThreadCreated(event)
	})

	hub.eventBus.Subscribe("message_created", func(event utils.Event) {
		hub.logger.Infow("EventBus: message_created triggered")
		hub.handleMessageCreated(event)
	})

	hub.eventBus.Subscribe("stats_updated", func(event utils.Event) {
		hub.logger.Infow("EventBus: stats_updated triggered")
		hub.handleStatsUpdated(event)
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
				"session_id", client.SessionID,
				"session_key", client.SessionKey,
				"clients_count", len(h.clients),
			)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)

				h.logger.Infow("Client disconnected",
					"client_id", client.ID,
					"user_id", client.UserID,
					"session_id", client.SessionID,
					"clients_count", len(h.clients),
				)

				go func() {
					if err := h.sessionSvc.UpdateSessionEndedAt(client.SessionID); err != nil {
						h.logger.Errorw("Failed to close session on disconnect",
							"session_id", client.SessionID,
							"user_id", client.UserID,
							"error", err,
						)
					} else {
						h.logger.Debugw("Session ended_at updated",
							"session_id", client.SessionID,
							"user_id", client.UserID,
						)
					}
				}()

				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()

					cacheKey := fmt.Sprintf("user:%d:session:%d", client.UserID, client.SessionID)
					if err := h.redisP.Client.Del(ctx, cacheKey).Err(); err != nil {
						h.logger.Errorw("Failed to delete Redis cache on disconnect",
							"cache_key", cacheKey,
							"error", err,
						)
					} else {
						h.logger.Debugw("Redis cache deleted on disconnect",
							"cache_key", cacheKey,
						)
					}
				}()
			}

		case event := <-eventCh:
			h.logger.Infow("EventBus: Received event", "event", event.Event, "data", event.Data)
			h.handleEvent(event)
		}
	}
}

func (h *Hub) handleEvent(event utils.Event) {
	switch event.Event {
	case "nickname_updated":
		h.handleNicknameUpdated(event)
	case "thread_created":
		h.handleThreadCreated(event)
	case "message_created":
		h.handleMessageCreated(event)
	case "stats_updated":
		h.handleStatsUpdated(event)
	default:
		h.logger.Warnw("Unknown event type", "event", event.Event)
	}
}

func (h *Hub) handleThreadCreated(event utils.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		h.logger.Errorw("handleThreadCreated: invalid data type",
			"data_type", fmt.Sprintf("%T", event.Data),
			"data", event.Data)
		return
	}

	timestamp, hasTimestamp := data["timestamp"]
	if !hasTimestamp {
		h.logger.Errorw("handleThreadCreated: missing timestamp in event data")
		return
	}

	threadID, hasThreadID := data["thread_id"]
	if !hasThreadID {
		h.logger.Errorw("handleThreadCreated: missing thread_id in event data")
		return
	}

	boardID, hasBoardID := data["board_id"]
	if !hasBoardID {
		h.logger.Errorw("handleThreadCreated: missing board_id in event data")
		return
	}

	msg := map[string]interface{}{
		"event":     "thread_created",
		"thread_id": threadID,
		"board_id":  boardID,
		"timestamp": timestamp,
	}

	for k, v := range data {
		if k != "thread_id" && k != "board_id" && k != "timestamp" {
			msg[k] = v
		}
	}

	sent := 0
	for client := range h.clients {
		if err := client.conn.WriteJSON(msg); err != nil {
			h.logger.Errorw("Failed to send thread_created to client",
				"client_id", client.ID,
				"user_id", client.UserID,
				"error", err)
			client.conn.Close()
			h.unregister <- client
		} else {
			h.logger.Debugw("Sent thread_created to client",
				"client_id", client.ID,
				"user_id", client.UserID)
			sent++
		}
	}

	h.logger.Infow("thread_created broadcast completed", "sent_to_clients", sent)
}

func (h *Hub) handleMessageCreated(event utils.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		h.logger.Errorw("handleMessageCreated: invalid data type",
			"data_type", fmt.Sprintf("%T", event.Data),
			"data", event.Data)
		return
	}

	timestamp, hasTimestamp := data["timestamp"]
	if !hasTimestamp {
		h.logger.Errorw("handleMessageCreated: missing timestamp in event data")
		return
	}

	messageID, hasMessageID := data["message_id"]
	if !hasMessageID {
		h.logger.Errorw("handleMessageCreated: missing message_id in event data")
		return
	}

	threadID, hasThreadID := data["thread_id"]
	if !hasThreadID {
		h.logger.Errorw("handleMessageCreated: missing thread_id in event data")
		return
	}

	msg := map[string]interface{}{
		"event":      "message_created",
		"message_id": messageID,
		"thread_id":  threadID,
		"timestamp":  timestamp,
	}

	for k, v := range data {
		if k != "message_id" && k != "thread_id" && k != "timestamp" {
			msg[k] = v
		}
	}

	sent := 0
	for client := range h.clients {
		if err := client.conn.WriteJSON(msg); err != nil {
			h.logger.Errorw("Failed to send message_created to client",
				"client_id", client.ID,
				"user_id", client.UserID,
				"error", err)
			client.conn.Close()
			h.unregister <- client
		} else {
			sent++
		}
	}

	h.logger.Infow("message_created broadcast completed", "sent_to_clients", sent)
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
	timestamp, _ := data["timestamp"]

	msg := map[string]interface{}{
		"event":     "nickname_updated",
		"user_id":   userID,
		"nickname":  nickname,
		"timestamp": timestamp,
	}

	sent := 0
	for client := range h.clients {
		if client.UserID == userID {
			if err := client.conn.WriteJSON(msg); err != nil {
				h.logger.Errorw("Failed to send nickname_updated to client",
					"client_id", client.ID,
					"user_id", client.UserID,
					"error", err)
				client.conn.Close()
				h.unregister <- client
			} else {
				h.logger.Debugw("Sent nickname_updated to client",
					"client_id", client.ID,
					"user_id", client.UserID,
					"nickname", nickname)
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
