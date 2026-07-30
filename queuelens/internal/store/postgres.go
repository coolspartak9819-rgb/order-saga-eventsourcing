package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/coolspartak9819-rgb/queuelens/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrJobNotFound = errors.New("job not found")

type Postgres struct{ pool *pgxpool.Pool }

func New(ctx context.Context, url string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

func (s *Postgres) Close() { s.pool.Close() }

func (s *Postgres) Create(ctx context.Context, job domain.Job) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO jobs (id, job_type, payload, status) VALUES ($1, $2, $3, $4)`, job.ID, job.Type, job.Payload, job.Status)
	return err
}

func (s *Postgres) Get(ctx context.Context, id string) (domain.Job, error) {
	var job domain.Job
	err := s.pool.QueryRow(ctx, `SELECT id, job_type, payload, status, attempts, COALESCE(error, ''), created_at, updated_at, completed_at FROM jobs WHERE id = $1`, id).Scan(&job.ID, &job.Type, &job.Payload, &job.Status, &job.Attempts, &job.Error, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return job, ErrJobNotFound
	}
	return job, err
}

func (s *Postgres) List(ctx context.Context, status string) ([]domain.Job, error) {
	query, args := `SELECT id, job_type, payload, status, attempts, COALESCE(error, ''), created_at, updated_at, completed_at FROM jobs ORDER BY created_at DESC LIMIT 100`, []any{}
	if status != "" {
		query = `SELECT id, job_type, payload, status, attempts, COALESCE(error, ''), created_at, updated_at, completed_at FROM jobs WHERE status = $1 ORDER BY created_at DESC LIMIT 100`
		args = []any{status}
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []domain.Job
	for rows.Next() {
		var job domain.Job
		if err := rows.Scan(&job.ID, &job.Type, &job.Payload, &job.Status, &job.Attempts, &job.Error, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Postgres) SetRunning(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status = $1, attempts = attempts + 1, updated_at = NOW() WHERE id = $2`, domain.StatusRunning, id)
	return err
}
func (s *Postgres) Complete(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status = $1, updated_at = NOW(), completed_at = NOW() WHERE id = $2`, domain.StatusCompleted, id)
	return err
}
func (s *Postgres) Retry(ctx context.Context, id, message string) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status = $1, error = $2, updated_at = NOW() WHERE id = $3`, domain.StatusRetrying, message, id)
	return err
}
func (s *Postgres) Fail(ctx context.Context, id, message string) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status = $1, error = $2, updated_at = NOW() WHERE id = $3`, domain.StatusFailed, message, id)
	return err
}

func (s *Postgres) Stats(ctx context.Context) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `SELECT status, COUNT(*) FROM jobs GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}
	return stats, rows.Err()
}

func EncodePayload(value any) (json.RawMessage, error) { return json.Marshal(value) }
func NewJob(id, jobType string, payload json.RawMessage) domain.Job {
	now := time.Now().UTC()
	return domain.Job{ID: id, Type: jobType, Payload: payload, Status: domain.StatusPending, CreatedAt: now, UpdatedAt: now}
}
