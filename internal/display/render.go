package display

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"steam-profit-calculator/internal/domain"
)

type Mode string

const (
	ModeTable Mode = "table"
	ModeCSV   Mode = "csv"
)

func ParseMode(raw string) (Mode, error) {
	switch Mode(strings.ToLower(raw)) {
	case "", ModeTable:
		return ModeTable, nil
	case ModeCSV:
		return ModeCSV, nil
	default:
		return "", fmt.Errorf("unsupported output mode %q", raw)
	}
}

func Write(w io.Writer, records []domain.ProfitRecord, mode Mode) error {
	switch mode {
	case ModeTable:
		header, rows := fieldRows(records)
		return writeTableRows(w, header, rows)
	case ModeCSV:
		header, rows := fieldRows(records)
		return writeCSVRows(w, header, rows)
	default:
		return fmt.Errorf("unsupported output mode %q", mode)
	}
}

func WriteLookup(w io.Writer, items []domain.ItemSummary, mode Mode) error {
	header, rows := lookupFieldRows(items)

	switch mode {
	case ModeTable:
		return writeTableRows(w, header, rows)
	case ModeCSV:
		return writeCSVRows(w, header, rows)
	default:
		return fmt.Errorf("unsupported output mode %q", mode)
	}
}

func fieldRows(records []domain.ProfitRecord) (header []string, rows [][]string) {
	header = domain.RecordHeader()
	rows = make([][]string, 0, len(records))
	for _, record := range records {
		rows = append(rows, record.Fields())
	}
	return header, rows
}

func lookupFieldRows(items []domain.ItemSummary) (header []string, rows [][]string) {
	header = domain.ItemSummaryHeader()
	rows = make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, item.Fields())
	}
	return header, rows
}

func writeCSVRows(w io.Writer, header []string, rows [][]string) error {
	writer := csv.NewWriter(w)

	if err := writer.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeTableRows(w io.Writer, header []string, rows [][]string) error {
	widths := columnWidths(header, rows)

	if _, err := fmt.Fprintln(w, formatRow(header, widths, false)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, separatorRow(widths)); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(w, formatRow(row, widths, true)); err != nil {
			return err
		}
	}
	return nil
}

func columnWidths(header []string, rows [][]string) []int {
	widths := make([]int, len(header))
	for i, value := range header {
		widths[i] = len(value)
	}
	for _, row := range rows {
		for i, value := range row {
			if len(value) > widths[i] {
				widths[i] = len(value)
			}
		}
	}
	return widths
}

func formatRow(values []string, widths []int, alignNumbers bool) string {
	cells := make([]string, len(values))
	for i, value := range values {
		if alignNumbers && i > 0 {
			cells[i] = fmt.Sprintf("%*s", widths[i], value)
			continue
		}
		cells[i] = fmt.Sprintf("%-*s", widths[i], value)
	}
	return strings.TrimRight(strings.Join(cells, "  "), " ")
}

func separatorRow(widths []int) string {
	cells := make([]string, len(widths))
	for i, width := range widths {
		cells[i] = strings.Repeat("-", width)
	}
	return strings.Join(cells, "  ")
}
