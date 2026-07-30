package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coolspartak9819-rgb/queuelens/internal/events"
)

type JobEventRecord struct {
	ID             int64           `json:"id"`
	JobID          string          `json:"job_id"`
	EventType      string          `json:"event_type"`
	Attempts       int             `json:"attempts"`
	Error          string          `json:"error,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	OccurredAt     string          `json:"occurred_at"`
	KafkaPartition int             `json:"kafka_partition"`
	KafkaOffset    int64           `json:"kafka_offset"`
}

func (s *Postgres) RecordEvent(ctx context.Context, event events.JobEvent, partition int, offset int64) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO job_events (job_id, event_type, attempts, error, payload, occurred_at, kafka_partition, kafka_offset)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8)
		ON CONFLICT (kafka_partition, kafka_offset) DO NOTHING`, event.JobID, event.Status, event.Attempts, event.Error, payload, event.OccurredAt, partition, offset)
	return err
}

func (s *Postgres) Events(ctx context.Context, jobID string) ([]JobEventRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, job_id, event_type, attempts, COALESCE(error, ''), payload, occurred_at::text, kafka_partition, kafka_offset FROM job_events WHERE job_id = $1 ORDER BY occurred_at, id`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []JobEventRecord
	for rows.Next() {
		var event JobEventRecord
		if err := rows.Scan(&event.ID, &event.JobID, &event.EventType, &event.Attempts, &event.Error, &event.Payload, &event.OccurredAt, &event.KafkaPartition, &event.KafkaOffset); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

// PurgeOld removes jobs and their dependent outbox and audit records in one transaction.
func (s *Postgres) PurgeOld(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays < 1 {
		return 0, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	result, err := tx.Exec(ctx, `DELETE FROM jobs WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
