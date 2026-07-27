package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	streamName        = "ORDERS"
	ordersSubject     = "orders.events"
	workerInterval    = 2 * time.Second
	outboxStatusReady = "PENDING"
	outboxPublished   = "PUBLISHED"
)

var ErrPublisherDatabaseRequired = errors.New("outbox database is required")

type Publisher struct {
	db *sql.DB
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewPublisher(ctx context.Context, db *sql.DB, natsURL string) (*Publisher, error) {
	if db == nil {
		return nil, ErrPublisherDatabaseRequired
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

	return &Publisher{
		db: db,
		nc: nc,
		js: js,
	}, nil
}

func (p *Publisher) StartWorker(ctx context.Context) {
	ticker := time.NewTicker(workerInterval)
	defer ticker.Stop()

	for {
		if err := p.publishPending(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("outbox worker error: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *Publisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}

func (p *Publisher) Ready() bool {
	return p != nil && p.nc != nil && p.nc.IsConnected() && p.js != nil
}

func (p *Publisher) publishPending(ctx context.Context) error {
	for {
		published, err := p.publishOne(ctx)
		if err != nil {
			return err
		}
		if !published {
			return nil
		}
	}
}

func (p *Publisher) publishOne(ctx context.Context) (bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var event outboxEvent
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, aggregate_id, event_type, payload
		 FROM outbox
		 WHERE status = $1
		 ORDER BY created_at ASC
		 LIMIT 1
		 FOR UPDATE SKIP LOCKED`,
		outboxStatusReady,
	).Scan(&event.ID, &event.AggregateID, &event.EventType, &event.Payload)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	message, err := json.Marshal(event)
	if err != nil {
		return false, err
	}

	if _, err := p.js.Publish(ordersSubject, message, nats.Context(ctx)); err != nil {
		return false, err
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE outbox
		 SET status = $1
		 WHERE id = $2`,
		outboxPublished,
		event.ID,
	); err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return true, nil
}

type outboxEvent struct {
	ID          string          `json:"id"`
	AggregateID string          `json:"aggregate_id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
}
