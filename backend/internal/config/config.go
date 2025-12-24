package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	Port        int
}

func Load() (Config, error) {
	_ = loadDotEnv(".env")

	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        8080,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	if portStr := os.Getenv("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return Config{}, err
		}
		cfg.Port = port
	}

	return cfg, nil
}
