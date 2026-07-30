package queue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/coolspartak9819-rgb/queuelens/internal/domain"
	"github.com/redis/go-redis/v9"
)

var ErrNoMessage = errors.New("no queue message")

type Redis struct {
	client                  *redis.Client
	stream, group, consumer string
}

func New(addr, stream, group, consumer string) *Redis {
	return &Redis{client: redis.NewClient(&redis.Options{Addr: addr}), stream: stream, group: group, consumer: consumer}
}
func (q *Redis) Close() error                   { return q.client.Close() }
func (q *Redis) Ping(ctx context.Context) error { return q.client.Ping(ctx).Err() }

func (q *Redis) EnsureGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.stream, q.group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (q *Redis) Enqueue(ctx context.Context, job domain.Job) error {
	return q.client.XAdd(ctx, &redis.XAddArgs{Stream: q.stream, Values: map[string]any{"id": job.ID, "type": job.Type, "payload": string(job.Payload)}}).Err()
}

type Message struct {
	ID      string
	JobID   string
	Type    string
	Payload json.RawMessage
}

func (q *Redis) Next(ctx context.Context) (Message, error) {
	entries, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: q.group, Consumer: q.consumer, Streams: []string{q.stream, ">"}, Count: 1, Block: 5 * time.Second}).Result()
	if errors.Is(err, redis.Nil) {
		return Message{}, ErrNoMessage
	}
	if err != nil {
		return Message{}, err
	}
	if len(entries) == 0 || len(entries[0].Messages) == 0 {
		return Message{}, ErrNoMessage
	}
	entry := entries[0].Messages[0]
	job, ok := entry.Values["id"].(string)
	if !ok {
		return Message{}, errors.New("queue message has no id")
	}
	typeName, _ := entry.Values["type"].(string)
	payload, _ := entry.Values["payload"].(string)
	return Message{ID: entry.ID, JobID: job, Type: typeName, Payload: json.RawMessage(payload)}, nil
}

func (q *Redis) Ack(ctx context.Context, id string) error {
	return q.client.XAck(ctx, q.stream, q.group, id).Err()
}
func (q *Redis) DeadLetter(ctx context.Context, message Message, reason string) error {
	return q.client.XAdd(ctx, &redis.XAddArgs{Stream: q.stream + ".dlq", Values: map[string]any{"id": message.JobID, "type": message.Type, "payload": string(message.Payload), "reason": reason}}).Err()
}
