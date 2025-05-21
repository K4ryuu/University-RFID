package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string

	EnableRESTAPI   bool
	EnableWebsocket bool

	APIKeyRequired bool
	APIKeys        []string

	DBPath string

	JWTSecret     string
	EncryptionKey string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Figyelmeztetés: .env fájl nem található")
	}

	config := &Config{
		Port: getEnv("PORT", "8080"),

		EnableRESTAPI:   getBoolEnv("ENABLE_REST_API", true),
		EnableWebsocket: getBoolEnv("ENABLE_WEBSOCKET", false),

		APIKeyRequired: getBoolEnv("API_KEY_REQUIRED", false),
		APIKeys:        getStringSliceEnv("API_KEYS", []string{}),

		DBPath: getEnv("DB_PATH", "rfid.db"),

		JWTSecret:     getEnv("JWT_SECRET", "32-karakter-aes-kulcs-ide12345678"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "12345678901234567890123456789012"),
	}

	return config
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return boolValue
}

func getStringSliceEnv(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return strings.Split(value, ",")
}
