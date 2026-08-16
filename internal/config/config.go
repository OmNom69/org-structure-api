package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRedisAddr         = "localhost:6379"
	defaultRedisDB           = 0
	defaultRedisDialTimeout  = 2 * time.Second
	defaultRedisReadTimeout  = time.Second
	defaultRedisWriteTimeout = time.Second
	defaultCacheTTL          = 5 * time.Minute
)

type Config struct {
	Port string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisEnabled      bool
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	RedisDialTimeout  time.Duration
	RedisReadTimeout  time.Duration
	RedisWriteTimeout time.Duration
	CacheTTL          time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port: getEnv("APP_PORT", "8080"),

		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "postgres"),
		DBName:            getEnv("DB_NAME", "org_structure"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		RedisAddr:         getEnv("REDIS_ADDR", defaultRedisAddr),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           defaultRedisDB,
		RedisDialTimeout:  defaultRedisDialTimeout,
		RedisReadTimeout:  defaultRedisReadTimeout,
		RedisWriteTimeout: defaultRedisWriteTimeout,
		CacheTTL:          defaultCacheTTL,
	}

	redisEnabled, err := strconv.ParseBool(getEnv("REDIS_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("REDIS_ENABLED must be a boolean: %w", err)
	}

	cfg.RedisEnabled = redisEnabled
	if !cfg.RedisEnabled {
		return cfg, nil
	}

	cfg.RedisAddr = strings.TrimSpace(cfg.RedisAddr)
	if cfg.RedisAddr == "" {
		return Config{}, fmt.Errorf("REDIS_ADDR must not be empty")
	}

	redisDB, err := parseNonNegativeInt("REDIS_DB", getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, err
	}

	redisDialTimeout, err := parsePositiveDuration(
		"REDIS_DIAL_TIMEOUT",
		getEnv("REDIS_DIAL_TIMEOUT", defaultRedisDialTimeout.String()),
	)
	if err != nil {
		return Config{}, err
	}

	redisReadTimeout, err := parsePositiveDuration(
		"REDIS_READ_TIMEOUT",
		getEnv("REDIS_READ_TIMEOUT", defaultRedisReadTimeout.String()),
	)
	if err != nil {
		return Config{}, err
	}

	redisWriteTimeout, err := parsePositiveDuration(
		"REDIS_WRITE_TIMEOUT",
		getEnv("REDIS_WRITE_TIMEOUT", defaultRedisWriteTimeout.String()),
	)
	if err != nil {
		return Config{}, err
	}

	cacheTTL, err := parsePositiveDuration(
		"CACHE_TTL",
		getEnv("CACHE_TTL", defaultCacheTTL.String()),
	)
	if err != nil {
		return Config{}, err
	}

	cfg.RedisDB = redisDB
	cfg.RedisDialTimeout = redisDialTimeout
	cfg.RedisReadTimeout = redisReadTimeout
	cfg.RedisWriteTimeout = redisWriteTimeout
	cfg.CacheTTL = cacheTTL

	return cfg, nil
}

func getEnv(key, defaultValeu string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValeu
	}

	return value
}

func parseNonNegativeInt(key, value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a non-negative integer: %w", key, err)
	}

	if parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}

	return parsed, nil
}

func parsePositiveDuration(key, value string) (time.Duration, error) {
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}

	return parsed, nil
}
