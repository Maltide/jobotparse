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
}

func GetConfig() (Config, error) {
	err := godotenv.Load("pkg/.env")
	if err != nil {
		fmt.Printf("config: error loading env: %v", err)
		return Config{}, fmt.Errorf("config: error loading .env file: %v", err)
	}

	return Config{
		LogLevel:     os.Getenv("LOG_LEVEL"),
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
	}, nil
}
