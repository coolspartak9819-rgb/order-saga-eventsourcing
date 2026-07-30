package gateway

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Plugin func(http.Handler) http.Handler

type pluginRegistry struct{ plugins map[string]Plugin }

func newPluginRegistry() *pluginRegistry {
	return &pluginRegistry{plugins: map[string]Plugin{
		"request-id": requestIDPlugin,
		"access-log": accessLogPlugin,
	}}
}

func (r *pluginRegistry) chain(names []string, next http.Handler) (http.Handler, error) {
	for i := len(names) - 1; i >= 0; i-- {
		plugin, ok := r.plugins[names[i]]
		if !ok {
			return nil, fmt.Errorf("unknown plugin %q", names[i])
		}
		next = plugin(next)
	}
	return next, nil
}

func requestIDPlugin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", requestCounter.Add(1))
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func accessLogPlugin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, timeSince(started))
	})
}

var requestCounter atomic.Uint64

var now = time.Now
var timeSince = time.Since

func clientKey(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-User-ID")); value != "" {
		return value
	}
	return r.RemoteAddr
}
