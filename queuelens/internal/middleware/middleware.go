package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

func APIKey(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			protected := r.Method == http.MethodPost && (r.URL.Path == "/api/jobs" || strings.HasSuffix(r.URL.Path, "/retry"))
			if required != "" && protected && r.Header.Get("X-API-Key") != required {
				writeError(w, http.StatusUnauthorized, "valid X-API-Key is required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type visitor struct {
	started time.Time
	count   int
}
type RateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	visitors map[string]visitor
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, visitors: make(map[string]visitor)}
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)
		now := time.Now()
		l.mu.Lock()
		state := l.visitors[key]
		if state.started.IsZero() || now.Sub(state.started) >= l.window {
			state = visitor{started: now}
		}
		state.count++
		l.visitors[key] = state
		allowed := state.count <= l.limit
		l.mu.Unlock()
		if !allowed {
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
