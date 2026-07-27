package middleware

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
)

const idempotencyKeyHeader = "X-Idempotency-Key"

var ErrIdempotencyDatabaseRequired = errors.New("idempotency database is required")

type IdempotencyMiddleware struct {
	db *sql.DB
}

func NewIdempotencyMiddleware(db *sql.DB) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{db: db}
}

func (m *IdempotencyMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.db == nil {
			http.Error(w, "idempotency database is not configured", http.StatusInternalServerError)
			return
		}

		key := r.Header.Get(idempotencyKeyHeader)
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		stored, ok, err := m.loadStoredResponse(r.Context(), key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(stored.StatusCode)
			_, _ = w.Write([]byte(stored.ResponseBody))
			return
		}

		recorder := newResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		if err := m.saveResponse(r.Context(), key, recorder.body.String(), recorder.statusCode); err != nil {
			log.Printf("failed to save idempotency response: %v", err)
		}
	})
}

type storedResponse struct {
	ResponseBody string
	StatusCode   int
}

func (m *IdempotencyMiddleware) loadStoredResponse(ctx context.Context, key string) (storedResponse, bool, error) {
	if m.db == nil {
		return storedResponse{}, false, ErrIdempotencyDatabaseRequired
	}

	var response storedResponse
	err := m.db.QueryRowContext(
		ctx,
		`SELECT response_body, status_code
		 FROM idempotency_keys
		 WHERE key = $1`,
		key,
	).Scan(&response.ResponseBody, &response.StatusCode)
	if errors.Is(err, sql.ErrNoRows) {
		return storedResponse{}, false, nil
	}
	if err != nil {
		return storedResponse{}, false, err
	}

	return response, true, nil
}

func (m *IdempotencyMiddleware) saveResponse(ctx context.Context, key, responseBody string, statusCode int) error {
	if m.db == nil {
		return ErrIdempotencyDatabaseRequired
	}

	_, err := m.db.ExecContext(
		ctx,
		`INSERT INTO idempotency_keys (key, response_body, status_code)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (key) DO NOTHING`,
		key,
		responseBody,
		statusCode,
	)
	return err
}

type responseRecorder struct {
	http.ResponseWriter
	body        bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.statusCode = statusCode
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}
