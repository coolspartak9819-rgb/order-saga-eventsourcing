package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/coolspartak9819-rgb/order-saga-eventsourcing/internal/domain"
	"github.com/lib/pq"
)

const uniqueViolationCode = "23505"

var (
	ErrDatabaseRequired   = errors.New("database is required")
	ErrNilDomainEvent     = errors.New("domain event is nil")
	ErrUnknownEventType   = errors.New("unknown event type")
	ErrEventPayloadDecode = errors.New("event payload decode failed")
)

type PostgresEventStore struct {
	db *sql.DB
}

func NewPostgresEventStore(db *sql.DB) *PostgresEventStore {
	return &PostgresEventStore{db: db}
}

func (s *PostgresEventStore) SaveEvents(ctx context.Context, aggregateID string, events []domain.DomainEvent, expectedVersion int) error {
	if s.db == nil {
		return ErrDatabaseRequired
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	currentVersion, err := currentAggregateVersion(ctx, tx, aggregateID)
	if err != nil {
		return err
	}
	if expectedVersion != currentVersion {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("%w: rollback: %v", domain.ErrConcurrencyConflict, rollbackErr)
		}
		tx = nil
		return domain.ErrConcurrencyConflict
	}

	for i, event := range events {
		if event == nil {
			return ErrNilDomainEvent
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", event.EventType(), err)
		}

		version := expectedVersion + i + 1
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO events (aggregate_id, event_type, payload, version, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			aggregateID,
			event.EventType(),
			payload,
			version,
			time.Now().UTC(),
		); err != nil {
			if isUniqueViolation(err) {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					return fmt.Errorf("%w: rollback: %v", domain.ErrConcurrencyConflict, rollbackErr)
				}
				tx = nil
				return domain.ErrConcurrencyConflict
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConcurrencyConflict
		}
		return err
	}
	tx = nil

	return nil
}

func (s *PostgresEventStore) LoadEvents(ctx context.Context, aggregateID string) ([]domain.DomainEvent, error) {
	if s.db == nil {
		return nil, ErrDatabaseRequired
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT event_type, payload
		 FROM events
		 WHERE aggregate_id = $1
		 ORDER BY version ASC`,
		aggregateID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.DomainEvent
	for rows.Next() {
		var eventType string
		var payload []byte
		if err := rows.Scan(&eventType, &payload); err != nil {
			return nil, err
		}

		event, err := decodeDomainEvent(eventType, payload)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func currentAggregateVersion(ctx context.Context, tx *sql.Tx, aggregateID string) (int, error) {
	var version int
	err := tx.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(version), 0)
		 FROM events
		 WHERE aggregate_id = $1`,
		aggregateID,
	).Scan(&version)
	if err != nil {
		return 0, err
	}

	return version, nil
}

func decodeDomainEvent(eventType string, payload []byte) (domain.DomainEvent, error) {
	switch eventType {
	case domain.EventTypeOrderCreated:
		var event domain.OrderCreatedEvent
		if err := decodePayload(payload, &event); err != nil {
			return nil, err
		}
		return event, nil
	case domain.EventTypePaymentAuthorized:
		var event domain.PaymentAuthorizedEvent
		if err := decodePayload(payload, &event); err != nil {
			return nil, err
		}
		return event, nil
	case domain.EventTypePaymentFailed:
		var event domain.PaymentFailedEvent
		if err := decodePayload(payload, &event); err != nil {
			return nil, err
		}
		return event, nil
	case domain.EventTypeInventoryReserved:
		var event domain.InventoryReservedEvent
		if err := decodePayload(payload, &event); err != nil {
			return nil, err
		}
		return event, nil
	case domain.EventTypeInventoryFailed:
		var event domain.InventoryFailedEvent
		if err := decodePayload(payload, &event); err != nil {
			return nil, err
		}
		return event, nil
	case domain.EventTypeOrderCompleted:
		var event domain.OrderCompletedEvent
		if err := decodePayload(payload, &event); err != nil {
			return nil, err
		}
		return event, nil
	case domain.EventTypeOrderCancelled:
		var event domain.OrderCancelledEvent
		if err := decodePayload(payload, &event); err != nil {
			return nil, err
		}
		return event, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownEventType, eventType)
	}
}

func decodePayload(payload []byte, event domain.DomainEvent) error {
	if err := json.Unmarshal(payload, event); err != nil {
		return fmt.Errorf("%w: %v", ErrEventPayloadDecode, err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && string(pqErr.Code) == uniqueViolationCode
}

var _ domain.EventStore = (*PostgresEventStore)(nil)
