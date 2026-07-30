package events

import "testing"

func TestNewPublisherRequiresConfiguration(t *testing.T) {
	if _, err := NewPublisher("", "job-events"); err == nil {
		t.Fatal("expected brokers validation error")
	}
	if _, err := NewPublisher("kafka:9092", ""); err == nil {
		t.Fatal("expected topic validation error")
	}
}
