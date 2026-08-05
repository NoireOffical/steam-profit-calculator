package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"

	"steam-profit-calculator/internal/calculation"
	"steam-profit-calculator/internal/display"
	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/importer"
	"steam-profit-calculator/internal/lookup"
	"steam-profit-calculator/internal/storage"
)

const (
	binaryName        = "steam-profit"
	defaultRecordFile = "steam-profit-records.csv"
)

type floatList []float64

func (f *floatList) String() string {
	return fmt.Sprint([]float64(*f))
}

func (f *floatList) Set(value string) error {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("parse sell price %q: %w", value, err)
	}
	*f = append(*f, parsed)
	return nil
}

type optionalInt struct {
	value int
	set   bool
}

func (o *optionalInt) Set(raw string, field string) error {
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("parse %s %q: %w", field, raw, err)
	}
	o.value = parsed
	o.set = true
	return nil
}

func (o *optionalInt) Ptr() *int {
	if !o.set {
		return nil
	}
	return &o.value
}

type optionalFloat struct {
	value float64
	set   bool
}

func (o *optionalFloat) Set(raw string, field string) error {
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("parse %s %q: %w", field, raw, err)
	}
	o.value = parsed
	o.set = true
	return nil
}

func (o *optionalFloat) Ptr() *float64 {
	if !o.set {
		return nil
	}
	return &o.value
}

// Run executes the CLI entrypoint.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeUsage(stderr)
		return 1
	}

	switch args[0] {
	case "lookup":
		return runLookup(args[1:], stdout, stderr)
	case "add":
		return runAdd(args[1:], stdout, stderr)
	case "reverse":
		return runReverse(args[1:], stdout, stderr)
	case "import":
		return runImport(args[1:], stdout, stderr)
	case "list":
		return runList(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "%s: unknown command %q\n", binaryName, args[0])
		writeUsage(stderr)
		return 1
	}
}

func runLookup(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("lookup", flag.ContinueOnError)
	fs.SetOutput(stderr)

	query := fs.String("query", "", "item query")
	outputMode := fs.String("output", string(display.ModeTable), "display mode: table|csv")
	var limit optionalInt
	fs.Func("limit", "maximum result count", func(value string) error {
		return limit.Set(value, "limit")
	})

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "%s lookup: unexpected arguments: %v\n", binaryName, fs.Args())
		return 2
	}
	if *query == "" {
		_, _ = fmt.Fprintf(stderr, "%s lookup: --query is required\n", binaryName)
		return 2
	}
	if limit.set && limit.value <= 0 {
		_, _ = fmt.Fprintf(stderr, "%s lookup: --limit must be greater than 0\n", binaryName)
		return 2
	}
	mode, err := display.ParseMode(*outputMode)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	client, err := lookup.NewClientFromEnvironment()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "load lookup configuration: %v\n", err)
		return 1
	}

	service := lookup.NewService(client)
	items, err := service.Lookup(context.Background(), domain.ItemLookupRequest{
		Query: *query,
		Limit: limit.Ptr(),
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "lookup items: %v\n", err)
		return 1
	}

	if err := display.WriteLookup(stdout, items, mode); err != nil {
		_, _ = fmt.Fprintf(stderr, "write output: %v\n", err)
		return 1
	}

	return 0
}

func runAdd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(stderr)

	name := fs.String("name", "", "item name")
	buy := fs.Float64("buy", 0, "buy price")
	filePath := fs.String("file", defaultRecordFile, "record file path")
	outputMode := fs.String("output", string(display.ModeTable), "display mode: table|csv")
	var quantity optionalInt
	var multiplier optionalFloat
	var sells floatList
	fs.Func("quantity", "item quantity", func(value string) error {
		return quantity.Set(value, "quantity")
	})
	fs.Func("multiplier", "net multiplier", func(value string) error {
		return multiplier.Set(value, "multiplier")
	})
	fs.Var(&sells, "sell", "sell price (repeatable)")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "%s add: unexpected arguments: %v\n", binaryName, fs.Args())
		return 2
	}
	mode, err := display.ParseMode(*outputMode)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	records, err := calculation.Calculate(domain.CalculationRequest{
		ItemName:   *name,
		BuyPrice:   *buy,
		Quantity:   quantity.Ptr(),
		Multiplier: multiplier.Ptr(),
		SellPrices: []float64(sells),
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	if err := persistAndDisplay(records, *filePath, stdout, mode); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runReverse(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reverse", flag.ContinueOnError)
	fs.SetOutput(stderr)

	name := fs.String("name", "", "item name")
	solve := fs.String("solve", "", "solve direction: buy|sell")
	filePath := fs.String("file", defaultRecordFile, "record file path")
	outputMode := fs.String("output", string(display.ModeTable), "display mode: table|csv")

	var quantity optionalInt
	var multiplier optionalFloat
	var buy optionalFloat
	var sell optionalFloat
	var targetNetProfit optionalFloat
	var targetTotalProfit optionalFloat
	var targetKnifeRatio optionalFloat

	fs.Func("quantity", "item quantity", func(value string) error {
		return quantity.Set(value, "quantity")
	})
	fs.Func("multiplier", "net multiplier", func(value string) error {
		return multiplier.Set(value, "multiplier")
	})
	fs.Func("buy", "known buy price", func(value string) error {
		return buy.Set(value, "buy price")
	})
	fs.Func("sell", "known sell price", func(value string) error {
		return sell.Set(value, "sell price")
	})
	fs.Func("target-net-profit", "target per-item net profit", func(value string) error {
		return targetNetProfit.Set(value, "target net profit")
	})
	fs.Func("target-total-profit", "target total profit", func(value string) error {
		return targetTotalProfit.Set(value, "target total profit")
	})
	fs.Func("target-knife-ratio", "target knife ratio", func(value string) error {
		return targetKnifeRatio.Set(value, "target knife ratio")
	})

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "%s reverse: unexpected arguments: %v\n", binaryName, fs.Args())
		return 2
	}
	mode, err := display.ParseMode(*outputMode)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	record, err := calculation.CalculateReverse(domain.ReverseCalculationRequest{
		ItemName:          *name,
		SolveDirection:    domain.SolveDirection(*solve),
		BuyPrice:          buy.Ptr(),
		SellPrice:         sell.Ptr(),
		Quantity:          quantity.Ptr(),
		Multiplier:        multiplier.Ptr(),
		TargetNetProfit:   targetNetProfit.Ptr(),
		TargetTotalProfit: targetTotalProfit.Ptr(),
		TargetKnifeRatio:  targetKnifeRatio.Ptr(),
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	if err := persistAndDisplay([]domain.ProfitRecord{record}, *filePath, stdout, mode); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}

func runImport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(stderr)

	inputPath := fs.String("input", "", "import file path")
	filePath := fs.String("file", defaultRecordFile, "record file path")
	outputMode := fs.String("output", string(display.ModeTable), "display mode: table|csv")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "%s import: unexpected arguments: %v\n", binaryName, fs.Args())
		return 2
	}
	if *inputPath == "" {
		_, _ = fmt.Fprintf(stderr, "%s import: --input is required\n", binaryName)
		return 2
	}
	mode, err := display.ParseMode(*outputMode)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	records, err := importer.ParseCSV(*inputPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "import records: %v\n", err)
		return 1
	}

	if err := persistAndDisplay(records, *filePath, stdout, mode); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}

func runList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)

	filePath := fs.String("file", defaultRecordFile, "record file path")
	outputMode := fs.String("output", string(display.ModeTable), "display mode: table|csv")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "%s list: unexpected arguments: %v\n", binaryName, fs.Args())
		return 2
	}
	mode, err := display.ParseMode(*outputMode)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	records, err := storage.NewCSVStore(*filePath).List()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "load records: %v\n", err)
		return 1
	}
	if err := display.Write(stdout, records, mode); err != nil {
		_, _ = fmt.Fprintf(stderr, "write output: %v\n", err)
		return 1
	}

	return 0
}

func persistAndDisplay(records []domain.ProfitRecord, filePath string, stdout io.Writer, mode display.Mode) error {
	store := storage.NewCSVStore(filePath)
	if err := store.Append(records); err != nil {
		return fmt.Errorf("save records: %w", err)
	}
	if err := display.Write(stdout, records, mode); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func writeUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, "usage: %s <command> [flags]\n", binaryName)
	_, _ = fmt.Fprintln(w, "commands: lookup, add, reverse, import, list")
}
