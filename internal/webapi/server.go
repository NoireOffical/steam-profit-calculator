package webapi

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"steam-profit-calculator/internal/assistant"
	"steam-profit-calculator/internal/calculation"
	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/lookup"
	"steam-profit-calculator/internal/observability"
	"steam-profit-calculator/internal/storage"
)

const (
	maxJSONBodyBytes = 1 << 20
	maxLookupLimit   = 50
	maxItemPrefix    = 100
)

//go:embed web/*
var embeddedWeb embed.FS

// RecordStore is the persistence boundary required by the web application.
type RecordStore = storage.RecordRepository

type server struct {
	lookupService *lookup.Service
	recordStore   RecordStore
	assistant     *assistant.Service
	metrics       *observability.Metrics
	access        *accessController
	static        http.Handler
}

// HandlerOptions configures process-local observability without changing API behavior.
type HandlerOptions struct {
	Logger  *slog.Logger
	Metrics *observability.Metrics
	Access  AccessOptions
}

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type dataEnvelope struct {
	Data any `json:"data"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type assistantRequest struct {
	Message string                          `json:"message"`
	History []assistant.ConversationMessage `json:"history,omitempty"`
}

type calculationRequest struct {
	ItemName   string    `json:"item_name"`
	BuyPrice   float64   `json:"buy_price"`
	Quantity   *int      `json:"quantity,omitempty"`
	Multiplier *float64  `json:"multiplier,omitempty"`
	SellPrices []float64 `json:"sell_prices"`
}

type reverseCalculationRequest struct {
	ItemName          string                `json:"item_name"`
	SolveDirection    domain.SolveDirection `json:"solve_direction"`
	BuyPrice          *float64              `json:"buy_price,omitempty"`
	SellPrice         *float64              `json:"sell_price,omitempty"`
	Quantity          *int                  `json:"quantity,omitempty"`
	Multiplier        *float64              `json:"multiplier,omitempty"`
	TargetNetProfit   *float64              `json:"target_net_profit,omitempty"`
	TargetTotalProfit *float64              `json:"target_total_profit,omitempty"`
	TargetKnifeRatio  *float64              `json:"target_knife_ratio,omitempty"`
}

type recordsResponse struct {
	Records    []profitRecordResponse `json:"records"`
	Pagination *paginationResponse    `json:"pagination,omitempty"`
}

type paginationResponse struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

type itemsResponse struct {
	Items []itemSummaryResponse `json:"items"`
}

type profitRecordResponse struct {
	ID          int64   `json:"id"`
	CreatedAt   string  `json:"created_at"`
	ItemName    string  `json:"item_name"`
	BuyPrice    float64 `json:"buy_price"`
	SellPrice   float64 `json:"sell_price"`
	Quantity    int     `json:"quantity"`
	NetProfit   float64 `json:"net_profit"`
	TotalProfit float64 `json:"total_profit"`
	KnifeRatio  float64 `json:"knife_ratio"`
	Fee         string  `json:"fee"`
}

type itemSummaryResponse struct {
	ItemID        string `json:"item_id"`
	CanonicalName string `json:"canonical_name"`
	AppName       string `json:"app_name"`
	QueryTime     string `json:"query_time"`
	SalePriceText string `json:"sale_price_text"`
	SellPriceText string `json:"sell_price_text"`
	SellListings  int    `json:"sell_listings"`
}

// NewHandler creates the complete HTTP application handler.
func NewHandler(lookupClient lookup.Client, recordStore RecordStore) (http.Handler, error) {
	return NewHandlerWithOptions(lookupClient, recordStore, nil, HandlerOptions{})
}

// NewHandlerWithAssistant creates the HTTP application with an optional AI strategy assistant.
func NewHandlerWithAssistant(lookupClient lookup.Client, recordStore RecordStore, assistantService *assistant.Service) (http.Handler, error) {
	return NewHandlerWithOptions(lookupClient, recordStore, assistantService, HandlerOptions{})
}

// NewHandlerWithOptions creates the HTTP application with explicit runtime observability.
func NewHandlerWithOptions(lookupClient lookup.Client, recordStore RecordStore, assistantService *assistant.Service, options HandlerOptions) (http.Handler, error) {
	if lookupClient == nil {
		return nil, fmt.Errorf("lookup client is required")
	}
	if recordStore == nil {
		return nil, fmt.Errorf("record store is required")
	}
	if assistantService == nil {
		toolRunner, err := assistant.NewToolRunner(lookupClient)
		if err != nil {
			return nil, fmt.Errorf("create disabled assistant tools: %w", err)
		}
		assistantService, err = assistant.NewService(
			assistant.NewUnavailableModel("disabled", "", "AI 策略助手未启用"),
			toolRunner,
		)
		if err != nil {
			return nil, fmt.Errorf("create disabled assistant: %w", err)
		}
	}

	web, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		return nil, fmt.Errorf("load embedded web application: %w", err)
	}
	metrics := options.Metrics
	if metrics == nil {
		metrics = observability.NewMetrics()
	}
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	access, err := newAccessController(options.Access, metrics)
	if err != nil {
		return nil, fmt.Errorf("configure hosted demo access: %w", err)
	}

	app := &server{
		lookupService: lookup.NewService(lookupClient),
		recordStore:   recordStore,
		assistant:     assistantService,
		metrics:       metrics,
		access:        access,
		static:        http.FileServer(http.FS(web)),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", app.handleHealth)
	mux.HandleFunc("/api/v1/metrics", app.handleMetrics)
	mux.HandleFunc("/api/v1/items", app.handleItems)
	mux.HandleFunc("/api/v1/calculations", app.handleCalculations)
	mux.HandleFunc("/api/v1/reverse-calculations", app.handleReverseCalculations)
	mux.HandleFunc("/api/v1/records", app.handleRecords)
	mux.HandleFunc("/api/v1/assistant/status", app.handleAssistantStatus)
	mux.HandleFunc("/api/v1/assistant/stream", app.handleAssistantStream)
	mux.HandleFunc("/api/", app.handleUnknownAPI)
	mux.Handle("/", app.static)

	return metrics.HTTPMiddleware(logger, securityHeaders(access.Middleware(mux))), nil
}

func (s *server) handleAssistantStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: s.assistant.Status()})
}

func (s *server) handleAssistantStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !s.assistant.Status().Available {
		writeAPIError(w, http.StatusServiceUnavailable, "assistant_unavailable", s.assistant.Status().Reason)
		return
	}

	var request assistantRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := assistant.ValidateMessage(request.Message); err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	if _, err := assistant.ValidateHistory(request.History); err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "streaming_unsupported", "response streaming is unavailable")
		return
	}
	if allowed, retryAfter := s.access.allowAssistantRun(); !allowed {
		s.metrics.AccessAssistantQuotaRejected()
		writeRetryAfter(w, retryAfter)
		writeAPIError(w, http.StatusTooManyRequests, "assistant_quota_exceeded", "AI assistant run quota exceeded; retry later")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	s.metrics.AssistantStarted()
	err := s.assistant.RunConversation(ctx, request.Message, request.History, func(event assistant.Event) error {
		if event.Completion != nil {
			s.metrics.ObserveAssistantCompletion(observability.AssistantCompletion{
				ElapsedMS:    event.Completion.ElapsedMS,
				ToolCalls:    event.Completion.ToolCount,
				InputTokens:  event.Completion.Usage.InputTokens,
				OutputTokens: event.Completion.Usage.OutputTokens,
				TotalTokens:  event.Completion.Usage.TotalTokens,
			})
		}
		if err := writeSSE(w, event.Type, event); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})
	if err != nil {
		s.metrics.AssistantFailed()
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		_ = writeSSE(w, "error", map[string]any{
			"error": apiError{Code: "assistant_error", Message: "AI 策略助手暂时无法完成请求", Retryable: true},
		})
		flusher.Flush()
	}
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if checker, ok := s.recordStore.(storage.HealthChecker); ok {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := checker.HealthCheck(ctx); err != nil {
			writeAPIError(w, http.StatusServiceUnavailable, "dependency_unavailable", "required persistence is unavailable")
			return
		}
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: healthResponse{Status: "ok"}})
}

func (s *server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: s.metrics.Snapshot()})
}

func (s *server) handleItems(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "query is required")
		return
	}

	var limit *int
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 || parsed > maxLookupLimit {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "limit must be an integer between 1 and 50")
			return
		}
		limit = &parsed
	}

	items, err := s.lookupService.Lookup(r.Context(), domain.ItemLookupRequest{Query: query, Limit: limit})
	if err != nil {
		if errors.Is(err, lookup.ErrNoMatches) {
			writeAPIError(w, http.StatusNotFound, "no_matches", err.Error())
			return
		}
		if errors.Is(err, lookup.ErrSteamOfficialRateLimit) {
			writeAPIError(w, http.StatusBadGateway, "official_provider_rate_limited", "Steam official market API is rate limited; configure a supported third-party Steam Market Provider")
			return
		}
		writeAPIError(w, http.StatusBadGateway, "provider_error", "market data provider did not complete the lookup")
		return
	}

	writeJSON(w, http.StatusOK, dataEnvelope{Data: itemsResponse{Items: mapItemSummaries(items)}})
}

func (s *server) handleCalculations(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var request calculationRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	records, err := calculation.Calculate(domain.CalculationRequest{
		ItemName:   request.ItemName,
		BuyPrice:   request.BuyPrice,
		Quantity:   request.Quantity,
		Multiplier: request.Multiplier,
		SellPrices: request.SellPrices,
	})
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	stored, err := s.recordStore.Append(r.Context(), records)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "storage_error", "unable to save calculation records")
		return
	}

	writeJSON(w, http.StatusCreated, dataEnvelope{Data: recordsResponse{Records: mapStoredProfitRecords(stored)}})
}

func (s *server) handleReverseCalculations(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var request reverseCalculationRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	record, err := calculation.CalculateReverse(domain.ReverseCalculationRequest{
		ItemName:          request.ItemName,
		SolveDirection:    request.SolveDirection,
		BuyPrice:          request.BuyPrice,
		SellPrice:         request.SellPrice,
		Quantity:          request.Quantity,
		Multiplier:        request.Multiplier,
		TargetNetProfit:   request.TargetNetProfit,
		TargetTotalProfit: request.TargetTotalProfit,
		TargetKnifeRatio:  request.TargetKnifeRatio,
	})
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	stored, err := s.recordStore.Append(r.Context(), []domain.ProfitRecord{record})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "storage_error", "unable to save calculation record")
		return
	}

	writeJSON(w, http.StatusCreated, dataEnvelope{Data: recordsResponse{Records: mapStoredProfitRecords(stored)}})
}

func (s *server) handleRecords(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	query, err := parseRecordQuery(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	page, err := s.recordStore.Query(r.Context(), query)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "storage_error", "unable to load calculation records")
		return
	}

	writeJSON(w, http.StatusOK, dataEnvelope{Data: recordsResponse{
		Records: mapStoredProfitRecords(page.Records),
		Pagination: &paginationResponse{
			Total:   page.Total,
			Limit:   page.Limit,
			Offset:  page.Offset,
			HasMore: page.Offset+len(page.Records) < page.Total,
		},
	}})
}

func parseRecordQuery(r *http.Request) (storage.RecordQuery, error) {
	values := r.URL.Query()
	allowed := map[string]bool{"item": true, "outcome": true, "order": true, "limit": true, "offset": true}
	for key, entries := range values {
		if !allowed[key] {
			return storage.RecordQuery{}, fmt.Errorf("unsupported query parameter %q", key)
		}
		if len(entries) != 1 {
			return storage.RecordQuery{}, fmt.Errorf("query parameter %q must be supplied once", key)
		}
	}

	itemPrefix := strings.TrimSpace(values.Get("item"))
	if utf8.RuneCountInString(itemPrefix) > maxItemPrefix {
		return storage.RecordQuery{}, fmt.Errorf("item must not exceed %d characters", maxItemPrefix)
	}
	query := storage.RecordQuery{
		ItemPrefix: itemPrefix,
		Outcome:    storage.RecordOutcome(strings.TrimSpace(values.Get("outcome"))),
		Order:      storage.RecordOrder(strings.TrimSpace(values.Get("order"))),
	}
	if rawLimit := strings.TrimSpace(values.Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return storage.RecordQuery{}, fmt.Errorf("limit must be an integer between 1 and %d", storage.MaxRecordLimit)
		}
		query.Limit = limit
	}
	if rawOffset := strings.TrimSpace(values.Get("offset")); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil {
			return storage.RecordQuery{}, fmt.Errorf("offset must be a non-negative integer")
		}
		query.Offset = offset
	}
	return storage.NormalizeRecordQuery(query)
}

func (s *server) handleUnknownAPI(w http.ResponseWriter, _ *http.Request) {
	writeAPIError(w, http.StatusNotFound, "not_found", "API route not found")
}

func requireMethod(w http.ResponseWriter, r *http.Request, allowed string) bool {
	if r.Method == allowed {
		return true
	}
	w.Header().Set("Allow", allowed)
	writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	return false
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	if contentType != "application/json" {
		return fmt.Errorf("Content-Type must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("request body must contain a single JSON object")
		}
		return fmt.Errorf("decode trailing request body: %w", err)
	}
	return nil
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSSE(w io.Writer, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode server-sent event: %w", err)
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return fmt.Errorf("write server-sent event: %w", err)
	}
	return nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; script-src 'self'; style-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func mapStoredProfitRecords(records []storage.StoredProfitRecord) []profitRecordResponse {
	mapped := make([]profitRecordResponse, 0, len(records))
	for _, stored := range records {
		record := stored.Record
		mapped = append(mapped, profitRecordResponse{
			ID:          stored.ID,
			CreatedAt:   stored.CreatedAt.UTC().Format(time.RFC3339Nano),
			ItemName:    record.ItemName,
			BuyPrice:    canonicalNumber(domain.FormatPrice(record.BuyPrice)),
			SellPrice:   canonicalNumber(domain.FormatPrice(record.SellPrice)),
			Quantity:    record.Quantity,
			NetProfit:   canonicalNumber(domain.FormatPrice(record.NetProfit)),
			TotalProfit: canonicalNumber(domain.FormatPrice(record.TotalProfit)),
			KnifeRatio:  canonicalNumber(domain.FormatRatio(record.KnifeRatio)),
			Fee:         record.Fee,
		})
	}
	return mapped
}

func canonicalNumber(formatted string) float64 {
	value, err := strconv.ParseFloat(formatted, 64)
	if err != nil {
		return 0
	}
	return value
}

func mapItemSummaries(items []domain.ItemSummary) []itemSummaryResponse {
	mapped := make([]itemSummaryResponse, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, itemSummaryResponse{
			ItemID:        item.ItemID,
			CanonicalName: item.CanonicalName,
			AppName:       item.AppName,
			QueryTime:     item.QueryTime,
			SalePriceText: item.SalePriceText,
			SellPriceText: item.SellPriceText,
			SellListings:  item.SellListings,
		})
	}
	return mapped
}
