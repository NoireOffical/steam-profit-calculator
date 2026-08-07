package observability

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHTTPMiddlewareRecordsAndSafelyLogsRequests(t *testing.T) {
	clock := &testClock{now: time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)}
	metrics := newMetrics(clock.Now)
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := metrics.HTTPMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clock.Advance(25 * time.Millisecond)
		http.Error(w, "safe public error", http.StatusUnprocessableEntity)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/calculations?secret=query-value", strings.NewReader(`{"prompt":"body-secret"}`))
	request.Header.Set("Authorization", "Bearer header-secret")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
	if requestID := recorder.Header().Get("X-Request-ID"); len(requestID) != 24 {
		t.Fatalf("X-Request-ID = %q, want 24 hex characters", requestID)
	}
	snapshot := metrics.Snapshot()
	if snapshot.Requests.Total != 1 || snapshot.Requests.Responses4XX != 1 || snapshot.Requests.InFlight != 0 {
		t.Fatalf("request snapshot = %#v", snapshot.Requests)
	}
	if snapshot.Requests.DurationTotalMS != 25 {
		t.Fatalf("duration = %v, want 25", snapshot.Requests.DurationTotalMS)
	}
	logged := logs.String()
	for _, forbidden := range []string{"query-value", "body-secret", "header-secret", "safe public error"} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("log contains forbidden content %q: %s", forbidden, logged)
		}
	}
	for _, required := range []string{`"level":"WARN"`, `"method":"POST"`, `"path":"/api/v1/calculations"`, `"status":422`} {
		if !strings.Contains(logged, required) {
			t.Fatalf("log missing %q: %s", required, logged)
		}
	}
}

func TestHTTPMiddlewarePreservesFlushing(t *testing.T) {
	metrics := NewMetrics()
	handler := metrics.HTTPMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("wrapped response writer does not implement http.Flusher")
		}
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
	}))
	recorder := newFlushingRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if !recorder.flushed {
		t.Fatal("underlying writer was not flushed")
	}
}

func TestAssistantMetricsAndConcurrentRequests(t *testing.T) {
	metrics := NewMetrics()
	metrics.AssistantStarted()
	metrics.ObserveAssistantCompletion(AssistantCompletion{
		ElapsedMS: 125, ToolCalls: 2, InputTokens: 40, OutputTokens: 10, TotalTokens: 50,
	})
	metrics.AssistantFailed()
	metrics.AccessUnauthorized()
	metrics.AccessRateLimited()
	metrics.AccessAssistantQuotaRejected()

	handler := metrics.HTTPMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/health", nil))
		}()
	}
	group.Wait()

	snapshot := metrics.Snapshot()
	if snapshot.Requests.Total != 32 || snapshot.Requests.Responses2XX != 32 || snapshot.Requests.InFlight != 0 {
		t.Fatalf("request snapshot = %#v", snapshot.Requests)
	}
	if snapshot.Assistant.Runs != 1 || snapshot.Assistant.Failures != 1 || snapshot.Assistant.ToolCalls != 2 || snapshot.Assistant.TotalTokens != 50 || snapshot.Assistant.DurationTotalMS != 125 {
		t.Fatalf("assistant snapshot = %#v", snapshot.Assistant)
	}
	if snapshot.Access.Unauthorized != 1 || snapshot.Access.RateLimited != 1 || snapshot.Access.AssistantQuotaRejected != 1 {
		t.Fatalf("access snapshot = %#v", snapshot.Access)
	}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}

type flushingRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func newFlushingRecorder() *flushingRecorder {
	return &flushingRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (r *flushingRecorder) Flush() {
	r.flushed = true
	r.ResponseRecorder.Flush()
}
