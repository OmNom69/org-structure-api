package cache

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type failingCacheStore struct {
	err error
}

func (s *failingCacheStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s *failingCacheStore) Set(context.Context, string, []byte, time.Duration) error {
	return s.err
}

func (s *failingCacheStore) Increment(context.Context, string) (int64, error) {
	return 0, s.err
}

func TestLoggingStoreLogsAndReturnsCacheErrors(t *testing.T) {
	cacheErr := errors.New("cache unavailable")

	var output bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&output, nil))
	store := NewLoggingStore(&failingCacheStore{err: cacheErr}, logger)

	_, _, getErr := store.Get(context.Background(), "get-key")

	setErr := store.Set(context.Background(), "set-key", []byte("sensitive-cache-payload"), time.Minute)

	_, incrementErr := store.Increment(context.Background(), "epoch-key")

	for operation, err := range map[string]error{
		"get":       getErr,
		"set":       setErr,
		"increment": incrementErr,
	} {
		if !errors.Is(err, cacheErr) {
			t.Fatalf("%s error = %v, want %v", operation, err, cacheErr)
		}

		if !strings.Contains(output.String(), "operation="+operation) {
			t.Errorf("log does not contain operation %q: %s", operation, output.String())
		}
	}

	for _, key := range []string{"get-key", "set-key", "epoch-key"} {
		if !strings.Contains(output.String(), "key="+key) {
			t.Errorf("log does not contain key %q: %s", key, output.String())
		}
	}

	if strings.Count(output.String(), "level=WARN") != 3 {
		t.Errorf("warning count = %d, want 3: %s", strings.Count(output.String(), "level=WARN"), output.String())
	}
	if strings.Contains(output.String(), "sensitive-cache-payload") {
		t.Errorf("log contains cache payload: %s", output.String())
	}
}
