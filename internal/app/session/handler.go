package session

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	CreateSession(c *gin.Context)
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
		"CreatedAt":  session.CreatedAt,
		"SessionKey": session.SessionKey,
	})
}

func extractIP(c *gin.Context) string {
	clientIP := c.GetHeader("X-Forwarded-For")
	if clientIP != "" {
		ips := strings.Split(clientIP, ",")
		if len(ips) > 0 {
			netIP := net.ParseIP(strings.TrimSpace(ips[0]))
			if netIP != nil {
				return netIP.String()
			}
		}
	}

	clientIP = c.GetHeader("X-Real-IP")
	if clientIP != "" {
		netIP := net.ParseIP(clientIP)
		if netIP != nil {
			return netIP.String()
		}
	}

	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return ip
}
