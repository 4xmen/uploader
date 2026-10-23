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
	TrustProxy             bool
}

var AppConfig Config

func LoadConfig() {
	_ = godotenv.Load()

	AppConfig = Config{
		Port:                   getEnv("PORT", "8080"),
		MaxUploadSize:          getEnvAsInt64("MAX_UPLOAD_SIZE_MB", 50) * 1024 * 1024,
		MaxAllowedLifetimeDays: getEnvAsInt("MAX_ALLOWED_LIFETIME_DAYS", 7),
		AdminSecret:            getEnv("ADMIN_SECRET", ""),
		DatabasePath:           getEnv("DATABASE_PATH", "uploader.db"),
		StoragePath:            getEnv("STORAGE_PATH", "./uploads"),
		AllowedOrigins:         getEnvAsSlice("ALLOWED_ORIGINS", []string{"*"}),
		TrustProxy:             getEnvAsBool("TRUST_PROXY", false),
	}

	if AppConfig.AdminSecret == "" {
		log.Println("WARNING: ADMIN_SECRET is not set. Admin endpoints will be disabled.")
	} else if AppConfig.AdminSecret == "admin123" {
		log.Println("WARNING: Insecure default ADMIN_SECRET 'admin123' is being used.")
	}

	if err := os.MkdirAll(AppConfig.StoragePath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}
}

func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return fallback
}

func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if valStr == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultVal
	}
	return parsed
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
