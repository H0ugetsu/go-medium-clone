package config

import (
	"errors"
	"os"
)

type Config struct {
	Port        string
	Environment string

	DB struct {
		URL string
	}
	JWT struct {
		Secret string
	}
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, errors.New("DB_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var cfg Config
	cfg.Environment = env
	cfg.Port = port
	cfg.DB.URL = dbURL
	cfg.JWT.Secret = jwtSecret

	return &cfg, nil
}
