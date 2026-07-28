package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	kafka "github.com/segmentio/kafka-go"
)

const (
	streamName   = "ORDERS"
	natsSubject  = "orders.events"
	durableName  = "kafka-event-bridge"
	defaultTopic = "orders.events"
)

// Bridge forwards committed order events from NATS JetStream to Kafka.
// A NATS message is acknowledged only after Kafka confirms the write.
type Bridge struct {
	nc     *nats.Conn
	sub    *nats.Subscription
	writer *kafka.Writer
}

type eventKey struct {
	AggregateID string `json:"aggregate_id"`
	EventType   string `json:"event_type"`
}

func NewBridge(ctx context.Context, natsURL, brokers, topic string) (*Bridge, error) {
	if strings.TrimSpace(natsURL) == "" || strings.TrimSpace(brokers) == "" {
		return nil, errors.New("nats URL and Kafka brokers are required")
	}
	if strings.TrimSpace(topic) == "" {
		topic = defaultTopic
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream(nats.Context(ctx))
	if err != nil {
		nc.Close()
		return nil, err
	}
	if _, err := js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{"orders.*"},
	}); err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		nc.Close()
		return nil, err
	}
	sub, err := js.PullSubscribe(natsSubject, durableName, nats.BindStream(streamName))
	if err != nil {
		nc.Close()
		return nil, err
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(brokers, ",")...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		BatchTimeout: 100 * time.Millisecond,
	}
	return &Bridge{nc: nc, sub: sub, writer: writer}, nil
}

func (b *Bridge) Start(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		messages, err := b.sub.Fetch(1, nats.MaxWait(time.Second))
		if errors.Is(err, nats.ErrTimeout) {
			continue
		}
		if err != nil {
			return err
		}
		for _, message := range messages {
			if err := b.forward(ctx, message); err != nil {
				return err
			}
		}
	}
}

func (b *Bridge) forward(ctx context.Context, message *nats.Msg) error {
	var key eventKey
	if err := json.Unmarshal(message.Data, &key); err != nil {
		return err
	}
	if err := b.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key.AggregateID),
		Value: message.Data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(key.EventType)},
		},
	}); err != nil {
		return err
	}
	return message.Ack()
}

func (b *Bridge) Close() error {
	if b == nil {
		return nil
	}
	if b.sub != nil {
		_ = b.sub.Unsubscribe()
	}
	if b.nc != nil {
		b.nc.Close()
	}
	if b.writer != nil {
		return b.writer.Close()
	}
	return nil
}
