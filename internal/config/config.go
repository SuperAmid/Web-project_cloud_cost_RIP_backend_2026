// Package config читает настройки сервиса из переменных окружения.
package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr           string // адрес HTTP-сервера
	DatabaseURL    string // DSN PostgreSQL
	MinioEndpoint  string // host:port MinIO (S3 API)
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioPublicURL string // базовый URL, по которому клиент открывает файлы
	RedisAddr      string // host:port Redis (blacklist JWT)
	RedisPassword  string
	JWTSecret      string        // секрет подписи JWT (HS256)
	JWTTTL         time.Duration // время жизни токена
}

func FromEnv() Config {
	return Config{
		Addr:           envOr("APP_ADDR", ":8080"),
		DatabaseURL:    envOr("DATABASE_URL", "host=127.0.0.1 user=provider_routers_user password=provider_routers_password dbname=provider_routers port=5433 sslmode=disable"),
		MinioEndpoint:  envOr("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: envOr("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: envOr("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    envOr("MINIO_BUCKET", "provider-media"),
		MinioPublicURL: strings.TrimRight(envOr("MINIO_PUBLIC_URL", "http://localhost:9000/provider-media"), "/"),
		RedisAddr:      envOr("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  envOr("REDIS_PASSWORD", "provider_routers_redis_password"),
		JWTSecret:      envOr("JWT_SECRET", "provider_routers_dev_jwt_secret"),
		JWTTTL:         durationOr("JWT_TTL", time.Hour),
	}
}

func durationOr(key string, fallback time.Duration) time.Duration {
	if value, err := time.ParseDuration(os.Getenv(key)); err == nil && value > 0 {
		return value
	}
	return fallback
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
