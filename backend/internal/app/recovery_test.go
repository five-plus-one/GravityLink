package app

import (
	"context"
	"testing"
	"time"
)

func TestStartupRetryStopsOnSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	calls := 0
	retryStartup(ctx, time.Millisecond, func() bool { calls++; return calls == 3 })
	if calls != 3 {
		t.Fatalf("attempts=%d", calls)
	}
}
func TestStartupRetryCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	retryStartup(ctx, time.Millisecond, func() bool { t.Fatal("attempt after cancellation"); return false })
}
