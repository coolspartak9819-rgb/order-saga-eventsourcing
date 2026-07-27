package observability

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

type Metrics struct {
	requests      atomic.Uint64
	clientErrors  atomic.Uint64
	serverErrors  atomic.Uint64
	totalDuration atomic.Uint64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)

		m.requests.Add(1)
		m.totalDuration.Add(uint64(time.Since(started).Microseconds()))
		switch {
		case writer.statusCode >= http.StatusInternalServerError:
			m.serverErrors.Add(1)
		case writer.statusCode >= http.StatusBadRequest:
			m.clientErrors.Add(1)
		}
	})
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		requests := m.requests.Load()
		duration := m.totalDuration.Load()
		averageDuration := uint64(0)
		if requests > 0 {
			averageDuration = duration / requests
		}

		_, _ = fmt.Fprintf(w,
			"# HELP order_service_http_requests_total Total HTTP requests.\n"+
				"# TYPE order_service_http_requests_total counter\n"+
				"order_service_http_requests_total %d\n"+
				"# HELP order_service_http_client_errors_total HTTP responses with 4xx status.\n"+
				"# TYPE order_service_http_client_errors_total counter\n"+
				"order_service_http_client_errors_total %d\n"+
				"# HELP order_service_http_server_errors_total HTTP responses with 5xx status.\n"+
				"# TYPE order_service_http_server_errors_total counter\n"+
				"order_service_http_server_errors_total %d\n"+
				"# HELP order_service_http_average_duration_microseconds Average request duration.\n"+
				"# TYPE order_service_http_average_duration_microseconds gauge\n"+
				"order_service_http_average_duration_microseconds %d\n",
			requests,
			m.clientErrors.Load(),
			m.serverErrors.Load(),
			averageDuration,
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.statusCode = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}
