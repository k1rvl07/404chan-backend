package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBHost          string
	DBPort          string
	DBUser          string
	DBPass          string
	DBName          string
	ServerPort      string
	RedisURL        string
	Env             string
	RedisTTL        time.Duration
	MinioURL        string
	MinioPublicURL  string
	MinioUser       string
	MinioPassword   string
	MinioBucket     string
	MaxFileSize     int64
	MaxFilesPerPost int
}

func LoadConfig() Config {
	ttlStr := getEnv("REDIS_TTL", "5m")
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		ttl = 5 * time.Minute
	}

	maxFileSize := getEnvAsInt64("MAX_FILE_SIZE", 10*1024*1024) // 10MB default
	maxFilesPerPost := getEnvAsInt("MAX_FILES_PER_POST", 5)

	return Config{
		DBHost:          getEnv("DB_HOST", "postgres"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPass:          getEnv("DB_PASSWORD", "password"),
		DBName:          getEnv("DB_NAME", "db_404chan"),
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		RedisURL:        getEnv("REDIS_URL", "redis:6379"),
		Env:             getEnv("ENV", "dev"),
		RedisTTL:        ttl,
		MinioURL:        getEnv("MINIO_URL", "localhost:9000"),
		MinioPublicURL:  getEnv("MINIO_PUBLIC_URL", ""),
		MinioUser:       getEnv("MINIO_USER", "minioadmin"),
		MinioPassword:   getEnv("MINIO_PASSWORD", "minioadmin"),
		MinioBucket:     getEnv("MINIO_BUCKET", "404chan-files"),
		MaxFileSize:     maxFileSize,
		MaxFilesPerPost: maxFilesPerPost,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return fallback
}

func getEnvAsInt64(key string, fallback int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			return v
		}
	}
	return fallback
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		c.DBHost, c.DBUser, c.DBPass, c.DBName, c.DBPort,
	)
}
