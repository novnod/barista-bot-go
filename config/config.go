package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load("config/.env"); err != nil {
		log.Printf("No .env file found, relying on environment variables")
	}

	config := &Config{
		BotToken: getEnv("DISCORD_TOKEN", ""),
	}
	return config, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
