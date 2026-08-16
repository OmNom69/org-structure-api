package cache

import (
	"context"
	"log/slog"
	"time"

	"github.com/OmNom69/org-structure-api/internal/service"
)

type LoggingStore struct {
	store  service.CacheStore
	logger *slog.Logger
}

var _ service.CacheStore = (*LoggingStore)(nil)

func NewLoggingStore(store service.CacheStore, logger *slog.Logger) *LoggingStore {
	return &LoggingStore{
		store:  store,
		logger: logger,
	}
}

func (s *LoggingStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, found, err := s.store.Get(ctx, key)
	if err != nil {
		s.logError("get", key, err)
	}

	return value, found, err
}

func (s *LoggingStore) Set(
	ctx context.Context,
	key string,
	value []byte,
	ttl time.Duration,
) error {
	err := s.store.Set(ctx, key, value, ttl)
	if err != nil {
		s.logError("set", key, err)
	}

	return err
}

func (s *LoggingStore) Increment(ctx context.Context, key string) (int64, error) {
	value, err := s.store.Increment(ctx, key)
	if err != nil {
		s.logError("increment", key, err)
	}

	return value, err
}

func (s *LoggingStore) logError(operation string, key string, err error) {
	s.logger.Warn(
		"redis cache operation failed",
		slog.String("operation", operation),
		slog.String("key", key),
		slog.Any("error", err),
	)
}
