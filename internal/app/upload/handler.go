package upload

import (
	"backend/internal/app/attachment"
	"backend/internal/providers/minio"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// @Summary Upload files
// @Description Upload files to MinIO storage
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Param files formData array true "Files to upload"
// @Success 200 {array} UploadedFileResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/upload [post]
//
// @Summary Confirm file uploads
// @Description Confirm temporary file uploads to make them permanent
// @Tags Upload
// @Accept json
// @Produce json
// @Param request body ConfirmFilesRequest true "File confirmation request"
// @Success 200 {object} ConfirmFilesResponse
// @Failure 400 {object} map[string]string
// @Router /api/upload/confirm [post]
type UploadedFileResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	ObjectName  string `json:"object_name"`
}

type Handler struct {
	minioP *minio.MinioProvider
	attSvc attachment.Service
	logger *zap.Logger
}

func NewHandler(minioP *minio.MinioProvider, attSvc attachment.Service, logger *zap.Logger) *Handler {
	return &Handler{
		minioP: minioP,
		attSvc: attSvc,
		logger: logger,
	}
}

// @Summary Upload files
// @Description Upload files to MinIO storage
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Param files formData array true "Files to upload"
// @Success 200 {array} UploadedFileResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/upload [post]
func (h *Handler) Upload(c *gin.Context) {
	if h.minioP == nil {
		c.JSON(503, ErrorResponse{Error: "MinIO not configured"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		h.logger.Error("Failed to parse multipart form", zap.Error(err))
		c.JSON(400, ErrorResponse{Error: "Failed to parse form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(400, ErrorResponse{Error: "No files provided"})
		return
	}

	uploadedFiles := make([]*UploadedFileResponse, 0, len(files))

	for _, fileHeader := range files {
		src, err := fileHeader.Open()
		if err != nil {
			h.logger.Error("Failed to open file", zap.String("filename", fileHeader.Filename), zap.Error(err))
			continue
		}

		result, err := h.minioP.UploadFromReader(
			src,
			"tmp/"+generateObjectName(fileHeader.Filename),
			fileHeader.Header.Get("Content-Type"),
			fileHeader.Size,
		)
		src.Close()

		if err != nil {
			h.logger.Error("Failed to upload file", zap.String("filename", fileHeader.Filename), zap.Error(err))
			continue
		}

		att, err := h.attSvc.CreateTemporary(c.Request.Context(), &attachment.CreateAttachmentRequest{
			FileID:      result.ID,
			FileName:    fileHeader.Filename,
			FileURL:     result.URL,
			FileSize:    fileHeader.Size,
			ContentType: fileHeader.Header.Get("Content-Type"),
			ObjectName:  result.ObjectName,
		})
		if err != nil {
			h.logger.Error("Failed to create attachment record", zap.Error(err))
			continue
		}

		uploadedFiles = append(uploadedFiles, &UploadedFileResponse{
			ID:          att.FileID,
			Name:        att.FileName,
			URL:         att.FileURL,
			Size:        att.FileSize,
			ContentType: att.ContentType,
			ObjectName:  att.ObjectName,
		})
	}

	if len(uploadedFiles) == 0 {
		c.JSON(500, ErrorResponse{Error: "Failed to upload any files"})
		return
	}

	c.JSON(200, uploadedFiles)
}

// @Summary Confirm file uploads
// @Description Confirm temporary file uploads to make them permanent
// @Tags Upload
// @Accept json
// @Produce json
// @Param request body ConfirmFilesRequest true "File confirmation request"
// @Success 200 {object} ConfirmFilesResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/upload/confirm [post]
func (h *Handler) ConfirmFiles(c *gin.Context) {
	if h.minioP == nil {
		c.JSON(503, ErrorResponse{Error: "MinIO not configured"})
		return
	}

	var req ConfirmFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ErrorResponse{Error: "Invalid request"})
		return
	}

	if len(req.FileIDs) == 0 {
		c.JSON(400, ErrorResponse{Error: "No file IDs provided"})
		return
	}

	attachments, err := h.attSvc.GetByFileIDs(c.Request.Context(), req.FileIDs)
	if err != nil {
		h.logger.Error("Failed to get attachments", zap.Error(err))
		c.JSON(500, ErrorResponse{Error: "Failed to get attachments"})
		return
	}

	response := ConfirmFilesResponse{
		Files: make([]UploadedFileResponse, 0, len(attachments)),
	}

	for _, att := range attachments {
		if !isTmpObject(att.ObjectName) {
			response.Files = append(response.Files, UploadedFileResponse{
				ID:          att.FileID,
				Name:        att.FileName,
				URL:         att.FileURL,
				Size:        att.FileSize,
				ContentType: att.ContentType,
				ObjectName:  att.ObjectName,
			})
			continue
		}

		permanentObjectName, err := h.minioP.ConfirmTmpObject(att.ObjectName)
		if err != nil {
			h.logger.Error("Failed to confirm tmp object",
				zap.String("file_id", att.FileID),
				zap.Error(err),
			)
			continue
		}

		publicURL := h.minioP.GetPublicURL()
		permanentURL := publicURL + "/" + permanentObjectName

		err = h.attSvc.UpdateObjectName(c.Request.Context(), att.ID, permanentObjectName, permanentURL)
		if err != nil {
			h.logger.Error("Failed to update attachment",
				zap.Uint64("attachment_id", att.ID),
				zap.Error(err),
			)
			continue
		}

		response.Files = append(response.Files, UploadedFileResponse{
			ID:          att.FileID,
			Name:        att.FileName,
			URL:         permanentURL,
			Size:        att.FileSize,
			ContentType: att.ContentType,
			ObjectName:  permanentObjectName,
		})
	}

	c.JSON(200, response)
}

func isTmpObject(objectName string) bool {
	return len(objectName) >= 4 && objectName[:4] == "tmp/"
}

func generateObjectName(filename string) string {
	return minio.GenerateObjectName(filename)
}
