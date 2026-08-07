package observability

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// Snapshot is a bounded, content-free view of process-local runtime metrics.
type Snapshot struct {
	StartedAt string            `json:"started_at"`
	UptimeSec int64             `json:"uptime_seconds"`
	Requests  RequestSnapshot   `json:"requests"`
	Assistant AssistantSnapshot `json:"assistant"`
	Access    AccessSnapshot    `json:"access"`
}

// RequestSnapshot contains aggregate HTTP request counters.
type RequestSnapshot struct {
	Total           uint64  `json:"total"`
	InFlight        int64   `json:"in_flight"`
	Responses2XX    uint64  `json:"responses_2xx"`
	Responses3XX    uint64  `json:"responses_3xx"`
	Responses4XX    uint64  `json:"responses_4xx"`
	Responses5XX    uint64  `json:"responses_5xx"`
	DurationTotalMS float64 `json:"duration_total_ms"`
}

// AssistantSnapshot contains aggregate model workflow counters without content.
type AssistantSnapshot struct {
	Runs            uint64  `json:"runs"`
	Failures        uint64  `json:"failures"`
	ToolCalls       uint64  `json:"tool_calls"`
	InputTokens     uint64  `json:"input_tokens"`
	OutputTokens    uint64  `json:"output_tokens"`
	TotalTokens     uint64  `json:"total_tokens"`
	DurationTotalMS float64 `json:"duration_total_ms"`
}

// AccessSnapshot contains aggregate hosted-demo rejection counters.
type AccessSnapshot struct {
	Unauthorized           uint64 `json:"unauthorized"`
	RateLimited            uint64 `json:"rate_limited"`
	AssistantQuotaRejected uint64 `json:"assistant_quota_rejected"`
}

// AssistantCompletion is the content-free completion metadata recorded by Metrics.
type AssistantCompletion struct {
	ElapsedMS    int64
	ToolCalls    int
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// Metrics stores bounded process-local counters using atomics.
type Metrics struct {
	startedAt time.Time
	now       func() time.Time

	requestsTotal        atomic.Uint64
	inFlight             atomic.Int64
	responses2XX         atomic.Uint64
	responses3XX         atomic.Uint64
	responses4XX         atomic.Uint64
	responses5XX         atomic.Uint64
	requestDurationNanos atomic.Uint64

	assistantRuns         atomic.Uint64
	assistantFailures     atomic.Uint64
	assistantToolCalls    atomic.Uint64
	assistantInputTokens  atomic.Uint64
	assistantOutputTokens atomic.Uint64
	assistantTotalTokens  atomic.Uint64
	assistantElapsedMS    atomic.Uint64

	accessUnauthorized           atomic.Uint64
	accessRateLimited            atomic.Uint64
	accessAssistantQuotaRejected atomic.Uint64
}

// NewMetrics creates an empty process-local metrics registry.
func NewMetrics() *Metrics {
	return newMetrics(time.Now)
}

func newMetrics(now func() time.Time) *Metrics {
	return &Metrics{startedAt: now().UTC(), now: now}
}

// Snapshot returns an atomic aggregate view. It never contains request content.
func (m *Metrics) Snapshot() Snapshot {
	if m == nil {
		return Snapshot{}
	}
	uptime := m.now().UTC().Sub(m.startedAt)
	if uptime < 0 {
		uptime = 0
	}
	return Snapshot{
		StartedAt: m.startedAt.Format(time.RFC3339Nano),
		UptimeSec: int64(uptime / time.Second),
		Requests: RequestSnapshot{
			Total:           m.requestsTotal.Load(),
			InFlight:        m.inFlight.Load(),
			Responses2XX:    m.responses2XX.Load(),
			Responses3XX:    m.responses3XX.Load(),
			Responses4XX:    m.responses4XX.Load(),
			Responses5XX:    m.responses5XX.Load(),
			DurationTotalMS: float64(m.requestDurationNanos.Load()) / float64(time.Millisecond),
		},
		Assistant: AssistantSnapshot{
			Runs:            m.assistantRuns.Load(),
			Failures:        m.assistantFailures.Load(),
			ToolCalls:       m.assistantToolCalls.Load(),
			InputTokens:     m.assistantInputTokens.Load(),
			OutputTokens:    m.assistantOutputTokens.Load(),
			TotalTokens:     m.assistantTotalTokens.Load(),
			DurationTotalMS: float64(m.assistantElapsedMS.Load()),
		},
		Access: AccessSnapshot{
			Unauthorized:           m.accessUnauthorized.Load(),
			RateLimited:            m.accessRateLimited.Load(),
			AssistantQuotaRejected: m.accessAssistantQuotaRejected.Load(),
		},
	}
}

// AssistantStarted records one model workflow that passed request validation.
func (m *Metrics) AssistantStarted() {
	if m != nil {
		m.assistantRuns.Add(1)
	}
}

// AssistantFailed records one model workflow failure.
func (m *Metrics) AssistantFailed() {
	if m != nil {
		m.assistantFailures.Add(1)
	}
}

// AccessUnauthorized records one rejected hosted-demo credential check.
func (m *Metrics) AccessUnauthorized() {
	if m != nil {
		m.accessUnauthorized.Add(1)
	}
}

// AccessRateLimited records one process-local API limit rejection.
func (m *Metrics) AccessRateLimited() {
	if m != nil {
		m.accessRateLimited.Add(1)
	}
}

// AccessAssistantQuotaRejected records one assistant budget rejection.
func (m *Metrics) AccessAssistantQuotaRejected() {
	if m != nil {
		m.accessAssistantQuotaRejected.Add(1)
	}
}

// ObserveAssistantCompletion records trusted completion metadata only.
func (m *Metrics) ObserveAssistantCompletion(completion AssistantCompletion) {
	if m == nil {
		return
	}
	m.assistantElapsedMS.Add(nonNegative(completion.ElapsedMS))
	m.assistantToolCalls.Add(nonNegative(int64(completion.ToolCalls)))
	m.assistantInputTokens.Add(nonNegative(int64(completion.InputTokens)))
	m.assistantOutputTokens.Add(nonNegative(int64(completion.OutputTokens)))
	m.assistantTotalTokens.Add(nonNegative(int64(completion.TotalTokens)))
}

// HTTPMiddleware correlates, counts, and safely logs completed HTTP requests.
func (m *Metrics) HTTPMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	if m == nil {
		m = NewMetrics()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := m.now()
		requestID := newRequestID()
		w.Header().Set("X-Request-ID", requestID)
		m.inFlight.Add(1)

		recorder := &statusRecorder{ResponseWriter: w}
		var responseWriter http.ResponseWriter = recorder
		if flusher, ok := w.(http.Flusher); ok {
			responseWriter = &flushRecorder{statusRecorder: recorder, flusher: flusher}
		}
		next.ServeHTTP(responseWriter, r)

		m.inFlight.Add(-1)
		m.requestsTotal.Add(1)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		switch {
		case status >= 500:
			m.responses5XX.Add(1)
		case status >= 400:
			m.responses4XX.Add(1)
		case status >= 300:
			m.responses3XX.Add(1)
		default:
			m.responses2XX.Add(1)
		}
		duration := m.now().Sub(started)
		if duration < 0 {
			duration = 0
		}
		m.requestDurationNanos.Add(uint64(duration))

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}
		logger.Log(r.Context(), level, "http_request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", float64(duration)/float64(time.Millisecond),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type flushRecorder struct {
	*statusRecorder
	flusher http.Flusher
}

func (w *flushRecorder) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	w.flusher.Flush()
}

func nonNegative(value int64) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value)
}

func newRequestID() string {
	var value [12]byte
	if _, err := rand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}
	return fmt.Sprintf("fallback-%d", time.Now().UTC().UnixNano())
}
