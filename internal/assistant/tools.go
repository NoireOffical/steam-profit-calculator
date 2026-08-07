package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"steam-profit-calculator/internal/calculation"
	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/lookup"
)

const (
	toolLookupItems     = "lookup_items"
	toolCalculateProfit = "calculate_profit"
	toolSolveTarget     = "solve_target"
	maxToolArguments    = 64 << 10
)

// ToolRunner executes the assistant's allow-listed, read-only project tools.
type ToolRunner struct {
	lookupService *lookup.Service
}

type toolExecution struct {
	Output   ToolOutput
	Evidence Evidence
}

type lookupArguments struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type calculateArguments struct {
	ItemName   string    `json:"item_name"`
	BuyPrice   float64   `json:"buy_price"`
	Quantity   int       `json:"quantity"`
	Multiplier float64   `json:"multiplier"`
	SellPrices []float64 `json:"sell_prices"`
}

type solveArguments struct {
	ItemName          string                `json:"item_name"`
	SolveDirection    domain.SolveDirection `json:"solve_direction"`
	BuyPrice          *float64              `json:"buy_price"`
	SellPrice         *float64              `json:"sell_price"`
	Quantity          int                   `json:"quantity"`
	Multiplier        float64               `json:"multiplier"`
	TargetNetProfit   *float64              `json:"target_net_profit"`
	TargetTotalProfit *float64              `json:"target_total_profit"`
	TargetKnifeRatio  *float64              `json:"target_knife_ratio"`
}

type toolError struct {
	EvidenceID string `json:"evidence_id"`
	Error      struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type toolRecords struct {
	EvidenceID string         `json:"evidence_id"`
	Records    []profitRecord `json:"records"`
}

type toolItems struct {
	EvidenceID string        `json:"evidence_id"`
	Items      []itemSummary `json:"items"`
}

type profitRecord struct {
	ItemName    string  `json:"item_name"`
	BuyPrice    float64 `json:"buy_price"`
	SellPrice   float64 `json:"sell_price"`
	Quantity    int     `json:"quantity"`
	NetProfit   float64 `json:"net_profit"`
	TotalProfit float64 `json:"total_profit"`
	KnifeRatio  float64 `json:"knife_ratio"`
	Fee         string  `json:"fee"`
}

type itemSummary struct {
	ItemID        string `json:"item_id"`
	CanonicalName string `json:"canonical_name"`
	AppName       string `json:"app_name"`
	QueryTime     string `json:"query_time"`
	SalePriceText string `json:"sale_price_text"`
	SellPriceText string `json:"sell_price_text"`
	SellListings  int    `json:"sell_listings"`
}

// NewToolRunner creates the read-only tool executor used by the assistant.
func NewToolRunner(lookupClient lookup.Client) (*ToolRunner, error) {
	if lookupClient == nil {
		return nil, fmt.Errorf("lookup client is required")
	}
	return &ToolRunner{lookupService: lookup.NewService(lookupClient)}, nil
}

// ToolDefinitions returns stable strict schemas for all assistant tools.
func ToolDefinitions() []ToolDefinition {
	nullableNumber := []string{"number", "null"}
	return []ToolDefinition{
		{
			Type:        "function",
			Name:        toolLookupItems,
			Description: "查询 Steam 市场快照。成功返回 evidence_id 和 items（规范名称、应用、查询时间、价格文本、挂牌数）；失败返回 evidence_id 和结构化 error。不执行计算。",
			Strict:      true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "商品关键词"},
					"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 10, "description": "返回结果数"},
				},
				"required":             []string{"query", "limit"},
				"additionalProperties": false,
			},
		},
		{
			Type:        "function",
			Name:        toolCalculateProfit,
			Description: "调用确定性引擎比较收益。成功返回 evidence_id 和 records（买卖价、数量、单件/总收益、挂刀比例、费率标签）；失败返回结构化 error。不会保存记录。",
			Strict:      true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"item_name":  map[string]any{"type": "string"},
					"buy_price":  map[string]any{"type": "number", "exclusiveMinimum": 0},
					"quantity":   map[string]any{"type": "integer", "minimum": 1, "maximum": 100000},
					"multiplier": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 1},
					"sell_prices": map[string]any{
						"type": "array", "items": map[string]any{"type": "number", "exclusiveMinimum": 0}, "minItems": 1, "maxItems": 20,
					},
				},
				"required":             []string{"item_name", "buy_price", "quantity", "multiplier", "sell_prices"},
				"additionalProperties": false,
			},
		},
		{
			Type:        "function",
			Name:        toolSolveTarget,
			Description: "调用确定性引擎反推买价或卖价。成功返回 evidence_id 和单条 records；失败返回结构化 error。不会保存记录，未使用的可空参数必须传 null。",
			Strict:      true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"item_name":           map[string]any{"type": "string"},
					"solve_direction":     map[string]any{"type": "string", "enum": []string{"buy", "sell"}},
					"buy_price":           map[string]any{"type": nullableNumber},
					"sell_price":          map[string]any{"type": nullableNumber},
					"quantity":            map[string]any{"type": "integer", "minimum": 1, "maximum": 100000},
					"multiplier":          map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 1},
					"target_net_profit":   map[string]any{"type": nullableNumber},
					"target_total_profit": map[string]any{"type": nullableNumber},
					"target_knife_ratio":  map[string]any{"type": nullableNumber},
				},
				"required": []string{
					"item_name", "solve_direction", "buy_price", "sell_price", "quantity", "multiplier",
					"target_net_profit", "target_total_profit", "target_knife_ratio",
				},
				"additionalProperties": false,
			},
		},
	}
}

func (r *ToolRunner) execute(ctx context.Context, call ToolCall, evidenceID string) toolExecution {
	switch call.Name {
	case toolLookupItems:
		return r.executeLookup(ctx, call, evidenceID)
	case toolCalculateProfit:
		return r.executeCalculation(call, evidenceID)
	case toolSolveTarget:
		return r.executeReverse(call, evidenceID)
	default:
		return failedExecution(call, evidenceID, "unknown_tool", "model requested an unsupported tool")
	}
}

func (r *ToolRunner) executeLookup(ctx context.Context, call ToolCall, evidenceID string) toolExecution {
	var arguments lookupArguments
	if err := decodeArguments(call.Arguments, &arguments); err != nil {
		return failedExecution(call, evidenceID, "invalid_arguments", err.Error())
	}
	arguments.Query = strings.TrimSpace(arguments.Query)
	if arguments.Query == "" || arguments.Limit < 1 || arguments.Limit > 10 {
		return failedExecution(call, evidenceID, "invalid_arguments", "query is required and limit must be between 1 and 10")
	}

	items, err := r.lookupService.Lookup(ctx, domain.ItemLookupRequest{Query: arguments.Query, Limit: &arguments.Limit})
	if err != nil {
		if errors.Is(err, lookup.ErrNoMatches) {
			return failedExecution(call, evidenceID, "no_matches", "no matching items found")
		}
		if errors.Is(err, lookup.ErrSteamOfficialRateLimit) {
			return failedExecution(call, evidenceID, "official_provider_rate_limited", "Steam official market API is rate limited; configure a supported third-party Steam Market Provider")
		}
		return failedExecution(call, evidenceID, "provider_error", "market data provider did not complete the lookup")
	}

	result := toolItems{EvidenceID: evidenceID, Items: mapItems(items)}
	return successfulExecution(call, evidenceID, result)
}

func (r *ToolRunner) executeCalculation(call ToolCall, evidenceID string) toolExecution {
	var arguments calculateArguments
	if err := decodeArguments(call.Arguments, &arguments); err != nil {
		return failedExecution(call, evidenceID, "invalid_arguments", err.Error())
	}
	if arguments.Quantity > 100000 || len(arguments.SellPrices) > 20 {
		return failedExecution(call, evidenceID, "invalid_arguments", "quantity must not exceed 100000 and sell_prices must not exceed 20 values")
	}
	quantity := arguments.Quantity
	multiplier := arguments.Multiplier
	records, err := calculation.Calculate(domain.CalculationRequest{
		ItemName: arguments.ItemName, BuyPrice: arguments.BuyPrice, Quantity: &quantity,
		Multiplier: &multiplier, SellPrices: arguments.SellPrices,
	})
	if err != nil {
		return failedExecution(call, evidenceID, "validation_error", err.Error())
	}

	result := toolRecords{EvidenceID: evidenceID, Records: mapRecords(records)}
	return successfulExecution(call, evidenceID, result)
}

func (r *ToolRunner) executeReverse(call ToolCall, evidenceID string) toolExecution {
	var arguments solveArguments
	if err := decodeArguments(call.Arguments, &arguments); err != nil {
		return failedExecution(call, evidenceID, "invalid_arguments", err.Error())
	}
	if arguments.Quantity > 100000 {
		return failedExecution(call, evidenceID, "invalid_arguments", "quantity must not exceed 100000")
	}
	quantity := arguments.Quantity
	multiplier := arguments.Multiplier
	record, err := calculation.CalculateReverse(domain.ReverseCalculationRequest{
		ItemName: arguments.ItemName, SolveDirection: arguments.SolveDirection,
		BuyPrice: arguments.BuyPrice, SellPrice: arguments.SellPrice, Quantity: &quantity,
		Multiplier: &multiplier, TargetNetProfit: arguments.TargetNetProfit,
		TargetTotalProfit: arguments.TargetTotalProfit, TargetKnifeRatio: arguments.TargetKnifeRatio,
	})
	if err != nil {
		return failedExecution(call, evidenceID, "validation_error", err.Error())
	}

	result := toolRecords{EvidenceID: evidenceID, Records: mapRecords([]domain.ProfitRecord{record})}
	return successfulExecution(call, evidenceID, result)
}

func successfulExecution(call ToolCall, evidenceID string, payload any) toolExecution {
	data, err := json.Marshal(payload)
	if err != nil {
		return failedExecution(call, evidenceID, "serialization_error", "unable to encode tool result")
	}
	return toolExecution{
		Output:   ToolOutput{CallID: call.CallID, Output: string(data)},
		Evidence: Evidence{ID: evidenceID, Tool: call.Name, Status: "verified", Data: data, Verified: true},
	}
}

func failedExecution(call ToolCall, evidenceID, code, message string) toolExecution {
	payload := toolError{EvidenceID: evidenceID}
	payload.Error.Code = code
	payload.Error.Message = message
	data, _ := json.Marshal(payload)
	return toolExecution{
		Output:   ToolOutput{CallID: call.CallID, Output: string(data)},
		Evidence: Evidence{ID: evidenceID, Tool: call.Name, Status: "error", Data: data, Verified: false},
	}
}

func decodeArguments(raw string, destination any) error {
	if len(raw) == 0 || len(raw) > maxToolArguments {
		return fmt.Errorf("tool arguments must contain one bounded JSON object")
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode tool arguments: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("tool arguments must contain one JSON object")
	}
	return nil
}

func mapRecords(records []domain.ProfitRecord) []profitRecord {
	mapped := make([]profitRecord, 0, len(records))
	for _, record := range records {
		mapped = append(mapped, profitRecord{
			ItemName: record.ItemName, BuyPrice: canonicalNumber(domain.FormatPrice(record.BuyPrice)),
			SellPrice: canonicalNumber(domain.FormatPrice(record.SellPrice)), Quantity: record.Quantity,
			NetProfit:   canonicalNumber(domain.FormatPrice(record.NetProfit)),
			TotalProfit: canonicalNumber(domain.FormatPrice(record.TotalProfit)),
			KnifeRatio:  canonicalNumber(domain.FormatRatio(record.KnifeRatio)), Fee: record.Fee,
		})
	}
	return mapped
}

func mapItems(items []domain.ItemSummary) []itemSummary {
	mapped := make([]itemSummary, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, itemSummary{
			ItemID: item.ItemID, CanonicalName: item.CanonicalName, AppName: item.AppName,
			QueryTime: item.QueryTime, SalePriceText: item.SalePriceText,
			SellPriceText: item.SellPriceText, SellListings: item.SellListings,
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
