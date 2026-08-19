package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr    string
	MySQLDSN    string
	AutoMigrate bool
	RedisAddr   string
	CacheTTL    time.Duration
	JWTSecret   []byte
	JWTTTL      time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    env("HTTP_ADDR", ":8080"),
		MySQLDSN:    env("MYSQL_DSN", ""),
		AutoMigrate: boolean("MYSQL_AUTO_MIGRATE", true),
		RedisAddr:   env("REDIS_ADDR", "127.0.0.1:6379"),
		CacheTTL:    5 * time.Minute,
		JWTSecret:   []byte(env("JWT_SECRET", "")),
		JWTTTL:      24 * time.Hour,
	}

	if raw := env("JWT_TTL", ""); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("JWT_TTL: %w", err)
		}
		cfg.JWTTTL = d
	}
	if raw := env("CACHE_TASK_LIST_TTL", ""); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("CACHE_TASK_LIST_TTL: %w", err)
		}
		cfg.CacheTTL = d
	}

	if cfg.MySQLDSN == "" {
		return Config{}, errors.New("MYSQL_DSN is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must be at least 32 characters")
	}
	if cfg.JWTTTL <= 0 {
		return Config{}, errors.New("JWT_TTL must be positive")
	}
	if cfg.CacheTTL <= 0 {
		return Config{}, errors.New("CACHE_TASK_LIST_TTL must be positive")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func boolean(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}
