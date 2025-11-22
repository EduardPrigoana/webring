package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL   string
	WebringURL    string
	Port          string
	CheckCooldown time.Duration
	FrontendRepo  string
	SiteTitle     string
	SiteFavicon   string
}

func Load() *Config {
	cooldownMins := getEnvInt("CHECK_COOLDOWN_MINUTES", 60)

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		WebringURL:    getEnv("WEBRING_URL", "http://localhost:8080"),
		Port:          getEnv("PORT", "8080"),
		CheckCooldown: time.Duration(cooldownMins) * time.Minute,
		FrontendRepo:  getEnv("FRONTEND_REPO", ""),
		SiteTitle:     getEnv("SITE_TITLE", "webring"),
		SiteFavicon:   getEnv("SITE_FAVICON", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}
