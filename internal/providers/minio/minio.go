package minio

import (
	"backend/internal/config"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

type MinioProvider struct {
	client    *minio.Client
	bucket    string
	maxSize   int64
	maxFiles  int
	logger    *zap.Logger
	publicURL string
}

func NewMinioProvider(cfg *config.Config, logger *zap.Logger) (*MinioProvider, error) {
	minioURL := cfg.MinioURL
	if !strings.HasPrefix(minioURL, "http://") && !strings.HasPrefix(minioURL, "https://") {
		minioURL = "https://" + minioURL
	}

	u, err := url.Parse(minioURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse minio URL: %w", err)
	}
	secure := u.Scheme == "https"

	logger.Info("Initializing MinIO", zap.String("url", minioURL), zap.Bool("secure", secure))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}
	tr.MaxIdleConnsPerHost = 256

	client, err := minio.New(minioURL, &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.MinioUser, cfg.MinioPassword, ""),
		Secure:    secure,
		Transport: tr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	publicURL := cfg.MinioPublicURL
	if publicURL == "" {
		publicURL = fmt.Sprintf("http://%s/%s", cfg.MinioURL, cfg.MinioBucket)
	}

	provider := &MinioProvider{
		client:    client,
		bucket:    cfg.MinioBucket,
		maxSize:   cfg.MaxFileSize,
		maxFiles:  cfg.MaxFilesPerPost,
		logger:    logger,
		publicURL: publicURL,
	}

	if err := provider.ensureBucket(); err != nil {
		return nil, err
	}

	return provider, nil
}

func (m *MinioProvider) ensureBucket() error {
	ctx := context.Background()

	m.logger.Info("Checking if bucket exists", zap.String("bucket", m.bucket))
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		m.logger.Error("BucketExists error", zap.Error(err), zap.String("bucket", m.bucket))
		return fmt.Errorf("failed to check bucket: %w", err)
	}

	m.logger.Info("Bucket exists check result", zap.Bool("exists", exists), zap.String("bucket", m.bucket))

	if !exists {
		err := m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		m.logger.Info("Created MinIO bucket", zap.String("bucket", m.bucket))
	}

	if err := m.setBucketPolicy(ctx); err != nil {
		m.logger.Warn("Failed to set bucket policy", zap.Error(err))
	}

	return nil
}

func (m *MinioProvider) setBucketPolicy(ctx context.Context) error {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "PublicReadGetObject",
				"Effect": "Allow",
				"Principal": "*",
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::` + m.bucket + `/*"]
			}
		]
	}`
	return m.client.SetBucketPolicy(ctx, m.bucket, policy)
}

func (m *MinioProvider) UploadFile(file *multipart.FileHeader) (*UploadedFile, error) {
	if file.Size > m.maxSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d MB", m.maxSize/(1024*1024))
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	ext := filepath.Ext(file.Filename)
	contentType := detectContentType(ext)

	objectName := GenerateObjectName(file.Filename)
	tmpObjectName := "tmp/" + objectName

	_, err = m.client.PutObject(context.Background(), m.bucket, tmpObjectName, src, file.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	m.logger.Info("File uploaded to tmp",
		zap.String("filename", file.Filename),
		zap.String("object_name", tmpObjectName),
		zap.Int64("size", file.Size),
	)

	return &UploadedFile{
		ID:          uuid.New().String(),
		Name:        file.Filename,
		URL:         m.publicURL + "/" + tmpObjectName,
		Size:        file.Size,
		ContentType: contentType,
		ObjectName:  tmpObjectName,
	}, nil
}

func (m *MinioProvider) ConfirmTmpObject(tmpObjectName string) (string, error) {
	src, err := m.client.GetObject(context.Background(), m.bucket, tmpObjectName, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get tmp object: %w", err)
	}
	defer src.Close()

	permanentObjectName := strings.TrimPrefix(tmpObjectName, "tmp/")

	dest := minio.CopyDestOptions{
		Bucket: m.bucket,
		Object: permanentObjectName,
	}

	srcOpts := minio.CopySrcOptions{
		Bucket: m.bucket,
		Object: tmpObjectName,
	}

	_, err = m.client.CopyObject(context.Background(), dest, srcOpts)
	if err != nil {
		return "", fmt.Errorf("failed to copy object: %w", err)
	}

	err = m.DeleteFile(tmpObjectName)
	if err != nil {
		m.logger.Warn("Failed to delete tmp file", zap.Error(err))
	}

	m.logger.Info("Confirmed tmp file",
		zap.String("tmp_object", tmpObjectName),
		zap.String("permanent_object", permanentObjectName),
	)

	return permanentObjectName, nil
}

func (m *MinioProvider) DeleteTmpFilesOlderThan(maxAge time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	objectsCh := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{
		Prefix:    "tmp/",
		Recursive: true,
	})

	for object := range objectsCh {
		if object.Err != nil {
			return object.Err
		}

		objectTime := object.LastModified
		if time.Since(objectTime) > maxAge {
			if err := m.DeleteFile(object.Key); err != nil {
				m.logger.Warn("Failed to delete old tmp file",
					zap.String("object", object.Key),
					zap.Error(err),
				)
			} else {
				m.logger.Info("Deleted old tmp file",
					zap.String("object", object.Key),
					zap.Duration("age", time.Since(objectTime)),
				)
			}
		}
	}

	return nil
}

func (m *MinioProvider) UploadMultiple(files []*multipart.FileHeader) ([]*UploadedFile, error) {
	if len(files) > m.maxFiles {
		return nil, fmt.Errorf("maximum %d files allowed per post", m.maxFiles)
	}

	uploaded := make([]*UploadedFile, 0, len(files))

	for _, file := range files {
		result, err := m.UploadFile(file)
		if err != nil {
			return nil, err
		}
		uploaded = append(uploaded, result)
	}

	return uploaded, nil
}

func (m *MinioProvider) DeleteFile(objectName string) error {
	ctx := context.Background()

	err := m.client.RemoveObject(ctx, m.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	m.logger.Info("File deleted successfully", zap.String("object_name", objectName))
	return nil
}

func (m *MinioProvider) DeleteFiles(objectNames []string) error {
	for _, name := range objectNames {
		if err := m.DeleteFile(name); err != nil {
			return err
		}
	}
	return nil
}

func (m *MinioProvider) GetClient() *minio.Client {
	return m.client
}

func (m *MinioProvider) GetBucket() string {
	return m.bucket
}

func (m *MinioProvider) GetPublicURL() string {
	return m.publicURL
}

func GenerateObjectName(filename string) string {
	timestamp := time.Now().Format("2006/01/02")
	uuidStr1 := uuid.New().String()
	uuidStr2 := uuid.New().String()
	ext := filepath.Ext(filename)
	return fmt.Sprintf("%s/%s_%s%s", timestamp, uuidStr1, uuidStr2, ext)
}

func detectContentType(ext string) string {
	ext = strings.ToLower(ext)
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".pdf":  "application/pdf",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

type UploadedFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	ObjectName  string `json:"object_name"`
}

type Attachment struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	ThreadID    *uint64   `json:"thread_id,omitempty"`
	MessageID   *uint64   `json:"message_id,omitempty"`
	FileID      string    `json:"file_id" gorm:"type:varchar(36);not null"`
	FileName    string    `json:"file_name" gorm:"not null"`
	FileURL     string    `json:"file_url" gorm:"not null"`
	FileSize    int64     `json:"file_size" gorm:"not null"`
	ContentType string    `json:"content_type" gorm:"type:varchar(100);not null"`
	ObjectName  string    `json:"object_name" gorm:"type:varchar(500);not null"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Attachment) TableName() string {
	return "attachments"
}

func (m *MinioProvider) GeneratePresignedURL(objectName string, expiry time.Duration) (string, error) {
	url, err := m.client.PresignedGetObject(context.Background(), m.bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

func (m *MinioProvider) UploadFromReader(reader io.Reader, objectName, contentType string, size int64) (*UploadedFile, error) {
	_, err := m.client.PutObject(context.Background(), m.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &UploadedFile{
		ID:          uuid.New().String(),
		Name:        filepath.Base(objectName),
		URL:         m.publicURL + "/" + objectName,
		Size:        size,
		ContentType: contentType,
		ObjectName:  objectName,
	}, nil
}
