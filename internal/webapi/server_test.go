package webapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"steam-profit-calculator/internal/assistant"
	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/lookup"
	"steam-profit-calculator/internal/webapi"
)

type stubLookupClient struct {
	items   []domain.ItemSummary
	err     error
	request domain.ItemLookupRequest
}

func (s *stubLookupClient) Lookup(_ context.Context, request domain.ItemLookupRequest) ([]domain.ItemSummary, error) {
	s.request = request
	return s.items, s.err
}

type memoryRecordStore struct {
	records   []domain.ProfitRecord
	appendErr error
	listErr   error
}

type stubAssistantModel struct {
	turns int
	err   error
}

func (s *stubAssistantModel) Status() assistant.Status {
	return assistant.Status{Available: true, Provider: "fake", Model: "fake-1", PromptVersion: assistant.PromptVersion}
}

func (s *stubAssistantModel) Respond(_ context.Context, _ assistant.ModelRequest) (assistant.ModelResponse, error) {
	s.turns++
	if s.err != nil {
		return assistant.ModelResponse{}, s.err
	}
	if s.turns == 1 {
		return assistant.ModelResponse{
			State: "turn-1",
			ToolCalls: []assistant.ToolCall{{
				CallID: "call-1", Name: "calculate_profit",
				Arguments: `{"item_name":"Capsule","buy_price":0.48,"quantity":1,"multiplier":0.837,"sell_prices":[0.8]}`,
			}},
		}, nil
	}
	return assistant.ModelResponse{Text: "计算结果已由工具验证 [T1]。仅供研究，不构成投资或交易建议。", RequestID: "fake-request"}, nil
}

func (s *memoryRecordStore) Append(records []domain.ProfitRecord) error {
	if s.appendErr != nil {
		return s.appendErr
	}
	s.records = append(s.records, records...)
	return nil
}

func (s *memoryRecordStore) List() ([]domain.ProfitRecord, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]domain.ProfitRecord(nil), s.records...), nil
}

func TestHealthEndpointAndSecurityHeaders(t *testing.T) {
	handler := newTestHandler(t, &stubLookupClient{}, &memoryRecordStore{})
	recorder := performRequest(handler, http.MethodGet, "/api/health", "", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("body = %q, want healthy response", recorder.Body.String())
	}
}

func TestAssistantStatusDefaultsToUnavailableWithoutBreakingHealth(t *testing.T) {
	handler := newTestHandler(t, &stubLookupClient{}, &memoryRecordStore{})
	status := performRequest(handler, http.MethodGet, "/api/v1/assistant/status", "", "")
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"available":false`) {
		t.Fatalf("status response = %d %s", status.Code, status.Body.String())
	}
	health := performRequest(handler, http.MethodGet, "/api/health", "", "")
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}
}

func TestAssistantUnavailableRejectsStreamBeforeStartingSSE(t *testing.T) {
	handler := newTestHandler(t, &stubLookupClient{}, &memoryRecordStore{})
	recorder := performRequest(handler, http.MethodPost, "/api/v1/assistant/stream", `{"message":"calculate"}`, "application/json")
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"assistant_unavailable"`) {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestAssistantStreamEmitsEvidenceWithoutPersistingRecords(t *testing.T) {
	store := &memoryRecordStore{}
	model := &stubAssistantModel{}
	handler := newAssistantTestHandler(t, &stubLookupClient{}, store, model)
	recorder := performRequest(handler, http.MethodPost, "/api/v1/assistant/stream", `{"message":"calculate Capsule"}`, "application/json")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q", got)
	}
	body := recorder.Body.String()
	for _, event := range []string{"event: status", "event: tool_start", "event: evidence", "event: delta", "event: done"} {
		if !strings.Contains(body, event) {
			t.Fatalf("body missing %q: %s", event, body)
		}
	}
	if !strings.Contains(body, `"grounded":true`) || !strings.Contains(body, `"total_profit":0.19`) {
		t.Fatalf("body = %s", body)
	}
	if len(store.records) != 0 {
		t.Fatalf("persisted records = %d, want 0", len(store.records))
	}
}

func TestAssistantStreamRejectsInvalidJSONWithoutCallingModel(t *testing.T) {
	model := &stubAssistantModel{}
	handler := newAssistantTestHandler(t, &stubLookupClient{}, &memoryRecordStore{}, model)
	recorder := performRequest(handler, http.MethodPost, "/api/v1/assistant/stream", `{"message":"hello","unknown":true}`, "application/json")
	if recorder.Code != http.StatusBadRequest || model.turns != 0 {
		t.Fatalf("status = %d, model turns = %d, body = %s", recorder.Code, model.turns, recorder.Body.String())
	}
}

func TestAssistantProviderFailureBecomesSafeStreamError(t *testing.T) {
	model := &stubAssistantModel{err: errors.New("secret provider detail")}
	handler := newAssistantTestHandler(t, &stubLookupClient{}, &memoryRecordStore{}, model)
	recorder := performRequest(handler, http.MethodPost, "/api/v1/assistant/stream", `{"message":"hello"}`, "application/json")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "event: error") {
		t.Fatalf("status/body = %d %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "secret provider detail") {
		t.Fatalf("provider detail leaked: %s", recorder.Body.String())
	}
}

func TestCalculationEndpointPersistsAndReturnsRecords(t *testing.T) {
	store := &memoryRecordStore{}
	handler := newTestHandler(t, &stubLookupClient{}, store)
	body := `{"item_name":"Capsule","buy_price":0.48,"quantity":2,"multiplier":0.85,"sell_prices":[0.7,0.8]}`
	recorder := performRequest(handler, http.MethodPost, "/api/v1/calculations", body, "application/json")

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if len(store.records) != 2 {
		t.Fatalf("persisted records = %d, want 2", len(store.records))
	}
	if got := store.records[1].TotalProfit; got < 0.399999 || got > 0.400001 {
		t.Fatalf("second TotalProfit = %v, want approximately 0.4", got)
	}

	payload := decodeResponse(t, recorder)
	data := payload["data"].(map[string]any)
	records := data["records"].([]any)
	if len(records) != 2 {
		t.Fatalf("response records = %d, want 2", len(records))
	}
	second := records[1].(map[string]any)
	if got := second["total_profit"].(float64); got != 0.4 {
		t.Fatalf("response total_profit = %v, want canonical 0.4", got)
	}
}

func TestCalculationEndpointRejectsUnknownFieldWithoutPersistence(t *testing.T) {
	store := &memoryRecordStore{}
	handler := newTestHandler(t, &stubLookupClient{}, store)
	body := `{"item_name":"Capsule","buy_price":0.48,"sell_prices":[0.8],"invented":true}`
	recorder := performRequest(handler, http.MethodPost, "/api/v1/calculations", body, "application/json")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if len(store.records) != 0 {
		t.Fatalf("persisted records = %d, want 0", len(store.records))
	}
	if !strings.Contains(recorder.Body.String(), `"code":"invalid_json"`) {
		t.Fatalf("body = %q, want invalid_json", recorder.Body.String())
	}
}

func TestCalculationEndpointRejectsDomainValidationFailure(t *testing.T) {
	store := &memoryRecordStore{}
	handler := newTestHandler(t, &stubLookupClient{}, store)
	body := `{"item_name":"Capsule","buy_price":0.48,"sell_prices":[]}`
	recorder := performRequest(handler, http.MethodPost, "/api/v1/calculations", body, "application/json")

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
	if len(store.records) != 0 {
		t.Fatalf("persisted records = %d, want 0", len(store.records))
	}
	if !strings.Contains(recorder.Body.String(), "at least one sell price is required") {
		t.Fatalf("body = %q, want domain validation message", recorder.Body.String())
	}
}

func TestCalculationEndpointReportsStorageFailure(t *testing.T) {
	store := &memoryRecordStore{appendErr: errors.New("disk unavailable")}
	handler := newTestHandler(t, &stubLookupClient{}, store)
	body := `{"item_name":"Capsule","buy_price":0.48,"sell_prices":[0.8]}`
	recorder := performRequest(handler, http.MethodPost, "/api/v1/calculations", body, "application/json")

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"storage_error"`) {
		t.Fatalf("body = %q, want storage_error", recorder.Body.String())
	}
}

func TestReverseCalculationEndpointPersistsSolvedRecord(t *testing.T) {
	store := &memoryRecordStore{}
	handler := newTestHandler(t, &stubLookupClient{}, store)
	body := `{"item_name":"Capsule","solve_direction":"sell","buy_price":0.48,"quantity":2,"target_total_profit":0.4}`
	recorder := performRequest(handler, http.MethodPost, "/api/v1/reverse-calculations", body, "application/json")

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if len(store.records) != 1 {
		t.Fatalf("persisted records = %d, want 1", len(store.records))
	}
	if got := store.records[0].TotalProfit; got < 0.399999 || got > 0.400001 {
		t.Fatalf("TotalProfit = %v, want approximately 0.4", got)
	}
}

func TestRecordsEndpointReturnsPersistedHistory(t *testing.T) {
	store := &memoryRecordStore{records: []domain.ProfitRecord{{
		ItemName:    "Capsule",
		BuyPrice:    0.48,
		SellPrice:   0.8,
		Quantity:    1,
		NetProfit:   0.1896,
		TotalProfit: 0.1896,
		KnifeRatio:  0.716845878,
		Fee:         "default",
	}}}
	handler := newTestHandler(t, &stubLookupClient{}, store)
	recorder := performRequest(handler, http.MethodGet, "/api/v1/records", "", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"item_name":"Capsule"`) {
		t.Fatalf("body = %q, want persisted record", recorder.Body.String())
	}
}

func TestItemsEndpointUsesNormalizedLookupService(t *testing.T) {
	client := &stubLookupClient{items: []domain.ItemSummary{{
		ItemID:        "123",
		CanonicalName: "AK-47 | Redline",
		AppName:       "Counter-Strike 2",
		QueryTime:     "2026-08-06T00:00:00Z",
		SellPriceText: "$12.50",
		SellListings:  42,
	}}}
	handler := newTestHandler(t, client, &memoryRecordStore{})
	recorder := performRequest(handler, http.MethodGet, "/api/v1/items?query=AK-47&limit=3", "", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request.Query != "AK-47" {
		t.Fatalf("lookup query = %q, want AK-47", client.request.Query)
	}
	if client.request.Limit == nil || *client.request.Limit != 3 {
		t.Fatalf("lookup limit = %v, want 3", client.request.Limit)
	}
	if !strings.Contains(recorder.Body.String(), `"canonical_name":"AK-47 | Redline"`) {
		t.Fatalf("body = %q, want normalized item", recorder.Body.String())
	}
}

func TestItemsEndpointDistinguishesNoMatchesAndProviderFailure(t *testing.T) {
	tests := []struct {
		name       string
		client     *stubLookupClient
		wantStatus int
		wantCode   string
	}{
		{name: "no matches", client: &stubLookupClient{}, wantStatus: http.StatusNotFound, wantCode: "no_matches"},
		{name: "provider failure", client: &stubLookupClient{err: errors.New("upstream down")}, wantStatus: http.StatusBadGateway, wantCode: "provider_error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newTestHandler(t, test.client, &memoryRecordStore{})
			recorder := performRequest(handler, http.MethodGet, "/api/v1/items?query=AK", "", "")
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if !strings.Contains(recorder.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("body = %q, want code %q", recorder.Body.String(), test.wantCode)
			}
		})
	}
}

func TestItemsEndpointRejectsExcessiveLimit(t *testing.T) {
	handler := newTestHandler(t, &stubLookupClient{}, &memoryRecordStore{})
	recorder := performRequest(handler, http.MethodGet, "/api/v1/items?query=AK&limit=51", "", "")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), "between 1 and 50") {
		t.Fatalf("body = %q, want bounded-limit message", recorder.Body.String())
	}
}

func TestAPIRejectsUnsupportedMethods(t *testing.T) {
	handler := newTestHandler(t, &stubLookupClient{}, &memoryRecordStore{})
	recorder := performRequest(handler, http.MethodDelete, "/api/v1/records", "", "")

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", got)
	}
}

func TestEmbeddedBrowserApplicationIsServed(t *testing.T) {
	handler := newTestHandler(t, &stubLookupClient{}, &memoryRecordStore{})
	recorder := performRequest(handler, http.MethodGet, "/", "", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "Steam Profit Lab") {
		t.Fatalf("body does not contain application title")
	}
	if got := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Fatalf("Content-Security-Policy = %q, want same-origin policy", got)
	}
}

func newTestHandler(t *testing.T, lookupClient lookup.Client, store webapi.RecordStore) http.Handler {
	t.Helper()
	handler, err := webapi.NewHandler(lookupClient, store)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

func newAssistantTestHandler(t *testing.T, lookupClient lookup.Client, store webapi.RecordStore, model assistant.Model) http.Handler {
	t.Helper()
	runner, err := assistant.NewToolRunner(lookupClient)
	if err != nil {
		t.Fatalf("NewToolRunner() error = %v", err)
	}
	service, err := assistant.NewService(model, runner)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler, err := webapi.NewHandlerWithAssistant(lookupClient, store, service)
	if err != nil {
		t.Fatalf("NewHandlerWithAssistant() error = %v", err)
	}
	return handler
}

func performRequest(handler http.Handler, method, target, body, contentType string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}
