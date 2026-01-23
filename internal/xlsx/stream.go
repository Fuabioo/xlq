package xlsx

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// RowResult wraps a row with potential error for channel-based streaming
type RowResult struct {
	Row *Row
	Err error
}

// StreamRows streams rows from startRow to endRow (1-based, inclusive)
// If endRow is 0, streams to end of sheet
// Returns a channel that yields rows and closes when done
func StreamRows(f *excelize.File, sheet string, startRow, endRow int) (<-chan RowResult, error) {
	resolvedSheet, err := ResolveSheetName(f, sheet)
	if err != nil {
		return nil, err
	}

	rows, err := f.Rows(resolvedSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to open row iterator: %w", err)
	}

	ch := make(chan RowResult)

	go func() {
		defer close(ch)
		defer rows.Close()

		rowNum := 0
		for rows.Next() {
			rowNum++

			// Skip rows before startRow
			if startRow > 0 && rowNum < startRow {
				continue
			}

			// Stop after endRow
			if endRow > 0 && rowNum > endRow {
				break
			}

			cols, err := rows.Columns()
			if err != nil {
				ch <- RowResult{Err: fmt.Errorf("error reading row %d: %w", rowNum, err)}
				return
			}

			cells := make([]Cell, len(cols))
			for i, val := range cols {
				cells[i] = Cell{
					Address: FormatCellAddress(i+1, rowNum),
					Value:   val,
					Type:    "string", // Basic type, could enhance later
					Row:     rowNum,
					Col:     i + 1,
				}
			}

			ch <- RowResult{Row: &Row{Number: rowNum, Cells: cells}}
		}

		if err := rows.Error(); err != nil {
			ch <- RowResult{Err: fmt.Errorf("row iteration error: %w", err)}
		}
	}()

	return ch, nil
}

// StreamRange streams cells within a specified range (e.g., "A1:C10")
func StreamRange(f *excelize.File, sheet, rangeStr string) (<-chan RowResult, error) {
	resolvedSheet, err := ResolveSheetName(f, sheet)
	if err != nil {
		return nil, err
	}

	cellRange, err := ParseRange(rangeStr)
	if err != nil {
		return nil, err
	}

	rows, err := f.Rows(resolvedSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to open row iterator: %w", err)
	}

	ch := make(chan RowResult)

	go func() {
		defer close(ch)
		defer rows.Close()

		rowNum := 0
		for rows.Next() {
			rowNum++

			// Skip rows before range
			if rowNum < cellRange.StartRow {
				continue
			}

			// Stop after range
			if rowNum > cellRange.EndRow {
				break
			}

			cols, err := rows.Columns()
			if err != nil {
				ch <- RowResult{Err: fmt.Errorf("error reading row %d: %w", rowNum, err)}
				return
			}

			// Extract only columns in range
			var cells []Cell
			for colIdx := cellRange.StartCol; colIdx <= cellRange.EndCol; colIdx++ {
				val := ""
				if colIdx-1 < len(cols) {
					val = cols[colIdx-1]
				}
				cells = append(cells, Cell{
					Address: FormatCellAddress(colIdx, rowNum),
					Value:   val,
					Type:    "string",
					Row:     rowNum,
					Col:     colIdx,
				})
			}

			ch <- RowResult{Row: &Row{Number: rowNum, Cells: cells}}
		}

		if err := rows.Error(); err != nil {
			ch <- RowResult{Err: fmt.Errorf("row iteration error: %w", err)}
		}
	}()

	return ch, nil
}

// StreamHead streams the first n rows of a sheet
func StreamHead(f *excelize.File, sheet string, n int) (<-chan RowResult, error) {
	if n <= 0 {
		n = 10 // Default to 10 rows
	}
	return StreamRows(f, sheet, 1, n)
}

// StreamTail returns the last n rows of a sheet
// Unlike other streaming functions, this must read the entire sheet
// and uses a ring buffer to keep memory bounded
func StreamTail(f *excelize.File, sheet string, n int) ([]Row, error) {
	if n <= 0 {
		n = 10 // Default to 10 rows
	}

	resolvedSheet, err := ResolveSheetName(f, sheet)
	if err != nil {
		return nil, err
	}

	rows, err := f.Rows(resolvedSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to open row iterator: %w", err)
	}
	defer rows.Close()

	// Ring buffer for last N rows
	buffer := make([]Row, n)
	bufIdx := 0
	totalRows := 0

	rowNum := 0
	for rows.Next() {
		rowNum++

		cols, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("error reading row %d: %w", rowNum, err)
		}

		cells := make([]Cell, len(cols))
		for i, val := range cols {
			cells[i] = Cell{
				Address: FormatCellAddress(i+1, rowNum),
				Value:   val,
				Type:    "string",
				Row:     rowNum,
				Col:     i + 1,
			}
		}

		buffer[bufIdx] = Row{Number: rowNum, Cells: cells}
		bufIdx = (bufIdx + 1) % n
		totalRows++
	}

	if err := rows.Error(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Extract rows from ring buffer in correct order
	if totalRows == 0 {
		return []Row{}, nil
	}

	resultSize := n
	if totalRows < n {
		resultSize = totalRows
	}

	result := make([]Row, resultSize)
	if totalRows < n {
		// Didn't fill the buffer, just copy from start
		copy(result, buffer[:totalRows])
	} else {
		// Buffer is full, read from bufIdx (oldest) to end, then start to bufIdx
		for i := 0; i < n; i++ {
			result[i] = buffer[(bufIdx+i)%n]
		}
	}

	return result, nil
}

// CollectRows collects all rows from a channel into a slice
// Useful for small datasets or when you need all rows in memory
func CollectRows(ch <-chan RowResult) ([]Row, error) {
	var rows []Row
	for result := range ch {
		if result.Err != nil {
			return nil, result.Err
		}
		if result.Row != nil {
			rows = append(rows, *result.Row)
		}
	}
	return rows, nil
}

// RowsToStringSlice converts rows to [][]string for output formatting
func RowsToStringSlice(rows []Row) [][]string {
	result := make([][]string, len(rows))
	for i, row := range rows {
		result[i] = make([]string, len(row.Cells))
		for j, cell := range row.Cells {
			result[i][j] = cell.Value
		}
	}
	return result
}

// StreamRowsToStrings is a convenience function that collects and converts
func StreamRowsToStrings(f *excelize.File, sheet string, startRow, endRow int) ([][]string, error) {
	ch, err := StreamRows(f, sheet, startRow, endRow)
	if err != nil {
		return nil, err
	}
	rows, err := CollectRows(ch)
	if err != nil {
		return nil, err
	}
	return RowsToStringSlice(rows), nil
}
