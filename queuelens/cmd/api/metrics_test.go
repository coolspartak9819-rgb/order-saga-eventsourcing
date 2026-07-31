package main

import (
	"testing"
	"time"
)

func TestObserveRequestDuration(t *testing.T) {
	requestLatencyCount.Store(0)
	requestLatencySumMicros.Store(0)
	for index := range requestLatencyBuckets {
		requestLatencyBuckets[index].Store(0)
	}

	observeRequestDuration(3 * time.Millisecond)
	observeRequestDuration(80 * time.Millisecond)
	observeRequestDuration(700 * time.Millisecond)

	if got := requestLatencyCount.Load(); got != 3 {
		t.Fatalf("request count = %d, want 3", got)
	}
	if got := requestLatencySumMicros.Load(); got != 783000 {
		t.Fatalf("latency sum = %d, want 783000", got)
	}
	if got := cumulativeLatencyBucket(0); got != 1 {
		t.Fatalf("5ms bucket = %d, want 1", got)
	}
	if got := cumulativeLatencyBucket(2); got != 2 {
		t.Fatalf("100ms bucket = %d, want 2", got)
	}
	if got := requestLatencyCount.Load(); got != 3 {
		t.Fatalf("infinite bucket = %d, want 3", got)
	}
}
