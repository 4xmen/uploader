package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                   string
	MaxUploadSize          int64
	MaxAllowedLifetimeDays int
	AdminSecret            string
	DatabasePath           string
	StoragePath            string
	AllowedOrigins         []string
}

var AppConfig Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, falling back to environment variables")
	}

	AppConfig = Config{
		Port:                   getEnv("PORT", "8080"),
		MaxUploadSize:          getEnvAsInt64("MAX_UPLOAD_SIZE_MB", 50) * 1024 * 1024,
		MaxAllowedLifetimeDays: getEnvAsInt("MAX_ALLOWED_LIFETIME_DAYS", 7),
		AdminSecret:            getEnv("ADMIN_SECRET", "admin123"),
		DatabasePath:           getEnv("DATABASE_PATH", "uploader.db"),
		StoragePath:            getEnv("STORAGE_PATH", "./uploads"),
		AllowedOrigins:         getEnvAsSlice("ALLOWED_ORIGINS", []string{"*"}),
	}

	// Ensure storage path exists
	if err := os.MkdirAll(AppConfig.StoragePath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}
}

func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsInt64(name string, defaultVal int64) int64 {
	valueStr := getEnv(name, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsSlice(name string, defaultVal []string) []string {
	valueStr := getEnv(name, "")
	if valueStr == "" {
		return defaultVal
	}
	parts := strings.Split(valueStr, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultVal
	}
	return result
}
