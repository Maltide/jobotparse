package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	LogLevel     string
	ClientID     string
	ClientSecret string
	PgUser       string
	PgPassword   string
	PgDB         string
	PgHost       string
	PgPort       string
	PgSSLMode    string
	BaseURL      string
	AuthUser     string
	AuthPass     string
}

// GetConfig loads configuration from .env and returns a Config.
func GetConfig() (Config, error) {
	// В Docker обычно используется env_file/переменные окружения, поэтому .env внутри контейнера может отсутствовать.
	// Если .env есть — подхватим. Если нет — продолжаем работать только с env vars.
	if _, statErr := os.Stat(".env"); statErr == nil {
		if err := godotenv.Load(".env"); err != nil {
			fmt.Printf("config: failed to load .env: %v\n", err)
		}
	} else if !os.IsNotExist(statErr) {
		fmt.Printf("config: .env stat error: %v\n", statErr)
	}

	return Config{
		LogLevel:     os.Getenv("LOG_LEVEL"),
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		PgUser:       os.Getenv("PG_USER"),
		PgPassword:   os.Getenv("PG_PASSWORD"),
		PgDB:         os.Getenv("PG_DB"),
		PgHost:       os.Getenv("PG_HOST"),
		PgPort:       os.Getenv("PG_PORT"),
		PgSSLMode:    os.Getenv("PG_SSLMODE"),
		BaseURL:      os.Getenv("BASE_URL"),
		AuthUser:     os.Getenv("ADMIN_USER"),
		AuthPass:     os.Getenv("ADMIN_PASS"),
	}, nil
}
