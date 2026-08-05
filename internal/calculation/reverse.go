package calculation

import (
	"errors"
	"strings"

	"steam-profit-calculator/internal/domain"
)

var (
	ErrMissingSolveDirection      = errors.New("solve direction must be provided")
	ErrInvalidSolveDirection      = errors.New(`solve direction must be either "buy" or "sell"`)
	ErrInvalidReverseBuyPrice     = errors.New("buy price must be greater than 0 when solving sell price")
	ErrInvalidReverseSellPrice    = errors.New("sell price must be greater than 0 when solving buy price")
	ErrInvalidReverseTarget       = errors.New("exactly one reverse target is required")
	ErrInvalidTargetKnifeRatio    = errors.New("target knife ratio must be greater than 0")
	ErrSolvedBuyPriceNonPositive  = errors.New("solved buy price must be greater than 0")
	ErrSolvedSellPriceNonPositive = errors.New("solved sell price must be greater than 0")
)

// ValidateReverseRequest validates one reverse-calculation request before solving.
func ValidateReverseRequest(req domain.ReverseCalculationRequest) error {
	if err := validateItemName(req.ItemName); err != nil {
		return err
	}
	if err := validateQuantity(req.Quantity); err != nil {
		return err
	}
	if err := validateMultiplier(req.Multiplier); err != nil {
		return err
	}

	switch req.SolveDirection {
	case "":
		return ErrMissingSolveDirection
	case domain.SolveDirectionSell:
		if req.BuyPrice == nil || *req.BuyPrice <= 0 {
			return ErrInvalidReverseBuyPrice
		}
	case domain.SolveDirectionBuy:
		if req.SellPrice == nil || *req.SellPrice <= 0 {
			return ErrInvalidReverseSellPrice
		}
	default:
		return ErrInvalidSolveDirection
	}

	targets := 0
	if req.TargetNetProfit != nil {
		targets++
	}
	if req.TargetTotalProfit != nil {
		targets++
	}
	if req.TargetKnifeRatio != nil {
		targets++
		if *req.TargetKnifeRatio <= 0 {
			return ErrInvalidTargetKnifeRatio
		}
	}
	if targets != 1 {
		return ErrInvalidReverseTarget
	}

	return nil
}

// CalculateReverse solves either a buy price or a sell price from one target metric.
func CalculateReverse(req domain.ReverseCalculationRequest) (domain.ProfitRecord, error) {
	if err := ValidateReverseRequest(req); err != nil {
		return domain.ProfitRecord{}, err
	}

	itemName := strings.TrimSpace(req.ItemName)
	quantity := effectiveQuantity(req.Quantity)
	multiplier := effectiveMultiplier(req.Multiplier)
	feeLabel := effectiveFeeLabel(req.Multiplier)

	switch req.SolveDirection {
	case domain.SolveDirectionSell:
		buyPrice := *req.BuyPrice
		sellPrice := solveSellPrice(req, buyPrice, quantity, multiplier)
		if sellPrice <= 0 {
			return domain.ProfitRecord{}, ErrSolvedSellPriceNonPositive
		}

		return buildRecord(itemName, buyPrice, sellPrice, quantity, multiplier, feeLabel), nil
	case domain.SolveDirectionBuy:
		sellPrice := *req.SellPrice
		buyPrice := solveBuyPrice(req, sellPrice, quantity, multiplier)
		if buyPrice <= 0 {
			return domain.ProfitRecord{}, ErrSolvedBuyPriceNonPositive
		}

		return buildRecord(itemName, buyPrice, sellPrice, quantity, multiplier, feeLabel), nil
	default:
		return domain.ProfitRecord{}, ErrInvalidSolveDirection
	}
}

func solveSellPrice(req domain.ReverseCalculationRequest, buyPrice float64, quantity int, multiplier float64) float64 {
	if req.TargetKnifeRatio != nil {
		return buyPrice / (*req.TargetKnifeRatio * multiplier)
	}

	return (buyPrice + effectiveTargetNetProfit(req, quantity)) / multiplier
}

func solveBuyPrice(req domain.ReverseCalculationRequest, sellPrice float64, quantity int, multiplier float64) float64 {
	if req.TargetKnifeRatio != nil {
		return *req.TargetKnifeRatio * sellPrice * multiplier
	}

	return sellPrice*multiplier - effectiveTargetNetProfit(req, quantity)
}

func effectiveTargetNetProfit(req domain.ReverseCalculationRequest, quantity int) float64 {
	if req.TargetNetProfit != nil {
		return *req.TargetNetProfit
	}
	if req.TargetTotalProfit != nil {
		return *req.TargetTotalProfit / float64(quantity)
	}

	return 0
}
