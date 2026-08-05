package calculation

import (
	"errors"
	"fmt"
	"strings"

	"steam-profit-calculator/internal/domain"
)

var (
	ErrEmptyItemName     = errors.New("item name must not be empty")
	ErrInvalidBuy        = errors.New("buy price must be greater than 0")
	ErrInvalidQuantity   = errors.New("quantity must be greater than 0")
	ErrInvalidMultiplier = errors.New("multiplier must be greater than 0 and less than or equal to 1")
	ErrMissingSells      = errors.New("at least one sell price is required")
)

// ValidateRequest validates one calculation request before formula execution.
func ValidateRequest(req domain.CalculationRequest) error {
	if err := validateItemName(req.ItemName); err != nil {
		return err
	}
	if req.BuyPrice <= 0 {
		return ErrInvalidBuy
	}
	if err := validateQuantity(req.Quantity); err != nil {
		return err
	}
	if err := validateMultiplier(req.Multiplier); err != nil {
		return err
	}
	if len(req.SellPrices) == 0 {
		return ErrMissingSells
	}

	for i, sellPrice := range req.SellPrices {
		if sellPrice <= 0 {
			return fmt.Errorf("sell price at index %d must be greater than 0", i)
		}
	}

	return nil
}

// Calculate returns one profit record for each sell price in the request.
func Calculate(req domain.CalculationRequest) ([]domain.ProfitRecord, error) {
	if err := ValidateRequest(req); err != nil {
		return nil, err
	}

	itemName := strings.TrimSpace(req.ItemName)
	quantity := effectiveQuantity(req.Quantity)
	multiplier := effectiveMultiplier(req.Multiplier)
	feeLabel := effectiveFeeLabel(req.Multiplier)
	records := make([]domain.ProfitRecord, 0, len(req.SellPrices))
	for _, sellPrice := range req.SellPrices {
		records = append(records, buildRecord(itemName, req.BuyPrice, sellPrice, quantity, multiplier, feeLabel))
	}

	return records, nil
}

func validateItemName(itemName string) error {
	if strings.TrimSpace(itemName) == "" {
		return ErrEmptyItemName
	}

	return nil
}

func validateQuantity(quantity *int) error {
	if quantity != nil && *quantity <= 0 {
		return ErrInvalidQuantity
	}

	return nil
}

func validateMultiplier(multiplier *float64) error {
	if multiplier != nil && (*multiplier <= 0 || *multiplier > 1) {
		return ErrInvalidMultiplier
	}

	return nil
}

func effectiveQuantity(quantity *int) int {
	if quantity != nil {
		return *quantity
	}

	return domain.DefaultQuantity()
}

func effectiveMultiplier(multiplier *float64) float64 {
	if multiplier != nil {
		return *multiplier
	}

	return domain.DefaultMultiplier()
}

func effectiveFeeLabel(multiplier *float64) string {
	if multiplier == nil {
		return domain.DefaultFeeLabel()
	}

	return domain.FormatMultiplier(*multiplier)
}

func buildRecord(itemName string, buyPrice, sellPrice float64, quantity int, multiplier float64, feeLabel string) domain.ProfitRecord {
	netSell := sellPrice * multiplier
	netProfit := netSell - buyPrice

	return domain.ProfitRecord{
		ItemName:    itemName,
		BuyPrice:    buyPrice,
		SellPrice:   sellPrice,
		Quantity:    quantity,
		NetProfit:   netProfit,
		TotalProfit: netProfit * float64(quantity),
		KnifeRatio:  buyPrice / netSell,
		Fee:         feeLabel,
	}
}
