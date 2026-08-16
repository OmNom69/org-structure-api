package main

import (
	"io"
	"log/slog"
	"testing"

	"github.com/OmNom69/org-structure-api/internal/config"
)

func TestSetupRedisDisabledDoesNotCreateStore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	store := setupRedis(logger, config.Config{RedisEnabled: false})
	if store != nil {
		t.Fatal("setupRedis() returned a store while Redis is disabled")
	}
}

func TestCacheStoreForRedisDoesNotCreateTypedNilInterface(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	store := cacheStoreForRedis(nil, logger)
	if store != nil {
		t.Fatalf("cacheStoreForRedis(nil) = %#v, want nil interface", store)
	}
}
