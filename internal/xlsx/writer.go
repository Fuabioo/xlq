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
