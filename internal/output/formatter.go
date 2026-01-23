package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Format represents output format options
type Format string

const (
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
	FormatTSV  Format = "tsv"
)

// Formatter interface for outputting data in various formats
type Formatter interface {
	// FormatValue formats a single value (for streaming)
	FormatValue(v interface{}) ([]byte, error)

	// FormatSlice formats a slice of values
	FormatSlice(v interface{}) ([]byte, error)

	// WriteHeader writes any format header (e.g., opening bracket for JSON array)
	WriteHeader(w io.Writer) error

	// WriteFooter writes any format footer (e.g., closing bracket for JSON array)
	WriteFooter(w io.Writer) error

	// WriteSeparator writes separator between items (e.g., comma for JSON)
	WriteSeparator(w io.Writer) error
}

// NewFormatter creates a formatter for the specified format
func NewFormatter(format string) (Formatter, error) {
	switch Format(strings.ToLower(format)) {
	case FormatJSON, "":
		return &JSONFormatter{}, nil
	case FormatCSV:
		return &CSVFormatter{}, nil
	case FormatTSV:
		return &TSVFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s (valid: json, csv, tsv)", format)
	}
}

// JSONFormatter outputs JSON format
type JSONFormatter struct {
	itemCount int
}

func (f *JSONFormatter) FormatValue(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON value: %w", err)
	}
	return data, nil
}

func (f *JSONFormatter) FormatSlice(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON slice: %w", err)
	}
	return data, nil
}

func (f *JSONFormatter) WriteHeader(w io.Writer) error {
	_, err := w.Write([]byte("["))
	if err != nil {
		return fmt.Errorf("failed to write JSON header: %w", err)
	}
	return nil
}

func (f *JSONFormatter) WriteFooter(w io.Writer) error {
	_, err := w.Write([]byte("]\n"))
	if err != nil {
		return fmt.Errorf("failed to write JSON footer: %w", err)
	}
	return nil
}

func (f *JSONFormatter) WriteSeparator(w io.Writer) error {
	f.itemCount++
	if f.itemCount > 1 {
		_, err := w.Write([]byte(","))
		if err != nil {
			return fmt.Errorf("failed to write JSON separator: %w", err)
		}
	}
	return nil
}

// CSVFormatter outputs CSV format
type CSVFormatter struct{}

func (f *CSVFormatter) FormatValue(v interface{}) ([]byte, error) {
	row, err := toStringSlice(v)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value to string slice: %w", err)
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	if err := w.Write(row); err != nil {
		return nil, fmt.Errorf("failed to write CSV row: %w", err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}
	return []byte(buf.String()), nil
}

func (f *CSVFormatter) FormatSlice(v interface{}) ([]byte, error) {
	rows, err := toStringSliceSlice(v)
	if err != nil {
		return nil, fmt.Errorf("failed to convert slice to string slice slice: %w", err)
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	for i, row := range rows {
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row %d: %w", i, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}
	return []byte(buf.String()), nil
}

func (f *CSVFormatter) WriteHeader(w io.Writer) error {
	return nil // CSV has no header wrapper
}

func (f *CSVFormatter) WriteFooter(w io.Writer) error {
	return nil // CSV has no footer wrapper
}

func (f *CSVFormatter) WriteSeparator(w io.Writer) error {
	return nil // Rows already include newlines
}

// TSVFormatter outputs tab-separated format
type TSVFormatter struct{}

func (f *TSVFormatter) FormatValue(v interface{}) ([]byte, error) {
	row, err := toStringSlice(v)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value to string slice: %w", err)
	}
	return []byte(strings.Join(row, "\t") + "\n"), nil
}

func (f *TSVFormatter) FormatSlice(v interface{}) ([]byte, error) {
	rows, err := toStringSliceSlice(v)
	if err != nil {
		return nil, fmt.Errorf("failed to convert slice to string slice slice: %w", err)
	}

	var lines []string
	for _, row := range rows {
		lines = append(lines, strings.Join(row, "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func (f *TSVFormatter) WriteHeader(w io.Writer) error {
	return nil
}

func (f *TSVFormatter) WriteFooter(w io.Writer) error {
	return nil
}

func (f *TSVFormatter) WriteSeparator(w io.Writer) error {
	return nil
}

// toStringSlice converts various types to []string for CSV/TSV output
func toStringSlice(v interface{}) ([]string, error) {
	switch val := v.(type) {
	case []string:
		return val, nil
	case []interface{}:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result, nil
	case map[string]interface{}:
		// For JSON objects, output values in a consistent order
		result := make([]string, 0, len(val))
		for _, v := range val {
			result = append(result, fmt.Sprintf("%v", v))
		}
		return result, nil
	default:
		return []string{fmt.Sprintf("%v", v)}, nil
	}
}

// toStringSliceSlice converts to [][]string for multi-row output
func toStringSliceSlice(v interface{}) ([][]string, error) {
	switch val := v.(type) {
	case [][]string:
		return val, nil
	case []interface{}:
		result := make([][]string, len(val))
		for i, row := range val {
			var err error
			result[i], err = toStringSlice(row)
			if err != nil {
				return nil, fmt.Errorf("failed to convert row %d: %w", i, err)
			}
		}
		return result, nil
	default:
		row, err := toStringSlice(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value to row: %w", err)
		}
		return [][]string{row}, nil
	}
}

// FormatRows is a convenience function for formatting row data
func FormatRows(format string, rows [][]string) ([]byte, error) {
	f, err := NewFormatter(format)
	if err != nil {
		return nil, fmt.Errorf("failed to create formatter: %w", err)
	}

	data, err := f.FormatSlice(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to format rows: %w", err)
	}
	return data, nil
}

// FormatSingle is a convenience function for formatting a single object
func FormatSingle(format string, v interface{}) ([]byte, error) {
	f, err := NewFormatter(format)
	if err != nil {
		return nil, fmt.Errorf("failed to create formatter: %w", err)
	}

	if format == "" || Format(format) == FormatJSON {
		// For JSON, format as single object, not array
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return append(data, '\n'), nil
	}

	data, err := f.FormatValue(v)
	if err != nil {
		return nil, fmt.Errorf("failed to format value: %w", err)
	}
	return data, nil
}
