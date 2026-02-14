package attachment

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	GetAttachments(c *gin.Context)
	DeleteTemporary(c *gin.Context)
}

type handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return &handler{service: service}
}

// @Summary Get attachments
// @Description Get attachments by thread or message ID
// @Tags Attachment
// @Accept json
// @Produce json
// @Param thread_id query int false "Thread ID"
// @Param message_id query int false "Message ID"
// @Success 200 {object} AttachmentListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/attachments [get]
func (h *handler) GetAttachments(c *gin.Context) {
	threadID := c.Query("thread_id")
	messageID := c.Query("message_id")

	var attachments []*Attachment
	var err error

	if threadID != "" {
		attachments, err = h.service.GetByThreadID(c.Request.Context(), parseUint64(threadID))
	} else if messageID != "" {
		attachments, err = h.service.GetByMessageID(c.Request.Context(), parseUint64(messageID))
	} else {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "thread_id or message_id required"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, AttachmentListResponse{Attachments: attachments})
}

// @Summary Delete temporary attachment
// @Description Delete a temporary attachment by file ID
// @Tags Attachment
// @Accept json
// @Produce json
// @Param file_id query string true "File ID"
// @Success 200 {object} DeleteTemporaryResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/attachments [delete]
func (h *handler) DeleteTemporary(c *gin.Context) {
	fileID := c.Query("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "file_id required"})
		return
	}

	if err := h.service.DeleteTemporary(c.Request.Context(), fileID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, DeleteTemporaryResponse{Success: true})
}

func parseUint64(s string) uint64 {
	var result uint64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + uint64(c-'0')
		}
	}
	return result
}
