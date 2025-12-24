package config

import (
	"errors"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL            string `env:"DATABASE_URL,required"`
	Port                   int    `env:"PORT" envDefault:"8080"`
	SMTPHost               string `env:"SMTP_HOST"`
	SMTPPort               int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser               string `env:"SMTP_USER"`
	SMTPPass               string `env:"SMTP_PASS"`
	SMTPFrom               string `env:"SMTP_FROM" envDefault:"no-reply@timesync"`
	AccessTTLMinutes       int    `env:"ACCESS_TTL_MINUTES" envDefault:"30"`
	RefreshTTLHours        int    `env:"REFRESH_TTL_HOURS" envDefault:"720"`
	CodeTTLMinutes         int    `env:"CODE_TTL_MINUTES" envDefault:"10"`
	RefreshGraceSeconds    int    `env:"REFRESH_GRACE_SECONDS" envDefault:"30"`
	TeamSizeLimit          int    `env:"TEAM_SIZE_LIMIT" envDefault:"30"`
	RequestCodeEmailLimit  int    `env:"REQUEST_CODE_EMAIL_LIMIT" envDefault:"3"`
	RequestCodeEmailWindow int    `env:"REQUEST_CODE_EMAIL_WINDOW_MINUTES" envDefault:"15"`
	RequestCodeIPLimit     int    `env:"REQUEST_CODE_IP_LIMIT" envDefault:"10"`
	RequestCodeIPWindow    int    `env:"REQUEST_CODE_IP_WINDOW_MINUTES" envDefault:"60"`
	VerifyCodeEmailLimit   int    `env:"VERIFY_CODE_EMAIL_LIMIT" envDefault:"5"`
	VerifyCodeEmailWindow  int    `env:"VERIFY_CODE_EMAIL_WINDOW_MINUTES" envDefault:"15"`
	VerifyCodeLockMinutes  int    `env:"VERIFY_CODE_LOCK_MINUTES" envDefault:"15"`
	VerifyCodeIPLimit      int    `env:"VERIFY_CODE_IP_LIMIT" envDefault:"20"`
	VerifyCodeIPWindow     int    `env:"VERIFY_CODE_IP_WINDOW_MINUTES" envDefault:"60"`
	RefreshDeviceLimit     int    `env:"REFRESH_DEVICE_LIMIT" envDefault:"10"`
	RefreshDeviceWindow    int    `env:"REFRESH_DEVICE_WINDOW_MINUTES" envDefault:"1"`
}

func Load() (Config, error) {
	_ = godotenv.Load()
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}
