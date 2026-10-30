// Package config membaca konfigurasi dari environment variable.
package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	Env            string
	MongoURI       string
	MongoDB        string
	JWTSecret      string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
	CORSOrigins    []string
	TrustedProxies []string
	AuthRatePerMin int
	AuthBurst      int
}

func Load() (*Config, error) {
	// Abaikan error: di produksi variabel biasanya diset langsung, bukan lewat file .env
	_ = godotenv.Load()

	env := getEnv("APP_ENV", "development")
	corsDefault := ""
	if env != "production" {
		corsDefault = "http://localhost:8081,http://localhost:19006"
	}

	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		Env:            env,
		MongoURI:       os.Getenv("MONGO_URI"),
		MongoDB:        getEnv("MONGO_DB", "fintrack"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		AccessTTL:      time.Duration(getInt("ACCESS_TTL_MINUTES", 15)) * time.Minute,
		RefreshTTL:     time.Duration(getInt("REFRESH_TTL_DAYS", 30)) * 24 * time.Hour,
		CORSOrigins:    splitCSV(getEnv("CORS_ORIGINS", corsDefault)),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		AuthRatePerMin: getInt("AUTH_RATE_PER_MIN", 10),
		AuthBurst:      getInt("AUTH_BURST", 5),
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

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
