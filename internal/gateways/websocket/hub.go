package websocket

import (
    "backend/internal/app/session"
    "go.uber.org/zap"
	"encoding/base64"
	"crypto/rand"
)

type Client struct {
    hub       *Hub
    conn      ClientConn
    ID        string
    SessionID string
    UserID    uint64
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
    sessionSvc session.Service
}

func NewHub(logger *zap.Logger, sessionSvc session.Service) *Hub {
    return &Hub{
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
        logger:     logger.Sugar(),
        sessionSvc: sessionSvc,
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
                "session_key", client.SessionID,
                "user_id", client.UserID,
                "clients_count", len(h.clients),
            )

        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)

                session, err := h.sessionSvc.GetSessionByKey(client.SessionID)
                if err == nil && session != nil {
                    if err := h.sessionSvc.UpdateSessionEndedAt(session.ID); err != nil {
                        h.logger.Errorw("Failed to update session ended_at",
                            "session_id", session.ID,
                            "error", err,
                        )
                    }
                }

                h.logger.Infow("Client disconnected",
                    "client_id", client.ID,
                    "session_key", client.SessionID,
                    "user_id", client.UserID,
                    "clients_count", len(h.clients),
                )
            }
        }
    }
}