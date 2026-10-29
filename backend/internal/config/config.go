// Package config membaca konfigurasi dari environment variable.
package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	Env        string
	MongoURI   string
	MongoDB    string
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func Load() (*Config, error) {
	// Abaikan error: di produksi variabel biasanya diset langsung, bukan lewat file .env
	_ = godotenv.Load()

	cfg := &Config{
		Port:       getEnv("PORT", "8080"),
		Env:        getEnv("APP_ENV", "development"),
		MongoURI:   os.Getenv("MONGO_URI"),
		MongoDB:    getEnv("MONGO_DB", "fintrack"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
		AccessTTL:  time.Duration(getInt("ACCESS_TTL_MINUTES", 15)) * time.Minute,
		RefreshTTL: time.Duration(getInt("REFRESH_TTL_DAYS", 30)) * 24 * time.Hour,
	}

	if cfg.MongoURI == "" {
		return nil, errors.New("MONGO_URI wajib diisi")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, errors.New("JWT_SECRET wajib diisi dan minimal 32 karakter")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}
