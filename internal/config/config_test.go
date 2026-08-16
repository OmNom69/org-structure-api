package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadRedisDefaults(t *testing.T) {
	clearRedisEnvironment(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisEnabled {
		t.Fatal("RedisEnabled = true, want false")
	}

	if cfg.RedisAddr != defaultRedisAddr {
		t.Fatalf("RedisAddr = %q, want %q", cfg.RedisAddr, defaultRedisAddr)
	}

	if cfg.RedisPassword != "" {
		t.Fatal("RedisPassword is not empty")
	}

	if cfg.RedisDB != defaultRedisDB {
		t.Fatalf("RedisDB = %d, want %d", cfg.RedisDB, defaultRedisDB)
	}

	if cfg.RedisDialTimeout != defaultRedisDialTimeout {
		t.Fatalf("RedisDialTimeout = %s, want %s", cfg.RedisDialTimeout, defaultRedisDialTimeout)
	}

	if cfg.RedisReadTimeout != defaultRedisReadTimeout {
		t.Fatalf("RedisReadTimeout = %s, want %s", cfg.RedisReadTimeout, defaultRedisReadTimeout)
	}

	if cfg.RedisWriteTimeout != defaultRedisWriteTimeout {
		t.Fatalf("RedisWriteTimeout = %s, want %s", cfg.RedisWriteTimeout, defaultRedisWriteTimeout)
	}

	if cfg.CacheTTL != defaultCacheTTL {
		t.Fatalf("CacheTTL = %s, want %s", cfg.CacheTTL, defaultCacheTTL)
	}
}

func TestLoadRedisEnabled(t *testing.T) {
	setValidRedisEnvironment(t)
	t.Setenv("REDIS_ADDR", " cache.internal:6380 ")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("REDIS_DB", "4")
	t.Setenv("REDIS_DIAL_TIMEOUT", "750ms")
	t.Setenv("REDIS_READ_TIMEOUT", "300ms")
	t.Setenv("REDIS_WRITE_TIMEOUT", "400ms")
	t.Setenv("CACHE_TTL", "2m30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.RedisEnabled {
		t.Fatal("RedisEnabled = false, want true")
	}

	if cfg.RedisAddr != "cache.internal:6380" {
		t.Fatalf("RedisAddr = %q, want %q", cfg.RedisAddr, "cache.internal:6380")
	}

	if cfg.RedisPassword != "secret" {
		t.Fatal("RedisPassword does not match configured value")
	}

	if cfg.RedisDB != 4 {
		t.Fatalf("RedisDB = %d, want 4", cfg.RedisDB)
	}

	if cfg.RedisDialTimeout != 750*time.Millisecond {
		t.Fatalf("RedisDialTimeout = %s, want 750ms", cfg.RedisDialTimeout)
	}

	if cfg.RedisReadTimeout != 300*time.Millisecond {
		t.Fatalf("RedisReadTimeout = %s, want 300ms", cfg.RedisReadTimeout)
	}

	if cfg.RedisWriteTimeout != 400*time.Millisecond {
		t.Fatalf("RedisWriteTimeout = %s, want 400ms", cfg.RedisWriteTimeout)
	}

	if cfg.CacheTTL != 150*time.Second {
		t.Fatalf("CacheTTL = %s, want 2m30s", cfg.CacheTTL)
	}
}

func TestLoadRedisDisabledIgnoresRedisOnlyValidation(t *testing.T) {
	clearRedisEnvironment(t)
	t.Setenv("REDIS_ENABLED", "false")
	t.Setenv("REDIS_DB", "not-a-number")
	t.Setenv("REDIS_DIAL_TIMEOUT", "not-a-duration")
	t.Setenv("REDIS_READ_TIMEOUT", "0s")
	t.Setenv("REDIS_WRITE_TIMEOUT", "-1s")
	t.Setenv("CACHE_TTL", "0s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisEnabled {
		t.Fatal("RedisEnabled = true, want false")
	}
}

func TestLoadRejectsInvalidRedisEnabled(t *testing.T) {
	clearRedisEnvironment(t)
	t.Setenv("REDIS_ENABLED", "sometimes")

	_, err := Load()
	assertConfigErrorContains(t, err, "REDIS_ENABLED")
}

func TestLoadRejectsInvalidRedisConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "empty address", key: "REDIS_ADDR", value: " "},
		{name: "database is not a number", key: "REDIS_DB", value: "abc"},
		{name: "database is negative", key: "REDIS_DB", value: "-1"},
		{name: "dial timeout is malformed", key: "REDIS_DIAL_TIMEOUT", value: "soon"},
		{name: "dial timeout is zero", key: "REDIS_DIAL_TIMEOUT", value: "0s"},
		{name: "read timeout is negative", key: "REDIS_READ_TIMEOUT", value: "-1s"},
		{name: "write timeout is zero", key: "REDIS_WRITE_TIMEOUT", value: "0s"},
		{name: "cache ttl is malformed", key: "CACHE_TTL", value: "later"},
		{name: "cache ttl is zero", key: "CACHE_TTL", value: "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidRedisEnvironment(t)
			t.Setenv(tt.key, tt.value)

			_, err := Load()
			assertConfigErrorContains(t, err, tt.key)
		})
	}
}

func TestLoadDoesNotExposeRedisPasswordInErrors(t *testing.T) {
	setValidRedisEnvironment(t)
	const password = "do-not-log-this-password"
	t.Setenv("REDIS_PASSWORD", password)
	t.Setenv("CACHE_TTL", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want configuration error")
	}

	if strings.Contains(err.Error(), password) {
		t.Fatalf("Load() error exposes Redis password: %v", err)
	}
}

func clearRedisEnvironment(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"REDIS_ENABLED",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"REDIS_DB",
		"REDIS_DIAL_TIMEOUT",
		"REDIS_READ_TIMEOUT",
		"REDIS_WRITE_TIMEOUT",
		"CACHE_TTL",
	} {
		t.Setenv(key, "")
	}
}

func setValidRedisEnvironment(t *testing.T) {
	t.Helper()
	clearRedisEnvironment(t)
	t.Setenv("REDIS_ENABLED", "true")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("REDIS_DB", "0")
	t.Setenv("REDIS_DIAL_TIMEOUT", "2s")
	t.Setenv("REDIS_READ_TIMEOUT", "1s")
	t.Setenv("REDIS_WRITE_TIMEOUT", "1s")
	t.Setenv("CACHE_TTL", "5m")
}

func assertConfigErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("Load() error = nil, want error containing %q", want)
	}

	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Load() error = %q, want it to contain %q", err, want)
	}
}
