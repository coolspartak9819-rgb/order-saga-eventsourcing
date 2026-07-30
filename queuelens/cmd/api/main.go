package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

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
	mux := http.NewServeMux()
	service := &api{store: jobStore, queue: jobQueue}
	mux.HandleFunc("GET /health", service.health)
	mux.HandleFunc("GET /metrics", metrics)
	mux.HandleFunc("GET /api/jobs", service.list)
	mux.HandleFunc("POST /api/jobs", service.create)
	mux.HandleFunc("GET /api/jobs/{id}", service.get)
	mux.HandleFunc("POST /api/jobs/{id}/retry", service.retry)
	mux.HandleFunc("GET /api/stats", service.stats)
	mux.Handle("/", http.FileServer(http.Dir("web")))
	server := &http.Server{Addr: ":8080", Handler: logging(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("QueueLens API listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	if err := a.queue.Ping(r.Context()); err != nil {
		jsonError(w, err, 503)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (a *api) create(w http.ResponseWriter, r *http.Request) {
	var input createRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Type) == "" {
		jsonError(w, "type and payload are required", 400)
		return
	}
	id := newID()
	job := store.NewJob(id, input.Type, input.Payload)
	if err := a.store.CreateWithOutbox(r.Context(), job); err != nil {
		jsonError(w, err, 500)
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
func metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "queuelens_http_requests_total %d\nqueuelens_jobs_created_total %d\n", requestsTotal.Load(), jobsCreated.Load())
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestsTotal.Add(1)
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, time.Since(started))
	})
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
