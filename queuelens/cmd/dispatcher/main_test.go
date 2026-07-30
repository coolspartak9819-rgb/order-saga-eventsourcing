package main

import (
	"testing"
	"time"
)

func TestRetryDelayUsesExponentialBackoff(t *testing.T) {
	if got := retryDelay(1); got != 2*time.Second {
		t.Fatalf("got %s, want 2s", got)
	}
	if got := retryDelay(3); got != 8*time.Second {
		t.Fatalf("got %s, want 8s", got)
	}
	if got := retryDelay(99); got != 64*time.Second {
		t.Fatalf("got %s, want 64s", got)
	}
}
