package websocket

import (
	"crypto/rand"
	"encoding/base64"

	"go.uber.org/zap"
)

type Client struct {
	hub  *Hub
	conn ClientConn
	ID   string
}

type ClientConn interface {
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
}

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		logger:     logger.Sugar(),
	}
}

func (h *Hub) Run() {
	h.logger.Info("WebSocket Hub started")

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.logger.Infow("Client connected",
				"client_id", client.ID,
				"clients_count", len(h.clients),
			)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				h.logger.Infow("Client disconnected",
					"client_id", client.ID,
					"clients_count", len(h.clients),
				)
			}
		}
	}
}
