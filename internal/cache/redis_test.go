package cache

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisClientFake struct {
	getValue  string
	getErr    error
	setErr    error
	incrValue int64
	incrErr   error
	pingErr   error
	closeErr  error

	getKeys  []string
	getCtx   context.Context
	setKeys  []string
	setData  [][]byte
	setTTLs  []time.Duration
	setCtx   context.Context
	incrKeys []string
	incrCtx  context.Context
	pingCtx  context.Context
	closed   int
}

func (f *redisClientFake) Get(ctx context.Context, key string) *redis.StringCmd {
	f.getCtx = ctx
	f.getKeys = append(f.getKeys, key)

	return redis.NewStringResult(f.getValue, f.getErr)
}

func (f *redisClientFake) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	f.setCtx = ctx
	f.setKeys = append(f.setKeys, key)
	f.setTTLs = append(f.setTTLs, ttl)

	if data, ok := value.([]byte); ok {
		f.setData = append(f.setData, append([]byte(nil), data...))
	}

	return redis.NewStatusResult("OK", f.setErr)
}

func (f *redisClientFake) Incr(ctx context.Context, key string) *redis.IntCmd {
	f.incrCtx = ctx
	f.incrKeys = append(f.incrKeys, key)

	return redis.NewIntResult(f.incrValue, f.incrErr)
}

func (f *redisClientFake) Ping(ctx context.Context) *redis.StatusCmd {
	f.pingCtx = ctx

	return redis.NewStatusResult("PONG", f.pingErr)
}

func (f *redisClientFake) Close() error {
	f.closed++

	return f.closeErr
}

func TestRedisStoreGet(t *testing.T) {
	clientErr := errors.New("connection lost")
	tests := []struct {
		name      string
		value     string
		redisErr  error
		wantValue []byte
		wantFound bool
		wantErr   bool
	}{
		{
			name:      "hit",
			value:     `{"id":1}`,
			wantValue: []byte(`{"id":1}`),
			wantFound: true,
		},
		{
			name:      "empty value is a hit",
			value:     "",
			wantValue: []byte{},
			wantFound: true,
		},
		{
			name:     "miss",
			redisErr: redis.Nil,
		},
		{
			name:     "client error",
			redisErr: clientErr,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &redisClientFake{
				getValue: tt.value,
				getErr:   tt.redisErr,
			}
			store := newRedisStoreForTest(client)

			value, found, err := store.Get(context.Background(), "department-tree:1")
			if tt.wantErr {
				if !errors.Is(err, tt.redisErr) {
					t.Fatalf("Get() error = %v, want wrapped %v", err, tt.redisErr)
				}
			} else if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if found != tt.wantFound {
				t.Fatalf("Get() found = %t, want %t", found, tt.wantFound)
			}

			if !bytes.Equal(value, tt.wantValue) {
				t.Fatalf("Get() value = %q, want %q", value, tt.wantValue)
			}

			if len(client.getKeys) != 1 || client.getKeys[0] != "department-tree:1" {
				t.Fatalf("Get() keys = %v, want [department-tree:1]", client.getKeys)
			}

			if _, ok := client.getCtx.Deadline(); !ok {
				t.Fatal("Get() context has no operation deadline")
			}
		})
	}
}

func TestRedisOptions(t *testing.T) {
	options := Options{
		Addr:         "cache.internal:6380",
		Password:     "secret",
		DB:           4,
		DialTimeout:  750 * time.Millisecond,
		ReadTimeout:  300 * time.Millisecond,
		WriteTimeout: 400 * time.Millisecond,
	}

	got := redisOptions(options)
	if got.Addr != options.Addr ||
		got.Password != options.Password ||
		got.DB != options.DB ||
		got.DialTimeout != options.DialTimeout ||
		got.ReadTimeout != options.ReadTimeout ||
		got.WriteTimeout != options.WriteTimeout {
		t.Fatalf("redisOptions() did not preserve configured connection options")
	}

	if got.PoolTimeout != options.ReadTimeout {
		t.Fatalf("PoolTimeout = %s, want %s", got.PoolTimeout, options.ReadTimeout)
	}

	if got.MaxRetries != -1 {
		t.Fatalf("MaxRetries = %d, want -1", got.MaxRetries)
	}

	if got.DialerRetries != 1 {
		t.Fatalf("DialerRetries = %d, want 1", got.DialerRetries)
	}

	if !got.ContextTimeoutEnabled {
		t.Fatal("ContextTimeoutEnabled = false, want true")
	}
}

func TestRedisStoreSet(t *testing.T) {
	client := &redisClientFake{}
	store := newRedisStoreForTest(client)
	value := []byte(`{"id":1}`)

	err := store.Set(context.Background(), "department-tree:1", value, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if len(client.setKeys) != 1 || client.setKeys[0] != "department-tree:1" {
		t.Fatalf("Set() keys = %v, want [department-tree:1]", client.setKeys)
	}

	if len(client.setData) != 1 || !bytes.Equal(client.setData[0], value) {
		t.Fatalf("Set() data = %q, want %q", client.setData, value)
	}

	if len(client.setTTLs) != 1 || client.setTTLs[0] != 5*time.Minute {
		t.Fatalf("Set() TTLs = %v, want [5m0s]", client.setTTLs)
	}

	if _, ok := client.setCtx.Deadline(); !ok {
		t.Fatal("Set() context has no operation deadline")
	}
}

func TestRedisStoreSetWrapsClientError(t *testing.T) {
	clientErr := errors.New("write failed")
	store := newRedisStoreForTest(&redisClientFake{setErr: clientErr})

	err := store.Set(context.Background(), "key", []byte("value"), time.Minute)
	if !errors.Is(err, clientErr) {
		t.Fatalf("Set() error = %v, want wrapped %v", err, clientErr)
	}
}

func TestRedisStoreSetRejectsNonPositiveTTL(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Second} {
		client := &redisClientFake{}
		store := newRedisStoreForTest(client)

		err := store.Set(context.Background(), "key", []byte("value"), ttl)
		if err == nil {
			t.Fatalf("Set() with TTL %s error = nil, want validation error", ttl)
		}

		if len(client.setKeys) != 0 {
			t.Fatalf("Set() with TTL %s called Redis", ttl)
		}
	}
}

func TestRedisStoreIncrement(t *testing.T) {
	client := &redisClientFake{incrValue: 12}
	store := newRedisStoreForTest(client)

	value, err := store.Increment(context.Background(), "department-tree:epoch")
	if err != nil {
		t.Fatalf("Increment() error = %v", err)
	}

	if value != 12 {
		t.Fatalf("Increment() value = %d, want 12", value)
	}

	if len(client.incrKeys) != 1 || client.incrKeys[0] != "department-tree:epoch" {
		t.Fatalf("Increment() keys = %v, want [department-tree:epoch]", client.incrKeys)
	}

	if _, ok := client.incrCtx.Deadline(); !ok {
		t.Fatal("Increment() context has no operation deadline")
	}
}

func TestRedisStoreIncrementWrapsClientError(t *testing.T) {
	clientErr := errors.New("increment failed")
	store := newRedisStoreForTest(&redisClientFake{incrErr: clientErr})

	value, err := store.Increment(context.Background(), "department-tree:epoch")
	if !errors.Is(err, clientErr) {
		t.Fatalf("Increment() error = %v, want wrapped %v", err, clientErr)
	}

	if value != 0 {
		t.Fatalf("Increment() value = %d, want 0", value)
	}
}

func TestRedisStorePingAndClose(t *testing.T) {
	client := &redisClientFake{}
	store := newRedisStoreForTest(client)
	ctx := context.Background()

	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	if client.pingCtx != ctx {
		t.Fatal("Ping() did not forward context")
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if client.closed != 1 {
		t.Fatalf("Close() calls = %d, want 1", client.closed)
	}
}

func TestRedisStorePingAndCloseWrapClientErrors(t *testing.T) {
	pingErr := errors.New("ping failed")
	closeErr := errors.New("close failed")
	client := &redisClientFake{
		pingErr:  pingErr,
		closeErr: closeErr,
	}
	store := newRedisStoreForTest(client)

	if err := store.Ping(context.Background()); !errors.Is(err, pingErr) {
		t.Fatalf("Ping() error = %v, want wrapped %v", err, pingErr)
	}

	if err := store.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Close() error = %v, want wrapped %v", err, closeErr)
	}
}

func newRedisStoreForTest(client redisClient) *RedisStore {
	return &RedisStore{
		client:       client,
		readTimeout:  time.Second,
		writeTimeout: time.Second,
	}
}
