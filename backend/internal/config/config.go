package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strconv"
)

type Config struct {
	// Server
	Port        string
	Environment string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret          string
	JWTAccessExpiry    int // minutes
	JWTRefreshExpiry   int // days

	// Torrent
	DownloadDir     string
	MaxConcurrent   int
	DefaultPort     int

	// Stripe
	StripeSecretKey  string
	StripeWebhookKey string

	// Storage
	StorageType string // local, s3
	S3Bucket    string
	S3Region    string
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "7842"),
		Environment:       getEnv("ENVIRONMENT", "development"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/freetorrent?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6380"),
		JWTSecret:         getJWTSecret(),
		JWTAccessExpiry:   getEnvInt("JWT_ACCESS_EXPIRY", 15),
		JWTRefreshExpiry:  getEnvInt("JWT_REFRESH_EXPIRY", 7),
		DownloadDir:       getEnv("DOWNLOAD_DIR", "./downloads"),
		MaxConcurrent:     getEnvInt("MAX_CONCURRENT", 10),
		DefaultPort:       getEnvInt("TORRENT_PORT", 42069),
		StripeSecretKey:   getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookKey:  getEnv("STRIPE_WEBHOOK_KEY", ""),
		StorageType:       getEnv("STORAGE_TYPE", "local"),
		S3Bucket:          getEnv("S3_BUCKET", ""),
		S3Region:          getEnv("S3_REGION", "us-east-1"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getJWTSecret returns JWT secret from environment or generates a secure one for development
func getJWTSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	
	// In production, require JWT_SECRET to be set
	if os.Getenv("ENVIRONMENT") == "production" {
		log.Fatal("FATAL: JWT_SECRET environment variable is required in production")
	}
	
	// For development, generate a random key and warn
	log.Println("WARNING: JWT_SECRET not set. Generating random key for development. Sessions will not persist across restarts.")
	return generateSecureKey(32)
}

// generateSecureKey generates a cryptographically secure random key
func generateSecureKey(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate secure key:", err)
	}
	return hex.EncodeToString(bytes)
}
