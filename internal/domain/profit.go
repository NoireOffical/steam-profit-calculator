package domain

import "strconv"

const (
        SteamNetMultiplier   = 0.837
        defaultQuantity      = 1
        defaultFeeLabel      = "default"
        pricePrecision       = 3
        ratioPrecision       = 6
        multiplierPrecision  = 3
)

var recordHeader = []string{
	"item_name",
	"buy_price",
	"sell_price",
	"quantity",
	"net_profit",
	"total_profit",
	"knife_ratio",
}

var storageRecordHeader = []string{
        "item_name",
        "buy_price",
        "sell_price",
        "quantity",
        "net_profit",
        "total_profit",
        "knife_ratio",
        "fee",
}

var legacyRecordHeader = []string{
	"item_name",
	"buy_price",
	"sell_price",
	"net_profit",
	"knife_ratio",
}

type SolveDirection string

const (
	SolveDirectionBuy  SolveDirection = "buy"
	SolveDirectionSell SolveDirection = "sell"
)

// CalculationRequest is the normalized input for one calculation command.
type CalculationRequest struct {
	ItemName   string
	BuyPrice   float64
	Quantity   *int
	Multiplier *float64
	SellPrices []float64
}

// ReverseCalculationRequest is the normalized input for one reverse-calculation command.
type ReverseCalculationRequest struct {
	ItemName          string
	SolveDirection    SolveDirection
	BuyPrice          *float64
	SellPrice         *float64
	Quantity          *int
	Multiplier        *float64
	TargetNetProfit   *float64
	TargetTotalProfit *float64
	TargetKnifeRatio  *float64
}

// ProfitRecord represents one computed sell-price scenario.
type ProfitRecord struct {
	ItemName    string
	BuyPrice    float64
	SellPrice   float64
	Quantity    int
	NetProfit   float64
	TotalProfit float64
	KnifeRatio  float64
        Fee         string
}

// RecordHeader returns the canonical field order used by display.
func RecordHeader() []string {
	header := make([]string, len(recordHeader))
	copy(header, recordHeader)
	return header
}

// StorageRecordHeader returns the canonical field order used by CSV persistence.
func StorageRecordHeader() []string {
        header := make([]string, len(storageRecordHeader))
        copy(header, storageRecordHeader)
        return header
}

// LegacyRecordHeader returns the previous on-disk header for backward compatibility.
func LegacyRecordHeader() []string {
	header := make([]string, len(legacyRecordHeader))
	copy(header, legacyRecordHeader)
	return header
}

// DefaultQuantity returns the compatibility default for omitted quantity values.
func DefaultQuantity() int {
	return defaultQuantity
}

// DefaultMultiplier returns the compatibility default for omitted multiplier values.
func DefaultMultiplier() float64 {
	return SteamNetMultiplier
}

// DefaultFeeLabel returns the persisted label for default multiplier usage.
func DefaultFeeLabel() string {
        return defaultFeeLabel
}

// Fields returns the canonical field order used by both storage and display.
func (r ProfitRecord) Fields() []string {
	return []string{
		r.ItemName,
		FormatPrice(r.BuyPrice),
		FormatPrice(r.SellPrice),
		strconv.Itoa(r.Quantity),
		FormatPrice(r.NetProfit),
		FormatPrice(r.TotalProfit),
		FormatRatio(r.KnifeRatio),
	}
}

// StorageFields returns the canonical field order used by CSV persistence.
func (r ProfitRecord) StorageFields() []string {
        return append(r.Fields(), r.Fee)
}

// CSVRow formats a record deterministically for CSV persistence.
func (r ProfitRecord) CSVRow() []string {
        return r.StorageFields()
}

// FormatPrice formats price-like values with fixed precision.
func FormatPrice(value float64) string {
	return strconv.FormatFloat(value, 'f', pricePrecision, 64)
}

// FormatRatio formats knife-ratio values with fixed precision.
func FormatRatio(value float64) string {
	return strconv.FormatFloat(value, 'f', ratioPrecision, 64)
}

// FormatMultiplier formats multiplier values with fixed precision.
func FormatMultiplier(value float64) string {
        return strconv.FormatFloat(value, 'f', multiplierPrecision, 64)
}
