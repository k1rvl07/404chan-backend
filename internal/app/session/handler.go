package session

import (
    "github.com/gin-gonic/gin"
    "net"
    "net/http"
    "strings"
)

type Handler interface {
    CreateSession(c *gin.Context)
    GetUser(c *gin.Context)
}

type handler struct {
    service Service
}

func NewHandler(service Service) Handler {
    return &handler{service: service}
}

func (h *handler) CreateSession(c *gin.Context) {
    userAgent := c.GetHeader("User-Agent")
    ip := extractIP(c)

    session, user, err := h.service.CreateSessionAndUser(userAgent, ip)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "ID":         user.ID,
        "Nickname":   user.Nickname,
        "CreatedAt":  user.CreatedAt,
        "SessionKey": session.SessionKey,
    })
}

func (h *handler) GetUser(c *gin.Context) {
    sessionKey := c.Query("session_key")
    if sessionKey == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "session_key is required"})
        return
    }

    user, err := h.service.GetUserBySessionKey(sessionKey)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "ID":         user.ID,
        "Nickname":   user.Nickname,
        "CreatedAt":  user.CreatedAt,
        "SessionKey": sessionKey,
    })
}

func extractIP(c *gin.Context) string {
    clientIP := c.GetHeader("X-Forwarded-For")
    if clientIP != "" {
        ips := strings.Split(clientIP, ",")
        netIP := net.ParseIP(strings.TrimSpace(ips[0]))
        if netIP != nil {
            return netIP.String()
        }
    }

    clientIP = c.GetHeader("X-Real-IP")
    netIP := net.ParseIP(clientIP)
    if netIP != nil {
        return netIP.String()
    }

    ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
    return ip
}