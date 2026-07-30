package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolspartak9819-rgb/edgegate/internal/config"
)

func TestGatewayProxyAndWAF(t *testing.T) {
	proxy, err := New(config.Config{Routes: []config.Route{{
		Path: "/api", Backends: []string{"http://127.0.0.1:1"}, Plugins: []string{"request-id"}, WAF: &config.WAFConfig{Enabled: true},
	}}})
	if err != nil {
		t.Fatal(err)
	}

	blocked := httptest.NewRequest(http.MethodGet, "http://edgegate.local/api?q=union%20select%20password%20from%20users", nil)
	blockedRecorder := httptest.NewRecorder()
	proxy.ServeHTTP(blockedRecorder, blocked)
	if blockedRecorder.Code != http.StatusForbidden || blockedRecorder.Header().Get("X-Request-ID") == "" {
		t.Fatalf("unexpected WAF response: code=%d headers=%v", blockedRecorder.Code, blockedRecorder.Header())
	}
}
