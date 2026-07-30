package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIKeyProtectsMutationsOnly(t *testing.T) {
	handler := APIKey("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, test := range []struct {
		name, method, path, key string
		status                  int
	}{
		{"public read", http.MethodGet, "/api/jobs", "", http.StatusNoContent},
		{"missing key", http.MethodPost, "/api/jobs", "", http.StatusUnauthorized},
		{"valid key", http.MethodPost, "/api/jobs", "secret", http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("X-API-Key", test.key)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("got %d, want %d", recorder.Code, test.status)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for i, expected := range []int{204, 204, 429} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.RemoteAddr = "192.0.2.1:1234"
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != expected {
			t.Fatalf("request %d: got %d, want %d", i, recorder.Code, expected)
		}
	}
}
