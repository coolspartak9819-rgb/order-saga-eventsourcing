package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/coolspartak9819-rgb/queuelens/internal/domain"
	"github.com/jackc/pgx/v5"
)

type OutboxMessage struct {
	ID       int64
	Job      domain.Job
	Attempts int
}

func (s *Postgres) CreateWithOutbox(ctx context.Context, job domain.Job, idempotencyKey string) (string, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx)
	if idempotencyKey != "" {
		var existingID string
		if err := tx.QueryRow(ctx, `SELECT job_id FROM idempotency_keys WHERE key = $1`, idempotencyKey).Scan(&existingID); err == nil {
			if err := tx.Commit(ctx); err != nil {
				return "", false, err
			}
			return existingID, false, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return "", false, err
		}
	}

	if _, err := tx.Exec(ctx, `INSERT INTO jobs (id, job_type, payload, status) VALUES ($1, $2, $3, $4)`, job.ID, job.Type, job.Payload, job.Status); err != nil {
		return "", false, err
	}
	payload, err := json.Marshal(map[string]any{"id": job.ID, "type": job.Type, "payload": json.RawMessage(job.Payload)})
	if err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outbox (job_id, payload) VALUES ($1, $2)`, job.ID, payload); err != nil {
		return "", false, err
	}
	if idempotencyKey != "" {
		result, err := tx.Exec(ctx, `INSERT INTO idempotency_keys (key, job_id) VALUES ($1, $2) ON CONFLICT (key) DO NOTHING`, idempotencyKey, job.ID)
		if err != nil {
			return "", false, err
		}
		if result.RowsAffected() == 0 {
			if err := tx.Rollback(ctx); err != nil {
				return "", false, err
			}
			return s.createExistingResult(ctx, idempotencyKey)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", false, err
	}
	return job.ID, true, nil
}

func (s *Postgres) createExistingResult(ctx context.Context, key string) (string, bool, error) {
	var existingID string
	if err := s.pool.QueryRow(ctx, `SELECT job_id FROM idempotency_keys WHERE key = $1`, key).Scan(&existingID); err != nil {
		return "", false, err
	}
	return existingID, false, nil
}

func (s *Postgres) ClaimOutbox(ctx context.Context) (OutboxMessage, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OutboxMessage{}, err
	}
	defer tx.Rollback(ctx)

	var message OutboxMessage
	var payload []byte
	var jobID, jobType string
	var rawPayload json.RawMessage
	err = tx.QueryRow(ctx, `
		SELECT o.id, o.job_id, o.payload, o.attempts
		FROM outbox o
		WHERE (o.status = 'PENDING' AND o.available_at <= NOW())
		   OR (o.status = 'PROCESSING' AND o.locked_at < NOW() - INTERVAL '30 seconds')
		ORDER BY o.id
		LIMIT 1 FOR UPDATE SKIP LOCKED`,
	).Scan(&message.ID, &message.Job.ID, &payload, &message.Attempts)
	if err != nil {
		return OutboxMessage{}, err
	}
	if err := json.Unmarshal(payload, &struct {
		ID      *string          `json:"id"`
		Type    *string          `json:"type"`
		Payload *json.RawMessage `json:"payload"`
	}{ID: &jobID, Type: &jobType, Payload: &rawPayload}); err != nil {
		return OutboxMessage{}, err
	}
	message.Job.ID = jobID
	message.Job.Type = jobType
	message.Job.Payload = rawPayload
	message.Job.Status = domain.StatusPending
	if _, err := tx.Exec(ctx, `UPDATE outbox SET status = 'PROCESSING', attempts = attempts + 1, locked_at = NOW() WHERE id = $1`, message.ID); err != nil {
		return OutboxMessage{}, err
	}
	message.Attempts++
	if err := tx.Commit(ctx); err != nil {
		return OutboxMessage{}, err
	}
	return message, nil
}

func (s *Postgres) MarkOutboxPublished(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE outbox SET status = 'PUBLISHED', published_at = NOW(), locked_at = NULL WHERE id = $1`, id)
	return err
}

func (s *Postgres) MarkOutboxRetry(ctx context.Context, id int64, message string, delay time.Duration) error {
	_, err := s.pool.Exec(ctx, `UPDATE outbox SET status = 'PENDING', error = $2, available_at = NOW() + $3::interval, locked_at = NULL WHERE id = $1`, id, message, delay.String())
	return err
}
