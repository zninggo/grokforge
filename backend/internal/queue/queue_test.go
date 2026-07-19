package queue

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNilQueueSafe(t *testing.T) {
	q := New(nil)
	if q.Available() {
		t.Fatal("expected unavailable")
	}
	ctx := context.Background()
	if err := q.Ping(ctx); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ping: %v", err)
	}
	if err := q.Enqueue(ctx, "noop", 1); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("enqueue: %v", err)
	}
	if _, _, err := q.BRPop(ctx, time.Millisecond); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("brpop: %v", err)
	}
	if _, err := q.TryLock(ctx, 1, time.Second); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("lock: %v", err)
	}
	if err := q.Unlock(ctx, 1); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unlock: %v", err)
	}
	if err := q.Publish(ctx, "x"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("publish: %v", err)
	}
	if q.Subscribe(ctx) != nil {
		t.Fatal("subscribe should be nil")
	}
}
