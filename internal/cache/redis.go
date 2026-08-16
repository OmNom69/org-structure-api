package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/OmNom69/org-structure-api/internal/service"
	"github.com/redis/go-redis/v9"
)

type redisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

type RedisStore struct {
	client       redisClient
	readTimeout  time.Duration
	writeTimeout time.Duration
}

type Options struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

var _ service.CacheStore = (*RedisStore)(nil)

func NewRedisStore(options Options) *RedisStore {
	return &RedisStore{
		client:       redis.NewClient(redisOptions(options)),
		readTimeout:  options.ReadTimeout,
		writeTimeout: options.WriteTimeout,
	}
}

func redisOptions(options Options) *redis.Options {
	return &redis.Options{
		Addr:                  options.Addr,
		Password:              options.Password,
		DB:                    options.DB,
		DialTimeout:           options.DialTimeout,
		ReadTimeout:           options.ReadTimeout,
		WriteTimeout:          options.WriteTimeout,
		PoolTimeout:           options.ReadTimeout,
		MaxRetries:            -1,
		DialerRetries:         1,
		ContextTimeoutEnabled: true,
	}
}

func (s *RedisStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	operationContext, cancel := context.WithTimeout(ctx, s.readTimeout)
	defer cancel()

	value, err := s.client.Get(operationContext, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("redis get: %w", err)
	}

	return value, true, nil
}

func (s *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("cache ttl must be greater than zero")
	}

	operationContext, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()

	if err := s.client.Set(operationContext, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}

	return nil
}

func (s *RedisStore) Increment(ctx context.Context, key string) (int64, error) {
	operationContext, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()

	value, err := s.client.Incr(operationContext, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis increment: %w", err)
	}

	return value, nil
}

func (s *RedisStore) Ping(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}

	return nil
}

func (s *RedisStore) Close() error {
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("close redis client: %w", err)
	}

	return nil
}
