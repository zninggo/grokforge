package yyds

import (
	"context"
	"testing"
	"time"
)

func TestWaitOTPSuccess(t *testing.T) {
	c := New("AC-test-key-not-real")
	calls := 0
	code, err := c.WaitOTP(context.Background(), "a@b.com", WaitOTPConfig{
		TotalTimeout: 10 * time.Second,
		LongPoll:     30 * time.Second,
		RetrySleep:   0,
		Now:          time.Now,
		Sleep: func(ctx context.Context, d time.Duration) error {
			return nil
		},
		Next: func(ctx context.Context, address string, waitSec int) (*Message, bool, error) {
			calls++
			if calls < 2 {
				return nil, false, nil
			}
			return &Message{VerificationCode: "123456"}, true, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != "123456" {
		t.Fatalf("code=%s", code)
	}
	if calls < 2 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestWaitOTPTimeout(t *testing.T) {
	c := New("AC-test")
	start := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	now := start
	code, err := c.WaitOTP(context.Background(), "a@b.com", WaitOTPConfig{
		TotalTimeout: 5 * time.Second,
		LongPoll:     2 * time.Second,
		RetrySleep:   1 * time.Second,
		Now:          func() time.Time { return now },
		Sleep: func(ctx context.Context, d time.Duration) error {
			now = now.Add(d)
			return nil
		},
		Next: func(ctx context.Context, address string, waitSec int) (*Message, bool, error) {
			// simulate long-poll consuming time
			now = now.Add(time.Duration(waitSec) * time.Second)
			return nil, false, nil
		},
	})
	if code != "" {
		t.Fatalf("code=%s", code)
	}
	if err != ErrOTPTimeout {
		t.Fatalf("err=%v", err)
	}
}

func TestMaskKey(t *testing.T) {
	if MaskKey("AC-1234567890abcdef") == "AC-1234567890abcdef" {
		t.Fatal("not masked")
	}
	if MaskKey("") != "" {
		t.Fatal("empty")
	}
}
