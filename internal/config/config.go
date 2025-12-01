package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPass     string
	DBName     string
	ServerPort string
	RedisURL   string
	Env        string
	RedisTTL   time.Duration
}

func LoadConfig() Config {
	ttlStr := getEnv("REDIS_TTL", "5m")
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		ttl = 5 * time.Minute
	}

	return Config{
		DBHost:     getEnv("DB_HOST", "postgres"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPass:     getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "db_404chan"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
		RedisURL:   getEnv("REDIS_URL", "redis:6379"),
		Env:        getEnv("ENV", "dev"),
		RedisTTL:   ttl,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		c.DBHost, c.DBUser, c.DBPass, c.DBName, c.DBPort,
	)
}
