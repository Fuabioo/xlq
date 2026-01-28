package xlsx

import (
	"context"
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
// The context can be used to cancel the streaming operation
func StreamRows(ctx context.Context, f *excelize.File, sheet string, startRow, endRow int) (<-chan RowResult, error) {
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
				select {
				case <-ctx.Done():
					return
				case ch <- RowResult{Err: fmt.Errorf("error reading row %d: %w", rowNum, err)}:
					return
				}
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

			select {
			case <-ctx.Done():
				return
			case ch <- RowResult{Row: &Row{Number: rowNum, Cells: cells}}:
			}
		}

		if err := rows.Error(); err != nil {
			select {
			case <-ctx.Done():
				return
			case ch <- RowResult{Err: fmt.Errorf("row iteration error: %w", err)}:
			}
		}
	}()

	return ch, nil
}

// StreamRange streams cells within a specified range (e.g., "A1:C10")
// The context can be used to cancel the streaming operation
func StreamRange(ctx context.Context, f *excelize.File, sheet, rangeStr string) (<-chan RowResult, error) {
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
				select {
				case <-ctx.Done():
					return
				case ch <- RowResult{Err: fmt.Errorf("error reading row %d: %w", rowNum, err)}:
					return
				}
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

			select {
			case <-ctx.Done():
				return
			case ch <- RowResult{Row: &Row{Number: rowNum, Cells: cells}}:
			}
		}

		if err := rows.Error(); err != nil {
			select {
			case <-ctx.Done():
				return
			case ch <- RowResult{Err: fmt.Errorf("row iteration error: %w", err)}:
			}
		}
	}()

	return ch, nil
}

// StreamHead streams the first n rows of a sheet
func StreamHead(ctx context.Context, f *excelize.File, sheet string, n int) (<-chan RowResult, error) {
	if n <= 0 {
		n = 10 // Default to 10 rows
	}
	return StreamRows(ctx, f, sheet, 1, n)
}

// rawRow stores raw column values before Cell construction
// This avoids allocating Cell structs for every row during iteration
type rawRow struct {
	number int
	values []string
}

// StreamTail returns the last n rows of a sheet
// Unlike other streaming functions, this must read the entire sheet
// and uses a ring buffer to keep memory bounded
// Memory optimization: only constructs Cell structs for the final N rows returned
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

	// Ring buffer for last N rows - stores raw values only
	// Pre-allocate the rawRow structs to reuse memory
	buffer := make([]rawRow, n)
	for i := 0; i < n; i++ {
		buffer[i].values = make([]string, 0) // Will grow as needed
	}
	bufIdx := 0
	totalRows := 0

	rowNum := 0
	for rows.Next() {
		rowNum++

		cols, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("error reading row %d: %w", rowNum, err)
		}

		// Reuse the slice in the ring buffer position, but ensure capacity
		// This way we only allocate N slices total, not one per row
		currentSlot := &buffer[bufIdx]

		// Resize the slice if needed
		if cap(currentSlot.values) < len(cols) {
			currentSlot.values = make([]string, len(cols))
		} else {
			currentSlot.values = currentSlot.values[:len(cols)]
		}

		// Copy the values (strings are immutable, so this is cheap)
		copy(currentSlot.values, cols)
		currentSlot.number = rowNum

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
		// Didn't fill the buffer, construct Cells from start
		for i := 0; i < totalRows; i++ {
			result[i] = constructRow(buffer[i])
		}
	} else {
		// Buffer is full, read from bufIdx (oldest) to end, then start to bufIdx
		// Now construct Cell structs ONLY for the N rows we're returning
		for i := 0; i < n; i++ {
			result[i] = constructRow(buffer[(bufIdx+i)%n])
		}
	}

	return result, nil
}

// constructRow builds a Row with Cell structs from raw values
// Only called for rows that will be returned to the caller
func constructRow(raw rawRow) Row {
	cells := make([]Cell, len(raw.values))
	for i, val := range raw.values {
		cells[i] = Cell{
			Address: FormatCellAddress(i+1, raw.number),
			Value:   val,
			Type:    "string",
			Row:     raw.number,
			Col:     i + 1,
		}
	}
	return Row{Number: raw.number, Cells: cells}
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

// CollectRowsWithLimit collects up to limit rows from a channel
// Returns: (rows, totalScanned, truncated, error)
// - rows: collected rows (up to limit)
// - totalScanned: total number of rows seen
// - truncated: true if more rows were available than limit
// - error: any error encountered during collection
func CollectRowsWithLimit(ch <-chan RowResult, limit int) ([]Row, int, bool, error) {
	var rows []Row
	total := 0

	for result := range ch {
		if result.Err != nil {
			return nil, total, false, result.Err
		}
		if result.Row != nil {
			total++
			if len(rows) < limit {
				rows = append(rows, *result.Row)
			}
		}
	}

	truncated := total > limit
	return rows, total, truncated, nil
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
func StreamRowsToStrings(ctx context.Context, f *excelize.File, sheet string, startRow, endRow int) ([][]string, error) {
	ch, err := StreamRows(ctx, f, sheet, startRow, endRow)
	if err != nil {
		return nil, err
	}
	rows, err := CollectRows(ch)
	if err != nil {
		return nil, err
	}
	return RowsToStringSlice(rows), nil
}
