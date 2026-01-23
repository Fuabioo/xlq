package xlsx

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Error types
var (
	ErrInvalidRange   = errors.New("invalid cell range")
	ErrInvalidAddress = errors.New("invalid cell address")
	ErrSheetNotFound  = errors.New("sheet not found")
	ErrFileNotFound   = errors.New("file not found")
	ErrInvalidFormat  = errors.New("invalid xlsx format")
)

// CellRange represents a rectangular range of cells (e.g., A1:C10)
type CellRange struct {
	StartCol int // 1-based column (A=1)
	StartRow int // 1-based row
	EndCol   int
	EndRow   int
}

// SheetInfo contains metadata about a worksheet
type SheetInfo struct {
	Name    string   `json:"name"`
	Rows    int      `json:"rows"`
	Cols    int      `json:"cols"`
	Headers []string `json:"headers,omitempty"`
}

// Cell represents a single cell with its value and metadata
type Cell struct {
	Address string `json:"address"`
	Value   string `json:"value"`
	Type    string `json:"type"` // string, number, bool, formula, error, empty
	Row     int    `json:"row"`
	Col     int    `json:"col"`
}

// Row represents a row of cells
type Row struct {
	Number int    `json:"row"`
	Cells  []Cell `json:"cells"`
}

// SearchResult represents a cell that matched a search pattern
type SearchResult struct {
	Sheet   string `json:"sheet"`
	Address string `json:"address"`
	Value   string `json:"value"`
	Row     int    `json:"row"`
	Col     int    `json:"col"`
}

// cellAddrRegex matches cell addresses like A1, B23, AA100
var cellAddrRegex = regexp.MustCompile(`^([A-Za-z]+)([0-9]+)$`)

// ParseCellAddress parses a cell address like "A1" into column and row numbers
// Returns 1-based column and row numbers
func ParseCellAddress(addr string) (col, row int, err error) {
	addr = strings.TrimSpace(strings.ToUpper(addr))
	matches := cellAddrRegex.FindStringSubmatch(addr)
	if matches == nil {
		return 0, 0, fmt.Errorf("%w: %s", ErrInvalidAddress, addr)
	}

	col = ColumnNameToNumber(matches[1])
	row, err = strconv.Atoi(matches[2])
	if err != nil || row < 1 {
		return 0, 0, fmt.Errorf("%w: %s", ErrInvalidAddress, addr)
	}

	return col, row, nil
}

// ColumnNameToNumber converts a column name (A, B, ..., Z, AA, AB, ...) to a 1-based number
func ColumnNameToNumber(name string) int {
	name = strings.ToUpper(name)
	result := 0
	for _, ch := range name {
		result = result*26 + int(ch-'A'+1)
	}
	return result
}

// ColumnNumberToName converts a 1-based column number to a column name
func ColumnNumberToName(col int) string {
	name := ""
	for col > 0 {
		col-- // Adjust for 1-based
		name = string(rune('A'+col%26)) + name
		col /= 26
	}
	return name
}

// FormatCellAddress formats a column and row number into an address like "A1"
func FormatCellAddress(col, row int) string {
	return fmt.Sprintf("%s%d", ColumnNumberToName(col), row)
}

// ParseRange parses a range string like "A1:C10" or "A1" into a CellRange
func ParseRange(rangeStr string) (*CellRange, error) {
	rangeStr = strings.TrimSpace(strings.ToUpper(rangeStr))

	parts := strings.Split(rangeStr, ":")
	switch len(parts) {
	case 1:
		// Single cell: "A1"
		col, row, err := ParseCellAddress(parts[0])
		if err != nil {
			return nil, err
		}
		return &CellRange{
			StartCol: col,
			StartRow: row,
			EndCol:   col,
			EndRow:   row,
		}, nil

	case 2:
		// Range: "A1:C10"
		startCol, startRow, err := ParseCellAddress(parts[0])
		if err != nil {
			return nil, fmt.Errorf("%w: invalid start %s", ErrInvalidRange, parts[0])
		}
		endCol, endRow, err := ParseCellAddress(parts[1])
		if err != nil {
			return nil, fmt.Errorf("%w: invalid end %s", ErrInvalidRange, parts[1])
		}

		// Normalize: ensure start <= end
		if startCol > endCol {
			startCol, endCol = endCol, startCol
		}
		if startRow > endRow {
			startRow, endRow = endRow, startRow
		}

		return &CellRange{
			StartCol: startCol,
			StartRow: startRow,
			EndCol:   endCol,
			EndRow:   endRow,
		}, nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidRange, rangeStr)
	}
}

// Contains checks if a cell address is within this range
func (r *CellRange) Contains(col, row int) bool {
	return col >= r.StartCol && col <= r.EndCol &&
		row >= r.StartRow && row <= r.EndRow
}

// String returns the range as a string like "A1:C10"
func (r *CellRange) String() string {
	if r.StartCol == r.EndCol && r.StartRow == r.EndRow {
		return FormatCellAddress(r.StartCol, r.StartRow)
	}
	return fmt.Sprintf("%s:%s",
		FormatCellAddress(r.StartCol, r.StartRow),
		FormatCellAddress(r.EndCol, r.EndRow))
}

// IsValidRange checks if a string looks like a valid cell range
func IsValidRange(s string) bool {
	_, err := ParseRange(s)
	return err == nil
}
