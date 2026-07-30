package events

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type Publisher struct{ writer *kafka.Writer }

type JobEvent struct {
	JobID      string    `json:"job_id"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	Attempts   int       `json:"attempts"`
	Error      string    `json:"error,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

func NewPublisher(brokers, topic string) (*Publisher, error) {
	if strings.TrimSpace(brokers) == "" || strings.TrimSpace(topic) == "" {
		return nil, errors.New("Kafka brokers and topic are required")
	}
	return &Publisher{writer: &kafka.Writer{Addr: kafka.TCP(strings.Split(brokers, ",")...), Topic: topic, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll, BatchTimeout: 50 * time.Millisecond}}, nil
}

func (p *Publisher) Publish(ctx context.Context, event JobEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Key: []byte(event.JobID), Value: payload, Headers: []kafka.Header{{Key: "event_type", Value: []byte(event.Status)}}})
}

func (p *Publisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
