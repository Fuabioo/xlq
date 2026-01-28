package xlsx

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// OpenFileForWrite opens an existing xlsx file for write operations.
// It validates the file exists and is within size limits.
func OpenFileForWrite(path string) (*excelize.File, error) {
	// Check file exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return nil, fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	// Check file size against MaxWriteFileSize
	if fileInfo.Size() > MaxWriteFileSize {
		return nil, fmt.Errorf("%w: file size %d bytes exceeds limit of %d bytes",
			ErrFileTooLarge, fileInfo.Size(), MaxWriteFileSize)
	}

	// Open with excelize
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s for write: %w", path, err)
	}

	return f, nil
}

// SaveFile saves the xlsx file to disk.
func SaveFile(f *excelize.File, path string) error {
	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("failed to save file %s: %w", path, err)
	}
	return nil
}

// SaveFileAtomic saves the file atomically using temp file + rename.
// This prevents corruption if the process is interrupted.
func SaveFileAtomic(f *excelize.File, path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create temp file in same directory as target
	base := filepath.Base(path)
	tmpPath := filepath.Join(dir, base+".tmp")

	// Write to temp file using excelize's WriteTo for better control
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file %s: %w", tmpPath, err)
	}

	// Write the file content
	if err := f.Write(tmpFile); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write to temp file %s: %w", tmpPath, err)
	}

	// Close temp file
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file %s: %w", tmpPath, err)
	}

	// Rename temp to target (atomic on most filesystems)
	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file to %s: %w", path, err)
	}

	return nil
}

// setCellWithType writes a value to a cell with appropriate type handling.
// valueType can be: "auto", "string", "number", "bool", "formula"
// "auto" detects type from Go value
func setCellWithType(f *excelize.File, sheet, cell string, value any, valueType string) error {
	// Determine actual type to use
	actualType := valueType
	if valueType == "auto" {
		actualType = detectValueType(value)
	}

	// Write based on type
	switch actualType {
	case "string":
		val := fmt.Sprintf("%v", value)
		if err := f.SetCellStr(sheet, cell, val); err != nil {
			return fmt.Errorf("failed to set cell %s as string: %w", cell, err)
		}

	case "number":
		var num float64
		switch v := value.(type) {
		case float64:
			num = v
		case float32:
			num = float64(v)
		case int:
			num = float64(v)
		case int64:
			num = float64(v)
		case int32:
			num = float64(v)
		case string:
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("failed to parse string %q as number: %w", v, err)
			}
			num = parsed
		default:
			return fmt.Errorf("cannot convert %T to number", value)
		}
		if err := f.SetCellFloat(sheet, cell, num, -1, 64); err != nil {
			return fmt.Errorf("failed to set cell %s as number: %w", cell, err)
		}

	case "bool":
		var b bool
		switch v := value.(type) {
		case bool:
			b = v
		case string:
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("failed to parse string %q as bool: %w", v, err)
			}
			b = parsed
		default:
			return fmt.Errorf("cannot convert %T to bool", value)
		}
		if err := f.SetCellBool(sheet, cell, b); err != nil {
			return fmt.Errorf("failed to set cell %s as bool: %w", cell, err)
		}

	case "formula":
		formula, ok := value.(string)
		if !ok {
			return fmt.Errorf("formula must be string, got %T", value)
		}
		// Ensure formula starts with =
		if !strings.HasPrefix(formula, "=") {
			formula = "=" + formula
		}
		if err := f.SetCellFormula(sheet, cell, formula); err != nil {
			return fmt.Errorf("failed to set cell %s as formula: %w", cell, err)
		}

	default:
		return fmt.Errorf("unknown value type: %s", actualType)
	}

	return nil
}

// detectValueType infers the value type from a Go value
func detectValueType(value any) string {
	if value == nil {
		return "string"
	}

	switch v := value.(type) {
	case bool:
		return "bool"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "number"
	case float32, float64:
		return "number"
	case string:
		// Check if it looks like a formula
		if strings.HasPrefix(v, "=") {
			return "formula"
		}
		// Check if it's a parseable number
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			return "number"
		}
		// Check if it's a bool
		if _, err := strconv.ParseBool(v); err == nil {
			return "bool"
		}
		return "string"
	default:
		return "string"
	}
}

// getLastRow returns the last row number with data in the sheet.
// Uses streaming to avoid loading entire sheet.
func getLastRow(f *excelize.File, sheet string) (int, error) {
	rows, err := f.Rows(sheet)
	if err != nil {
		return 0, fmt.Errorf("failed to get rows for sheet %s: %w", sheet, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close rows: %w", closeErr)
		}
	}()

	lastRow := 0
	for rows.Next() {
		lastRow++
	}

	if err := rows.Error(); err != nil {
		return 0, fmt.Errorf("error while streaming rows: %w", err)
	}

	return lastRow, err
}

// WriteCell writes a value to a specific cell in an xlsx file.
// It opens the file, writes the cell, and saves atomically.
// Returns the previous value for confirmation.
func WriteCell(path, sheet, cell string, value any, valueType string) (*WriteResult, error) {
	// 1. Open file for write
	f, err := OpenFileForWrite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for write: %w", err)
	}
	defer f.Close()

	// 2. Resolve sheet name (use empty string for default sheet)
	resolvedSheet, err := ResolveSheetName(f, sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sheet name: %w", err)
	}

	// 3. Get previous value
	previousValue, err := f.GetCellValue(resolvedSheet, cell)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous cell value: %w", err)
	}

	// 4. Use setCellWithType to write new value
	if err := setCellWithType(f, resolvedSheet, cell, value, valueType); err != nil {
		return nil, fmt.Errorf("failed to write cell: %w", err)
	}

	// 5. Save atomically
	if err := SaveFileAtomic(f, path); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 6. Return WriteResult
	return &WriteResult{
		Success:       true,
		Cell:          cell,
		PreviousValue: previousValue,
		NewValue:      value,
	}, nil
}

// AppendRows appends rows to the end of a sheet.
// It finds the last row and writes new data starting at lastRow+1.
// Enforces MaxAppendRows limit.
func AppendRows(path, sheet string, rows [][]any) (*AppendResult, error) {
	// 1. Validate row count
	if len(rows) > MaxAppendRows {
		return nil, fmt.Errorf("%w: attempting to append %d rows, limit is %d",
			ErrRowLimitExceeded, len(rows), MaxAppendRows)
	}

	// 2. Open file for write
	f, err := OpenFileForWrite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for write: %w", err)
	}
	defer f.Close()

	// 3. Resolve sheet name
	resolvedSheet, err := ResolveSheetName(f, sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sheet name: %w", err)
	}

	// 4. Use getLastRow to find last row
	lastRow, err := getLastRow(f, resolvedSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to get last row: %w", err)
	}

	// 5. Write each row using f.SetSheetRow()
	startingRow := lastRow + 1
	for i, row := range rows {
		rowNum := startingRow + i

		// Convert []any to []any for SetSheetRow
		cells := make([]any, len(row))
		copy(cells, row)

		// Use column A (1-based) as the starting cell
		cellAddr := FormatCellAddress(1, rowNum)
		if err := f.SetSheetRow(resolvedSheet, cellAddr, &cells); err != nil {
			return nil, fmt.Errorf("failed to write row %d: %w", rowNum, err)
		}
	}

	// 6. Save atomically
	if err := SaveFileAtomic(f, path); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 7. Return AppendResult
	endingRow := startingRow + len(rows) - 1
	return &AppendResult{
		Success:     true,
		RowsAdded:   len(rows),
		StartingRow: startingRow,
		EndingRow:   endingRow,
	}, nil
}

// CreateFile creates a new xlsx file with optional initial data.
// Uses StreamWriter for efficiency when writing many rows.
func CreateFile(path, sheetName string, headers []string, rows [][]any, overwrite bool) (*CreateFileResult, error) {
	// 1. Validate row count
	if len(rows) > MaxCreateFileRows {
		return nil, fmt.Errorf("%w: attempting to create file with %d rows, limit is %d",
			ErrRowLimitExceeded, len(rows), MaxCreateFileRows)
	}

	// 2. Check if file exists
	if _, err := os.Stat(path); err == nil {
		// File exists
		if !overwrite {
			return nil, fmt.Errorf("%w: %s", ErrFileExists, path)
		}
	} else if !os.IsNotExist(err) {
		// Some other error occurred while checking
		return nil, fmt.Errorf("failed to check if file exists: %w", err)
	}

	// 3. Create new file
	f := excelize.NewFile()
	defer f.Close()

	// 4. Rename default "Sheet1" to sheetName if provided
	finalSheetName := "Sheet1"
	if sheetName != "" {
		finalSheetName = sheetName
		// Get the default sheet index
		defaultSheetIndex, err := f.GetSheetIndex("Sheet1")
		if err != nil {
			return nil, fmt.Errorf("failed to get default sheet index: %w", err)
		}
		// Rename the default sheet
		if err := f.SetSheetName("Sheet1", finalSheetName); err != nil {
			return nil, fmt.Errorf("failed to rename sheet: %w", err)
		}
		// Set as active sheet
		f.SetActiveSheet(defaultSheetIndex)
	}

	rowsWritten := 0
	currentRow := 1

	// 5. If headers provided, write to row 1
	if len(headers) > 0 {
		headerCells := make([]any, len(headers))
		for i, header := range headers {
			headerCells[i] = header
		}
		cellAddr := FormatCellAddress(1, currentRow)
		if err := f.SetSheetRow(finalSheetName, cellAddr, &headerCells); err != nil {
			return nil, fmt.Errorf("failed to write headers: %w", err)
		}
		rowsWritten++
		currentRow++
	}

	// 6. Write rows
	for _, row := range rows {
		cells := make([]any, len(row))
		copy(cells, row)
		cellAddr := FormatCellAddress(1, currentRow)
		if err := f.SetSheetRow(finalSheetName, cellAddr, &cells); err != nil {
			return nil, fmt.Errorf("failed to write row %d: %w", currentRow, err)
		}
		rowsWritten++
		currentRow++
	}

	// 7. Save atomically
	if err := SaveFileAtomic(f, path); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 8. Return CreateFileResult
	return &CreateFileResult{
		Success:     true,
		File:        path,
		SheetName:   finalSheetName,
		RowsWritten: rowsWritten,
	}, nil
}

// WriteRange writes a 2D array of values starting at the specified cell.
// The data array is rows x columns. Enforces MaxWriteRangeCells limit.
func WriteRange(path, sheet, startCell string, data [][]any) (*WriteResult, error) {
	// 1. Calculate total cells and validate against MaxWriteRangeCells
	totalCells := 0
	for _, row := range data {
		totalCells += len(row)
	}
	if totalCells > MaxWriteRangeCells {
		return nil, fmt.Errorf("%w: attempting to write %d cells, limit is %d",
			ErrCellLimitExceeded, totalCells, MaxWriteRangeCells)
	}

	// 2. Open file for write
	f, err := OpenFileForWrite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for write: %w", err)
	}
	defer f.Close()

	// 3. Resolve sheet name
	resolvedSheet, err := ResolveSheetName(f, sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sheet name: %w", err)
	}

	// 4. Parse startCell to get starting row/col
	startCol, startRow, err := ParseCellAddress(startCell)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start cell %s: %w", startCell, err)
	}

	// 5. Iterate data and write each cell using setCellWithType
	for rowOffset, row := range data {
		currentRow := startRow + rowOffset
		for colOffset, value := range row {
			currentCol := startCol + colOffset
			cellAddr := FormatCellAddress(currentCol, currentRow)

			// Use auto type detection for each value
			if err := setCellWithType(f, resolvedSheet, cellAddr, value, "auto"); err != nil {
				return nil, fmt.Errorf("failed to write cell %s: %w", cellAddr, err)
			}
		}
	}

	// 6. Save atomically
	if err := SaveFileAtomic(f, path); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 7. Return WriteResult with cell count
	endCol := startCol + len(data[0]) - 1
	if len(data) == 0 {
		endCol = startCol
	}
	endRow := startRow + len(data) - 1
	if len(data) == 0 {
		endRow = startRow
	}

	rangeStr := fmt.Sprintf("%s:%s",
		FormatCellAddress(startCol, startRow),
		FormatCellAddress(endCol, endRow))

	return &WriteResult{
		Success:  true,
		Cell:     rangeStr,
		NewValue: fmt.Sprintf("Wrote %d cells", totalCells),
	}, nil
}

// CreateSheet creates a new sheet in an existing workbook.
// Optionally writes a header row.
func CreateSheet(path, name string, headers []string) (*SheetResult, error) {
	// 1. Open file for write
	f, err := OpenFileForWrite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for write: %w", err)
	}
	defer f.Close()

	// 2. Check if sheet already exists
	sheetIndex, err := f.GetSheetIndex(name)
	if err != nil {
		return nil, fmt.Errorf("failed to check if sheet exists: %w", err)
	}
	if sheetIndex != -1 {
		return nil, fmt.Errorf("%w: sheet %s already exists", ErrSheetExists, name)
	}

	// 3. Create new sheet
	_, err = f.NewSheet(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet %s: %w", name, err)
	}

	// 4. If headers provided, write to row 1
	if len(headers) > 0 {
		headerCells := make([]any, len(headers))
		for i, header := range headers {
			headerCells[i] = header
		}
		cellAddr := FormatCellAddress(1, 1)
		if err := f.SetSheetRow(name, cellAddr, &headerCells); err != nil {
			return nil, fmt.Errorf("failed to write headers: %w", err)
		}
	}

	// 5. Save atomically
	if err := SaveFileAtomic(f, path); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 6. Return SheetResult
	return &SheetResult{
		Success: true,
		Sheet:   name,
	}, nil
}

// DeleteSheet deletes a sheet from the workbook.
// Returns error if trying to delete the last sheet.
func DeleteSheet(path, sheet string) (*SheetResult, error) {
	// 1. Open file for write
	f, err := OpenFileForWrite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for write: %w", err)
	}
	defer f.Close()

	// 2. Verify sheet exists
	sheetIndex, err := f.GetSheetIndex(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to check sheet index: %w", err)
	}
	if sheetIndex == -1 {
		return nil, fmt.Errorf("%w: sheet %s does not exist", ErrSheetNotFound, sheet)
	}

	// 3. Verify it's not the last sheet
	sheets := f.GetSheetList()
	if len(sheets) <= 1 {
		return nil, fmt.Errorf("%w: workbook must have at least one sheet", ErrCannotDeleteLastSheet)
	}

	// 4. Delete the sheet
	if err := f.DeleteSheet(sheet); err != nil {
		return nil, fmt.Errorf("failed to delete sheet %s: %w", sheet, err)
	}

	// 5. Save atomically
	if err := SaveFileAtomic(f, path); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 6. Return SheetResult
	return &SheetResult{
		Success: true,
		Sheet:   sheet,
	}, nil
}

// RenameSheet renames a sheet in the workbook.
func RenameSheet(path, oldName, newName string) (*SheetResult, error) {
	// 1. Open file for write
	f, err := OpenFileForWrite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for write: %w", err)
	}
	defer f.Close()

	// 2. Verify old sheet exists
	oldSheetIndex, err := f.GetSheetIndex(oldName)
	if err != nil {
		return nil, fmt.Errorf("failed to check old sheet index: %w", err)
	}
	if oldSheetIndex == -1 {
		return nil, fmt.Errorf("%w: sheet %s does not exist", ErrSheetNotFound, oldName)
	}

	// 3. Verify new name doesn't exist
	newSheetIndex, err := f.GetSheetIndex(newName)
	if err != nil {
		return nil, fmt.Errorf("failed to check new sheet name: %w", err)
	}
	if newSheetIndex != -1 {
		return nil, fmt.Errorf("%w: sheet %s already exists", ErrSheetExists, newName)
	}

	// 4. Rename the sheet
	if err := f.SetSheetName(oldName, newName); err != nil {
		return nil, fmt.Errorf("failed to rename sheet from %s to %s: %w", oldName, newName, err)
	}

	// 5. Save atomically
	if err := SaveFileAtomic(f, path); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 6. Return SheetResult
	return &SheetResult{
		Success: true,
		Sheet:   newName,
	}, nil
}

// InsertRows inserts rows at a specific position, shifting existing rows down.
// The row parameter is 1-based. Enforces MaxAppendRows limit.
func InsertRows(path, sheet string, row int, data [][]any) (*AppendResult, error) {
	// 1. Validate len(data) <= MaxAppendRows
	if len(data) > MaxAppendRows {
		return nil, fmt.Errorf("%w: attempting to insert %d rows, limit is %d",
			ErrRowLimitExceeded, len(data), MaxAppendRows)
	}

	// 2. Validate row >= 1
	if row < 1 {
		return nil, fmt.Errorf("invalid row number: %d (must be >= 1)", row)
	}

	// 3. OpenFileForWrite(path)
	f, err := OpenFileForWrite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for write: %w", err)
	}
	defer f.Close()

	// 4. Resolve sheet name
	resolvedSheet, err := ResolveSheetName(f, sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sheet name: %w", err)
	}

	// 5. f.InsertRows(sheet, row, len(data)) - this shifts existing rows
	if err := f.InsertRows(resolvedSheet, row, len(data)); err != nil {
		return nil, fmt.Errorf("failed to insert rows at row %d: %w", row, err)
	}

	// 6. Write each row of data starting at `row`
	for i, rowData := range data {
		rowNum := row + i

		// Convert []any to []any for SetSheetRow
		cells := make([]any, len(rowData))
		copy(cells, rowData)

		// Use column A (1-based) as the starting cell
		cellAddr := FormatCellAddress(1, rowNum)
		if err := f.SetSheetRow(resolvedSheet, cellAddr, &cells); err != nil {
			return nil, fmt.Errorf("failed to write row %d: %w", rowNum, err)
		}
	}

	// 7. SaveFileAtomic()
	if err := SaveFileAtomic(f, path); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 8. Return AppendResult
	endingRow := row + len(data) - 1
	return &AppendResult{
		Success:     true,
		RowsAdded:   len(data),
		StartingRow: row,
		EndingRow:   endingRow,
	}, nil
}

// DeleteRows deletes rows starting at startRow.
// Both startRow and count are validated. Max 1000 rows can be deleted at once.
func DeleteRows(path, sheet string, startRow, count int) (*DeleteRowsResult, error) {
	// 1. Validate startRow >= 1 and count >= 1 and count <= MaxAppendRows
	if startRow < 1 {
		return nil, fmt.Errorf("invalid start row: %d (must be >= 1)", startRow)
	}
	if count < 1 {
		return nil, fmt.Errorf("invalid count: %d (must be >= 1)", count)
	}
	if count > MaxAppendRows {
		return nil, fmt.Errorf("%w: attempting to delete %d rows, limit is %d",
			ErrRowLimitExceeded, count, MaxAppendRows)
	}

	// 2. OpenFileForWrite(path)
	f, err := OpenFileForWrite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for write: %w", err)
	}
	defer f.Close()

	// 3. Resolve sheet name
	resolvedSheet, err := ResolveSheetName(f, sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sheet name: %w", err)
	}

	// 4. Delete rows in reverse order to maintain indices:
	//    for i := startRow + count - 1; i >= startRow; i-- {
	//        f.RemoveRow(sheet, i)
	//    }
	for i := startRow + count - 1; i >= startRow; i-- {
		if err := f.RemoveRow(resolvedSheet, i); err != nil {
			return nil, fmt.Errorf("failed to remove row %d: %w", i, err)
		}
	}

	// 5. SaveFileAtomic()
	if err := SaveFileAtomic(f, path); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 6. Return DeleteRowsResult
	return &DeleteRowsResult{
		Success:     true,
		RowsDeleted: count,
	}, nil
}
