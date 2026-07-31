package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/coolspartak9819-rgb/queuelens/internal/middleware"
	"github.com/coolspartak9819-rgb/queuelens/internal/queue"
	"github.com/coolspartak9819-rgb/queuelens/internal/store"
)

type api struct {
	store *store.Postgres
	queue *queue.Redis
}
type createRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
type response struct {
	JobID string `json:"job_id"`
}

var requestsTotal atomic.Uint64
var jobsCreated atomic.Uint64
var requestLatencyCount atomic.Uint64
var requestLatencySumMicros atomic.Uint64
var requestLatencyBuckets [6]atomic.Uint64

var latencyBucketLimits = [...]time.Duration{
	5 * time.Millisecond,
	25 * time.Millisecond,
	100 * time.Millisecond,
	250 * time.Millisecond,
	500 * time.Millisecond,
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	jobStore, err := store.New(ctx, env("DATABASE_URL", "postgres://queuelens:queuelens@localhost:5432/queuelens"))
	if err != nil {
		log.Fatal(err)
	}
	defer jobStore.Close()
	jobQueue := queue.New(env("REDIS_ADDR", "localhost:6379"), env("QUEUE_STREAM", "jobs"), env("WORKER_GROUP", "queuelens-workers"), "api")
	defer jobQueue.Close()
	if err := jobQueue.EnsureGroup(ctx); err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	service := &api{store: jobStore, queue: jobQueue}
	mux.HandleFunc("GET /health", service.health)
	mux.HandleFunc("GET /ready", service.ready)
	mux.HandleFunc("GET /metrics", service.metrics)
	mux.HandleFunc("GET /api/jobs", service.list)
	mux.HandleFunc("POST /api/jobs", service.create)
	mux.HandleFunc("GET /api/jobs/{id}", service.get)
	mux.HandleFunc("GET /api/jobs/{id}/events", service.events)
	mux.HandleFunc("POST /api/jobs/{id}/retry", service.retry)
	mux.HandleFunc("GET /api/stats", service.stats)
	mux.HandleFunc("GET /api/queue", service.queueStats)
	mux.Handle("/", http.FileServer(http.Dir("web")))
	server := &http.Server{Addr: ":8080", Handler: logging(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	server.Handler = logging(middleware.NewRateLimiter(60, time.Minute).Middleware(middleware.APIKey(env("API_KEY", ""))(mux)))
	pprofServer := startPprofServer(env("PPROF_ADDR", ""))
	log.Printf("QueueLens API listening on %s", server.Addr)
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-stop:
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("API shutdown: %v", err)
		}
		if pprofServer != nil {
			if err := pprofServer.Shutdown(shutdownCtx); err != nil {
				log.Printf("pprof shutdown: %v", err)
			}
		}
	}
}

func startPprofServer(addr string) *http.Server {
	if strings.TrimSpace(addr) == "" {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	mux.Handle("GET /debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("GET /debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("GET /debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("GET /debug/pprof/block", pprof.Handler("block"))
	mux.Handle("GET /debug/pprof/mutex", pprof.Handler("mutex"))

	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("QueueLens pprof listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("pprof server: %v", err)
		}
	}()
	return server
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (a *api) ready(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Ping(r.Context()); err != nil {
		jsonError(w, err, http.StatusServiceUnavailable)
		return
	}
	if err := a.queue.Ping(r.Context()); err != nil {
		jsonError(w, err, http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
func (a *api) create(w http.ResponseWriter, r *http.Request) {
	var input createRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Type) == "" {
		jsonError(w, "type and payload are required", 400)
		return
	}
	id := newID()
	job := store.NewJob(id, input.Type, input.Payload)
	jobID, created, err := a.store.CreateWithOutbox(r.Context(), job, r.Header.Get("Idempotency-Key"))
	if err != nil {
		jsonError(w, err, 500)
		return
	}
	if !created {
		writeJSON(w, http.StatusOK, response{JobID: jobID})
		return
	}
	jobsCreated.Add(1)
	writeJSON(w, 202, response{JobID: id})
}
func (a *api) get(w http.ResponseWriter, r *http.Request) {
	job, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		jsonError(w, err, 404)
		return
	}
	writeJSON(w, 200, job)
}
func (a *api) events(w http.ResponseWriter, r *http.Request) {
	events, err := a.store.Events(r.Context(), r.PathValue("id"))
	if err != nil {
		jsonError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}
func (a *api) list(w http.ResponseWriter, r *http.Request) {
	jobs, err := a.store.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		jsonError(w, err, 500)
		return
	}
	writeJSON(w, 200, map[string]any{"jobs": jobs})
}
func (a *api) stats(w http.ResponseWriter, r *http.Request) {
	stats, err := a.store.Stats(r.Context())
	if err != nil {
		jsonError(w, err, 500)
		return
	}
	writeJSON(w, 200, stats)
}
func (a *api) queueStats(w http.ResponseWriter, r *http.Request) {
	stats, err := a.queue.Stats(r.Context())
	if err != nil {
		jsonError(w, err, http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
func (a *api) retry(w http.ResponseWriter, r *http.Request) {
	job, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		jsonError(w, err, 404)
		return
	}
	if err := a.store.Retry(r.Context(), job.ID, "manual retry"); err != nil {
		jsonError(w, err, 500)
		return
	}
	if err := a.queue.Enqueue(r.Context(), job); err != nil {
		jsonError(w, err, 503)
		return
	}
	writeJSON(w, 202, response{JobID: job.ID})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func jsonError(w http.ResponseWriter, err any, status int) {
	writeJSON(w, status, map[string]any{"error": fmt.Sprint(err)})
}
func (a *api) metrics(w http.ResponseWriter, r *http.Request) {
	queueStats, err := a.queue.Stats(r.Context())
	if err != nil {
		jsonError(w, err, http.StatusServiceUnavailable)
		return
	}
	jobStats, err := a.store.Stats(r.Context())
	if err != nil {
		jsonError(w, err, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "queuelens_http_requests_total %d\nqueuelens_jobs_created_total %d\nqueuelens_queue_stream_length %d\nqueuelens_queue_pending %d\n", requestsTotal.Load(), jobsCreated.Load(), queueStats.StreamLength, queueStats.Pending)
	fmt.Fprintf(w, "queuelens_http_request_duration_seconds_count %d\nqueuelens_http_request_duration_seconds_sum %.6f\n", requestLatencyCount.Load(), float64(requestLatencySumMicros.Load())/1_000_000)
	for index, limit := range latencyBucketLimits {
		fmt.Fprintf(w, "queuelens_http_request_duration_seconds_bucket{le=\"%.3f\"} %d\n", limit.Seconds(), cumulativeLatencyBucket(index))
	}
	fmt.Fprintf(w, "queuelens_http_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", requestLatencyCount.Load())
	for _, status := range []string{"PENDING", "RUNNING", "RETRYING", "COMPLETED", "FAILED"} {
		fmt.Fprintf(w, "queuelens_jobs{status=%q} %d\n", status, jobStats[status])
	}
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestsTotal.Add(1)
		started := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = newID()
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
		duration := time.Since(started)
		observeRequestDuration(duration)
		log.Printf("request_id=%s method=%s path=%s duration=%s", requestID, r.Method, r.URL.Path, duration)
	})
}

func observeRequestDuration(duration time.Duration) {
	requestLatencyCount.Add(1)
	requestLatencySumMicros.Add(uint64(duration / time.Microsecond))
	for index, limit := range latencyBucketLimits {
		if duration <= limit {
			requestLatencyBuckets[index].Add(1)
			return
		}
	}
}

func cumulativeLatencyBucket(index int) uint64 {
	var total uint64
	for current := 0; current <= index; current++ {
		total += requestLatencyBuckets[current].Load()
	}
	return total
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func newID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}
