package webapi

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"steam-profit-calculator/internal/calculation"
	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/lookup"
)

const (
	maxJSONBodyBytes = 1 << 20
	maxLookupLimit   = 50
)

//go:embed web/*
var embeddedWeb embed.FS

// RecordStore is the persistence boundary required by the web application.
type RecordStore interface {
	Append(records []domain.ProfitRecord) error
	List() ([]domain.ProfitRecord, error)
}

type server struct {
	lookupService *lookup.Service
	recordStore   RecordStore
	static        http.Handler
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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
	Records []profitRecordResponse `json:"records"`
}

type itemsResponse struct {
	Items []itemSummaryResponse `json:"items"`
}

type profitRecordResponse struct {
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
	if lookupClient == nil {
		return nil, fmt.Errorf("lookup client is required")
	}
	if recordStore == nil {
		return nil, fmt.Errorf("record store is required")
	}

	web, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		return nil, fmt.Errorf("load embedded web application: %w", err)
	}

	app := &server{
		lookupService: lookup.NewService(lookupClient),
		recordStore:   recordStore,
		static:        http.FileServer(http.FS(web)),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", app.handleHealth)
	mux.HandleFunc("/api/v1/items", app.handleItems)
	mux.HandleFunc("/api/v1/calculations", app.handleCalculations)
	mux.HandleFunc("/api/v1/reverse-calculations", app.handleReverseCalculations)
	mux.HandleFunc("/api/v1/records", app.handleRecords)
	mux.HandleFunc("/api/", app.handleUnknownAPI)
	mux.Handle("/", app.static)

	return securityHeaders(mux), nil
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: healthResponse{Status: "ok"}})
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
		writeAPIError(w, http.StatusBadGateway, "provider_error", "item provider is temporarily unavailable")
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

	if err := s.recordStore.Append(records); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "storage_error", "unable to save calculation records")
		return
	}

	writeJSON(w, http.StatusCreated, dataEnvelope{Data: recordsResponse{Records: mapProfitRecords(records)}})
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

	if err := s.recordStore.Append([]domain.ProfitRecord{record}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "storage_error", "unable to save calculation record")
		return
	}

	writeJSON(w, http.StatusCreated, dataEnvelope{Data: recordsResponse{Records: mapProfitRecords([]domain.ProfitRecord{record})}})
}

func (s *server) handleRecords(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	records, err := s.recordStore.List()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "storage_error", "unable to load calculation records")
		return
	}

	writeJSON(w, http.StatusOK, dataEnvelope{Data: recordsResponse{Records: mapProfitRecords(records)}})
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

func mapProfitRecords(records []domain.ProfitRecord) []profitRecordResponse {
	mapped := make([]profitRecordResponse, 0, len(records))
	for _, record := range records {
		mapped = append(mapped, profitRecordResponse{
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
