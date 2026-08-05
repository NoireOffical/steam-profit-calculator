package calculation_test

import (
	"errors"
	"strings"
	"testing"

	"steam-profit-calculator/internal/calculation"
	"steam-profit-calculator/internal/domain"
)

func TestCalculateSuccess(t *testing.T) {
	req := domain.CalculationRequest{
		ItemName:   "  AK-47 Redline  ",
		BuyPrice:   100,
		Quantity:   quantityPtr(3),
		SellPrices: []float64{120, 135},
	}

	records, err := calculation.Calculate(req)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}

	gotFirst := strings.Join(records[0].CSVRow(), ",")
        wantFirst := "AK-47 Redline,100.000,120.000,3,0.440,1.320,0.995619,default"
	if gotFirst != wantFirst {
		t.Fatalf("first record = %q, want %q", gotFirst, wantFirst)
	}

	gotSecond := strings.Join(records[1].CSVRow(), ",")
        wantSecond := "AK-47 Redline,100.000,135.000,3,12.995,38.985,0.884995,default"
	if gotSecond != wantSecond {
		t.Fatalf("second record = %q, want %q", gotSecond, wantSecond)
	}
}

func TestCalculateDefaultsQuantityToOne(t *testing.T) {
	req := domain.CalculationRequest{
		ItemName:   "Capsule",
		BuyPrice:   0.48,
		SellPrices: []float64{0.8},
	}

	records, err := calculation.Calculate(req)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	got := strings.Join(records[0].CSVRow(), ",")
        want := "Capsule,0.480,0.800,1,0.190,0.190,0.716846,default"
	if got != want {
		t.Fatalf("default-quantity record = %q, want %q", got, want)
	}
}

func TestCalculateAllowsNegativeProfit(t *testing.T) {
	req := domain.CalculationRequest{
		ItemName:   "AK-47 Redline",
		BuyPrice:   100,
		Quantity:   quantityPtr(2),
		SellPrices: []float64{110},
	}

	records, err := calculation.Calculate(req)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	got := strings.Join(records[0].CSVRow(), ",")
        want := "AK-47 Redline,100.000,110.000,2,-7.930,-15.860,1.086130,default"
	if got != want {
		t.Fatalf("negative-profit record = %q, want %q", got, want)
	}
}

func TestCalculateSupportsExplicitMultiplier(t *testing.T) {
	req := domain.CalculationRequest{
		ItemName:   "Capsule",
		BuyPrice:   0.48,
		Quantity:   quantityPtr(2),
		Multiplier: float64Ptr(0.85),
		SellPrices: []float64{0.8},
	}

	records, err := calculation.Calculate(req)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	got := strings.Join(records[0].CSVRow(), ",")
        want := "Capsule,0.480,0.800,2,0.200,0.400,0.705882,0.850"
	if got != want {
		t.Fatalf("multiplier record = %q, want %q", got, want)
	}
}

func TestValidateRequestFailures(t *testing.T) {
	tests := []struct {
		name string
		req  domain.CalculationRequest
		want error
		msg  string
	}{
		{
			name: "empty item name",
			req: domain.CalculationRequest{
				ItemName:   "   ",
				BuyPrice:   100,
				Quantity:   quantityPtr(1),
				SellPrices: []float64{120},
			},
			want: calculation.ErrEmptyItemName,
		},
		{
			name: "invalid buy price",
			req: domain.CalculationRequest{
				ItemName:   "AK-47 Redline",
				BuyPrice:   0,
				Quantity:   quantityPtr(1),
				SellPrices: []float64{120},
			},
			want: calculation.ErrInvalidBuy,
		},
		{
			name: "invalid quantity",
			req: domain.CalculationRequest{
				ItemName:   "AK-47 Redline",
				BuyPrice:   100,
				Quantity:   quantityPtr(0),
				SellPrices: []float64{120},
			},
			want: calculation.ErrInvalidQuantity,
		},
		{
			name: "invalid multiplier",
			req: domain.CalculationRequest{
				ItemName:   "AK-47 Redline",
				BuyPrice:   100,
				Quantity:   quantityPtr(1),
				Multiplier: float64Ptr(1.2),
				SellPrices: []float64{120},
			},
			want: calculation.ErrInvalidMultiplier,
		},
		{
			name: "missing sell prices",
			req: domain.CalculationRequest{
				ItemName: "AK-47 Redline",
				BuyPrice: 100,
				Quantity: quantityPtr(1),
			},
			want: calculation.ErrMissingSells,
		},
		{
			name: "invalid sell price",
			req: domain.CalculationRequest{
				ItemName:   "AK-47 Redline",
				BuyPrice:   100,
				Quantity:   quantityPtr(1),
				SellPrices: []float64{120, 0},
			},
			msg: "sell price at index 1 must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := calculation.ValidateRequest(tt.req)
			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("ValidateRequest() error = %v, want %v", err, tt.want)
				}
				return
			}

			if err == nil {
				t.Fatal("ValidateRequest() error = nil, want non-nil")
			}
			if err.Error() != tt.msg {
				t.Fatalf("ValidateRequest() error = %q, want %q", err.Error(), tt.msg)
			}
		})
	}
}

func TestCalculateReverseSuccess(t *testing.T) {
	tests := []struct {
		name string
		req  domain.ReverseCalculationRequest
		want string
	}{
		{
			name: "solve sell price from target total profit",
			req: domain.ReverseCalculationRequest{
				ItemName:          "Capsule",
				SolveDirection:    domain.SolveDirectionSell,
				BuyPrice:          float64Ptr(0.48),
				Quantity:          quantityPtr(5),
				Multiplier:        float64Ptr(0.85),
				TargetTotalProfit: float64Ptr(1.0),
			},
                        want: "Capsule,0.480,0.800,5,0.200,1.000,0.705882,0.850",
		},
		{
			name: "solve sell price from target knife ratio",
			req: domain.ReverseCalculationRequest{
				ItemName:         "Capsule",
				SolveDirection:   domain.SolveDirectionSell,
				BuyPrice:         float64Ptr(0.48),
				TargetKnifeRatio: float64Ptr(0.7),
			},
                        want: "Capsule,0.480,0.819,1,0.206,0.206,0.700000,default",
		},
		{
			name: "solve buy price from target knife ratio",
			req: domain.ReverseCalculationRequest{
				ItemName:         "Capsule",
				SolveDirection:   domain.SolveDirectionBuy,
				SellPrice:        float64Ptr(0.8),
				TargetKnifeRatio: float64Ptr(0.7),
			},
                        want: "Capsule,0.469,0.800,1,0.201,0.201,0.700000,default",
		},
		{
			name: "solve buy price from target net profit",
			req: domain.ReverseCalculationRequest{
				ItemName:        "Capsule",
				SolveDirection:  domain.SolveDirectionBuy,
				SellPrice:       float64Ptr(0.8),
				Quantity:        quantityPtr(2),
				Multiplier:      float64Ptr(0.85),
				TargetNetProfit: float64Ptr(0.15),
			},
                        want: "Capsule,0.530,0.800,2,0.150,0.300,0.779412,0.850",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := calculation.CalculateReverse(tt.req)
			if err != nil {
				t.Fatalf("CalculateReverse() error = %v", err)
			}

			got := strings.Join(record.CSVRow(), ",")
			if got != tt.want {
				t.Fatalf("record = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateReverseRequestFailures(t *testing.T) {
	tests := []struct {
		name string
		req  domain.ReverseCalculationRequest
		want error
	}{
		{
			name: "missing solve direction",
			req: domain.ReverseCalculationRequest{
				ItemName:        "Capsule",
				BuyPrice:        float64Ptr(0.48),
				TargetNetProfit: float64Ptr(0.19),
			},
			want: calculation.ErrMissingSolveDirection,
		},
		{
			name: "invalid solve direction",
			req: domain.ReverseCalculationRequest{
				ItemName:        "Capsule",
				SolveDirection:  domain.SolveDirection("other"),
				SellPrice:       float64Ptr(0.8),
				TargetNetProfit: float64Ptr(0.19),
			},
			want: calculation.ErrInvalidSolveDirection,
		},
		{
			name: "missing buy price for solve sell",
			req: domain.ReverseCalculationRequest{
				ItemName:        "Capsule",
				SolveDirection:  domain.SolveDirectionSell,
				TargetNetProfit: float64Ptr(0.19),
			},
			want: calculation.ErrInvalidReverseBuyPrice,
		},
		{
			name: "conflicting targets",
			req: domain.ReverseCalculationRequest{
				ItemName:         "Capsule",
				SolveDirection:   domain.SolveDirectionBuy,
				SellPrice:        float64Ptr(0.8),
				TargetNetProfit:  float64Ptr(0.19),
				TargetKnifeRatio: float64Ptr(0.7),
			},
			want: calculation.ErrInvalidReverseTarget,
		},
		{
			name: "invalid target knife ratio",
			req: domain.ReverseCalculationRequest{
				ItemName:         "Capsule",
				SolveDirection:   domain.SolveDirectionBuy,
				SellPrice:        float64Ptr(0.8),
				TargetKnifeRatio: float64Ptr(0),
			},
			want: calculation.ErrInvalidTargetKnifeRatio,
		},
		{
			name: "invalid multiplier",
			req: domain.ReverseCalculationRequest{
				ItemName:        "Capsule",
				SolveDirection:  domain.SolveDirectionBuy,
				SellPrice:       float64Ptr(0.8),
				Multiplier:      float64Ptr(1.2),
				TargetNetProfit: float64Ptr(0.19),
			},
			want: calculation.ErrInvalidMultiplier,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := calculation.ValidateReverseRequest(tt.req)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ValidateReverseRequest() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func quantityPtr(value int) *int {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
