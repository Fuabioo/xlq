package xlsx

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// OpenFile opens an xlsx file and returns the excelize handle
func OpenFile(path string) (*excelize.File, error) {
	// Check file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open xlsx file %s: %w", path, err)
	}

	return f, nil
}

// GetSheets returns a list of all sheet names in the workbook
func GetSheets(f *excelize.File) ([]string, error) {
	if f == nil {
		return nil, fmt.Errorf("file handle is nil")
	}

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in workbook")
	}

	return sheets, nil
}

// GetSheetInfo returns metadata about a sheet using streaming to count rows
func GetSheetInfo(f *excelize.File, sheet string) (*SheetInfo, error) {
	if f == nil {
		return nil, fmt.Errorf("file handle is nil")
	}

	// Verify sheet exists
	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if strings.EqualFold(s, sheet) {
			sheet = s // Use actual casing
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("%w: %s", ErrSheetNotFound, sheet)
	}

	// Use streaming API to count rows without loading all data
	rows, err := f.Rows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows from sheet %s: %w", sheet, err)
	}
	defer rows.Close()

	info := &SheetInfo{
		Name: sheet,
		Rows: 0,
		Cols: 0,
	}

	rowNum := 0
	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("failed to read columns at row %d: %w", rowNum, err)
		}

		// Track max columns
		if len(cols) > info.Cols {
			info.Cols = len(cols)
		}

		// First row = headers
		if rowNum == 1 && len(cols) > 0 {
			info.Headers = make([]string, len(cols))
			copy(info.Headers, cols)
		}
	}

	if err := rows.Error(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	info.Rows = rowNum
	return info, nil
}

// GetCell retrieves a single cell value
func GetCell(f *excelize.File, sheet, addr string) (*Cell, error) {
	if f == nil {
		return nil, fmt.Errorf("file handle is nil")
	}

	// Verify sheet exists
	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if strings.EqualFold(s, sheet) {
			sheet = s
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("%w: %s", ErrSheetNotFound, sheet)
	}

	// Parse and validate address
	col, row, err := ParseCellAddress(addr)
	if err != nil {
		return nil, err
	}

	// Get cell value
	value, err := f.GetCellValue(sheet, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get cell %s: %w", addr, err)
	}

	// Get cell type
	cellType := detectCellType(f, sheet, addr, value)

	return &Cell{
		Address: strings.ToUpper(addr),
		Value:   value,
		Type:    cellType,
		Row:     row,
		Col:     col,
	}, nil
}

// GetDefaultSheet returns the first sheet name or error if none exist
func GetDefaultSheet(f *excelize.File) (string, error) {
	sheets, err := GetSheets(f)
	if err != nil {
		return "", err
	}
	return sheets[0], nil
}

// detectCellType determines the type of a cell
func detectCellType(f *excelize.File, sheet, addr, value string) string {
	if value == "" {
		return "empty"
	}

	// Check for formula
	formula, _ := f.GetCellFormula(sheet, addr)
	if formula != "" {
		return "formula"
	}

	// Check cell type from excelize
	cellType, err := f.GetCellType(sheet, addr)
	if err != nil {
		// Fallback to value-based detection
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return "number"
		}
		if value == "true" || value == "false" {
			return "bool"
		}
		return "string"
	}

	switch cellType {
	case excelize.CellTypeNumber, excelize.CellTypeDate:
		return "number"
	case excelize.CellTypeBool:
		return "bool"
	case excelize.CellTypeFormula:
		return "formula"
	case excelize.CellTypeError:
		return "error"
	case excelize.CellTypeInlineString, excelize.CellTypeSharedString:
		return "string"
	default:
		// Fallback: try to detect by value content
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return "number"
		}
		if value == "true" || value == "false" {
			return "bool"
		}
		return "string"
	}
}

// SheetExists checks if a sheet exists in the workbook
func SheetExists(f *excelize.File, sheet string) bool {
	sheets := f.GetSheetList()
	for _, s := range sheets {
		if strings.EqualFold(s, sheet) {
			return true
		}
	}
	return false
}

// ResolveSheetName returns the actual sheet name (with correct casing) or default
func ResolveSheetName(f *excelize.File, sheet string) (string, error) {
	if sheet == "" {
		return GetDefaultSheet(f)
	}

	sheets := f.GetSheetList()
	for _, s := range sheets {
		if strings.EqualFold(s, sheet) {
			return s, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrSheetNotFound, sheet)
}
