package webapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"steam-profit-calculator/internal/observability"
)

func TestAccessOptionsFromEnvironment(t *testing.T) {
	for _, name := range []string{
		"STEAM_PROFIT_DEMO_USERNAME",
		"STEAM_PROFIT_DEMO_PASSWORD",
		"STEAM_PROFIT_RATE_LIMIT_PER_MINUTE",
		"STEAM_PROFIT_AI_RUNS_PER_HOUR",
		"STEAM_PROFIT_TRUST_PROXY_HEADERS",
	} {
		t.Setenv(name, "")
	}

	options, err := AccessOptionsFromEnvironment()
	if err != nil {
		t.Fatalf("AccessOptionsFromEnvironment() error = %v", err)
	}
	if options != (AccessOptions{}) {
		t.Fatalf("options = %#v, want zero values", options)
	}

	t.Setenv("STEAM_PROFIT_DEMO_USERNAME", "reviewer")
	if _, err := AccessOptionsFromEnvironment(); err == nil {
		t.Fatal("partial credentials error = nil")
	}
	t.Setenv("STEAM_PROFIT_DEMO_PASSWORD", "short")
	if _, err := AccessOptionsFromEnvironment(); err == nil {
		t.Fatal("short password error = nil")
	}
	t.Setenv("STEAM_PROFIT_DEMO_PASSWORD", "long-demo-password")
	t.Setenv("STEAM_PROFIT_RATE_LIMIT_PER_MINUTE", "60")
	t.Setenv("STEAM_PROFIT_AI_RUNS_PER_HOUR", "12")
	t.Setenv("STEAM_PROFIT_TRUST_PROXY_HEADERS", "true")
	options, err = AccessOptionsFromEnvironment()
	if err != nil {
		t.Fatalf("valid AccessOptionsFromEnvironment() error = %v", err)
	}
	if options.Username != "reviewer" || options.RequestsPerMinute != 60 || options.AssistantRunsPerHour != 12 || !options.TrustProxyClientHeader {
		t.Fatalf("valid options = %#v", options)
	}

	t.Setenv("STEAM_PROFIT_RATE_LIMIT_PER_MINUTE", "-1")
	if _, err := AccessOptionsFromEnvironment(); err == nil {
		t.Fatal("negative rate error = nil")
	}
	t.Setenv("STEAM_PROFIT_RATE_LIMIT_PER_MINUTE", "60")
	t.Setenv("STEAM_PROFIT_TRUST_PROXY_HEADERS", "sometimes")
	if _, err := AccessOptionsFromEnvironment(); err == nil {
		t.Fatal("invalid proxy boolean error = nil")
	}

	t.Setenv("STEAM_PROFIT_TRUST_PROXY_HEADERS", "false")
	t.Setenv("STEAM_PROFIT_DEMO_USERNAME", "reviewer:admin")
	if _, err := AccessOptionsFromEnvironment(); err == nil {
		t.Fatal("username containing colon error = nil")
	}
	t.Setenv("STEAM_PROFIT_DEMO_USERNAME", "reviewer")
	t.Setenv("STEAM_PROFIT_DEMO_PASSWORD", "long-demo-password\nsecond-line")
	if _, err := AccessOptionsFromEnvironment(); err == nil {
		t.Fatal("password containing line break error = nil")
	}
	t.Setenv("STEAM_PROFIT_DEMO_PASSWORD", "  spaced-demo-password  ")
	options, err = AccessOptionsFromEnvironment()
	if err != nil {
		t.Fatalf("password containing spaces error = %v", err)
	}
	if options.Password != "  spaced-demo-password  " {
		t.Fatalf("password = %q, want surrounding spaces preserved", options.Password)
	}
}

func TestAccessMiddlewareProtectsDemoAndKeepsHealthPublic(t *testing.T) {
	metrics := observability.NewMetrics()
	controller, err := newAccessController(AccessOptions{Username: "reviewer", Password: "long-demo-password"}, metrics)
	if err != nil {
		t.Fatalf("newAccessController() error = %v", err)
	}
	handler := controller.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusNoContent {
		t.Fatalf("public health status = %d", health.Code)
	}

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/", nil))
	if missing.Code != http.StatusUnauthorized || missing.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("missing credential response = %d / %#v", missing.Code, missing.Header())
	}

	wrongRequest := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	wrongRequest.SetBasicAuth("reviewer", "wrong-password")
	wrong := httptest.NewRecorder()
	handler.ServeHTTP(wrong, wrongRequest)
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong credential status = %d", wrong.Code)
	}

	validRequest := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	validRequest.SetBasicAuth("reviewer", "long-demo-password")
	valid := httptest.NewRecorder()
	handler.ServeHTTP(valid, validRequest)
	if valid.Code != http.StatusNoContent {
		t.Fatalf("valid credential status = %d", valid.Code)
	}
	if got := metrics.Snapshot().Access.Unauthorized; got != 2 {
		t.Fatalf("unauthorized metric = %d, want 2", got)
	}
}

func TestAccessMiddlewareLimitsAPIByClientAndResetsWindow(t *testing.T) {
	now := time.Date(2026, time.August, 7, 4, 0, 0, 0, time.UTC)
	metrics := observability.NewMetrics()
	controller, err := newAccessControllerWithClock(AccessOptions{RequestsPerMinute: 1}, metrics, func() time.Time { return now }, 8)
	if err != nil {
		t.Fatalf("newAccessControllerWithClock() error = %v", err)
	}
	handler := controller.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := func(remote, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.RemoteAddr = remote
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}

	if got := request("192.0.2.1:1000", "/api/v1/records").Code; got != http.StatusNoContent {
		t.Fatalf("first client status = %d", got)
	}
	limited := request("192.0.2.1:2000", "/api/v1/records")
	if limited.Code != http.StatusTooManyRequests || limited.Header().Get("Retry-After") != "60" {
		t.Fatalf("limited response = %d / Retry-After %q", limited.Code, limited.Header().Get("Retry-After"))
	}
	if got := request("192.0.2.2:1000", "/api/v1/records").Code; got != http.StatusNoContent {
		t.Fatalf("independent client status = %d", got)
	}
	if got := request("192.0.2.1:3000", "/styles.css").Code; got != http.StatusNoContent {
		t.Fatalf("static request status = %d", got)
	}
	now = now.Add(time.Minute)
	if got := request("192.0.2.1:4000", "/api/v1/records").Code; got != http.StatusNoContent {
		t.Fatalf("reset client status = %d", got)
	}
	if got := metrics.Snapshot().Access.RateLimited; got != 1 {
		t.Fatalf("rate-limited metric = %d, want 1", got)
	}
}

func TestAccessControllerTrustsForwardedClientOnlyWhenConfigured(t *testing.T) {
	now := time.Date(2026, time.August, 7, 4, 0, 0, 0, time.UTC)
	controller, err := newAccessControllerWithClock(AccessOptions{RequestsPerMinute: 1, TrustProxyClientHeader: true}, nil, func() time.Time { return now }, 8)
	if err != nil {
		t.Fatalf("newAccessControllerWithClock() error = %v", err)
	}
	handler := controller.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	call := func(realIP, forwarded string) int {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
		r.RemoteAddr = "192.0.2.10:1000"
		r.Header.Set("X-Real-IP", realIP)
		r.Header.Set("X-Forwarded-For", forwarded)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}
	if got := call("198.51.100.1", "203.0.113.9"); got != http.StatusNoContent {
		t.Fatalf("first forwarded client status = %d", got)
	}
	if got := call("198.51.100.2", "203.0.113.9"); got != http.StatusNoContent {
		t.Fatalf("second forwarded client status = %d", got)
	}
	if got := call("198.51.100.2", "203.0.113.10"); got != http.StatusTooManyRequests {
		t.Fatalf("repeated forwarded client status = %d", got)
	}

	untrusted, err := newAccessControllerWithClock(AccessOptions{RequestsPerMinute: 1}, nil, func() time.Time { return now }, 8)
	if err != nil {
		t.Fatalf("new untrusted controller error = %v", err)
	}
	first := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	first.RemoteAddr = "192.0.2.10:1000"
	first.Header.Set("X-Real-IP", "198.51.100.1")
	first.Header.Set("X-Forwarded-For", "198.51.100.1")
	second := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	second.RemoteAddr = "192.0.2.10:2000"
	second.Header.Set("X-Real-IP", "198.51.100.2")
	second.Header.Set("X-Forwarded-For", "198.51.100.2")
	if allowed, _ := untrusted.allowAPIRequest(untrusted.clientKey(first)); !allowed {
		t.Fatal("first untrusted-proxy request was rejected")
	}
	if allowed, _ := untrusted.allowAPIRequest(untrusted.clientKey(second)); allowed {
		t.Fatal("untrusted forwarded header created an independent budget")
	}
}

func TestAccessControllerBoundsClientsAndAssistantQuota(t *testing.T) {
	now := time.Date(2026, time.August, 7, 4, 0, 0, 0, time.UTC)
	controller, err := newAccessControllerWithClock(AccessOptions{RequestsPerMinute: 1, AssistantRunsPerHour: 1}, nil, func() time.Time { return now }, 2)
	if err != nil {
		t.Fatalf("newAccessControllerWithClock() error = %v", err)
	}
	controller.allowAPIRequest("client-a")
	now = now.Add(time.Second)
	controller.allowAPIRequest("client-b")
	now = now.Add(time.Second)
	controller.allowAPIRequest("client-c")
	if len(controller.clients) != 2 {
		t.Fatalf("tracked clients = %d, want 2", len(controller.clients))
	}
	if _, exists := controller.clients["client-a"]; exists {
		t.Fatal("oldest client was not evicted")
	}
	if allowed, _ := controller.allowAssistantRun(); !allowed {
		t.Fatal("first assistant run was rejected")
	}
	if allowed, retryAfter := controller.allowAssistantRun(); allowed || retryAfter <= 59*time.Minute {
		t.Fatalf("second assistant run = %v / %v", allowed, retryAfter)
	}
	now = now.Add(time.Hour)
	if allowed, _ := controller.allowAssistantRun(); !allowed {
		t.Fatal("assistant quota did not reset")
	}
}
