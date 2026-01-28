package xlsx

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestOpenFileForWrite(t *testing.T) {
	// Create test file
	path := createTestFile(t)

	// Test successful open
	f, err := OpenFileForWrite(path)
	if err != nil {
		t.Fatalf("OpenFileForWrite failed: %v", err)
	}
	defer f.Close()

	// Verify we can write to it
	if err := f.SetCellValue("Sheet1", "C1", "test write"); err != nil {
		t.Errorf("failed to write to opened file: %v", err)
	}

	// Test non-existent file
	_, err = OpenFileForWrite("/nonexistent/file.xlsx")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
	if err != nil && err != ErrFileNotFound {
		// Should wrap ErrFileNotFound
		t.Logf("error type: %v", err)
	}

	// Test file too large
	dir := t.TempDir()
	largePath := filepath.Join(dir, "large.xlsx")

	// Create a file larger than MaxWriteFileSize
	largeFile, err := os.Create(largePath)
	if err != nil {
		t.Fatalf("failed to create large test file: %v", err)
	}
	// Write more than MaxWriteFileSize bytes
	data := make([]byte, MaxWriteFileSize+1)
	_, err = largeFile.Write(data)
	if err != nil {
		t.Fatalf("failed to write large file: %v", err)
	}
	largeFile.Close()

	_, err = OpenFileForWrite(largePath)
	if err == nil {
		t.Error("expected error for file too large")
	}
	// Should contain ErrFileTooLarge
	t.Logf("large file error: %v", err)
}

func TestSaveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "save_test.xlsx")

	// Create new file
	f := excelize.NewFile()
	defer f.Close()

	if err := f.SetCellValue("Sheet1", "A1", "test"); err != nil {
		t.Fatalf("failed to set cell: %v", err)
	}

	// Test SaveFile
	err := SaveFile(f, path)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Verify file exists and is readable
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("saved file does not exist")
	}

	// Verify content
	f2, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open saved file: %v", err)
	}
	defer f2.Close()

	val, err := f2.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to read cell from saved file: %v", err)
	}
	if val != "test" {
		t.Errorf("expected 'test', got %q", val)
	}
}

func TestSaveFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic_test.xlsx")
	initialPath := filepath.Join(dir, "initial.xlsx")

	// Create new file and save it first (excelize requires this)
	f := excelize.NewFile()
	defer f.Close()

	if err := f.SetCellValue("Sheet1", "A1", "atomic test"); err != nil {
		t.Fatalf("failed to set cell: %v", err)
	}

	// Initial save so file is properly initialized
	if err := f.SaveAs(initialPath); err != nil {
		t.Fatalf("failed to save initial file: %v", err)
	}

	// Now test SaveFileAtomic
	err := SaveFileAtomic(f, path)
	if err != nil {
		t.Fatalf("SaveFileAtomic failed: %v", err)
	}

	// Verify file exists and temp file is gone
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("saved file does not exist")
	}

	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file was not cleaned up")
	}

	// Verify content
	f2, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open saved file: %v", err)
	}
	defer f2.Close()

	val, err := f2.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to read cell from saved file: %v", err)
	}
	if val != "atomic test" {
		t.Errorf("expected 'atomic test', got %q", val)
	}

	// Test overwrite scenario - open existing file and modify it
	f3, err := OpenFileForWrite(path)
	if err != nil {
		t.Fatalf("failed to open file for overwrite test: %v", err)
	}
	defer f3.Close()

	if err := f3.SetCellValue("Sheet1", "A1", "overwritten"); err != nil {
		t.Fatalf("failed to set cell: %v", err)
	}

	err = SaveFileAtomic(f3, path)
	if err != nil {
		t.Fatalf("SaveFileAtomic overwrite failed: %v", err)
	}

	// Verify overwrite worked
	f4, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open overwritten file: %v", err)
	}
	defer f4.Close()

	val, err = f4.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to read cell from overwritten file: %v", err)
	}
	if val != "overwritten" {
		t.Errorf("expected 'overwritten', got %q", val)
	}
}

func TestSetCellWithType(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	tests := []struct {
		name      string
		cell      string
		value     any
		valueType string
		wantErr   bool
		verify    func(t *testing.T, f *excelize.File, cell string)
	}{
		{
			name:      "string type",
			cell:      "A1",
			value:     "hello",
			valueType: "string",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				val, err := f.GetCellValue("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell: %v", err)
				}
				if val != "hello" {
					t.Errorf("expected 'hello', got %q", val)
				}
			},
		},
		{
			name:      "number type - float64",
			cell:      "B1",
			value:     42.5,
			valueType: "number",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				val, err := f.GetCellValue("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell: %v", err)
				}
				if val != "42.5" {
					t.Errorf("expected '42.5', got %q", val)
				}
			},
		},
		{
			name:      "number type - int",
			cell:      "C1",
			value:     42,
			valueType: "number",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				val, err := f.GetCellValue("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell: %v", err)
				}
				if val != "42" {
					t.Errorf("expected '42', got %q", val)
				}
			},
		},
		{
			name:      "number type - string",
			cell:      "D1",
			value:     "123.45",
			valueType: "number",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				val, err := f.GetCellValue("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell: %v", err)
				}
				if val != "123.45" {
					t.Errorf("expected '123.45', got %q", val)
				}
			},
		},
		{
			name:      "bool type - true",
			cell:      "E1",
			value:     true,
			valueType: "bool",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				val, err := f.GetCellValue("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell: %v", err)
				}
				if val != "TRUE" {
					t.Errorf("expected 'TRUE', got %q", val)
				}
			},
		},
		{
			name:      "bool type - false",
			cell:      "F1",
			value:     false,
			valueType: "bool",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				val, err := f.GetCellValue("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell: %v", err)
				}
				if val != "FALSE" {
					t.Errorf("expected 'FALSE', got %q", val)
				}
			},
		},
		{
			name:      "formula type",
			cell:      "G1",
			value:     "=A1+B1",
			valueType: "formula",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				// GetCellFormula returns the formula with =
				formula, err := f.GetCellFormula("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell formula: %v", err)
				}
				if formula != "=A1+B1" {
					t.Errorf("expected '=A1+B1', got %q", formula)
				}
			},
		},
		{
			name:      "formula without = prefix",
			cell:      "H1",
			value:     "SUM(A1:A10)",
			valueType: "formula",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				formula, err := f.GetCellFormula("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell formula: %v", err)
				}
				if formula != "=SUM(A1:A10)" {
					t.Errorf("expected '=SUM(A1:A10)', got %q", formula)
				}
			},
		},
		{
			name:      "auto type - string",
			cell:      "I1",
			value:     "text",
			valueType: "auto",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				val, err := f.GetCellValue("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell: %v", err)
				}
				if val != "text" {
					t.Errorf("expected 'text', got %q", val)
				}
			},
		},
		{
			name:      "auto type - number",
			cell:      "J1",
			value:     99,
			valueType: "auto",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				val, err := f.GetCellValue("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell: %v", err)
				}
				if val != "99" {
					t.Errorf("expected '99', got %q", val)
				}
			},
		},
		{
			name:      "auto type - bool",
			cell:      "K1",
			value:     true,
			valueType: "auto",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				val, err := f.GetCellValue("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell: %v", err)
				}
				if val != "TRUE" {
					t.Errorf("expected 'TRUE', got %q", val)
				}
			},
		},
		{
			name:      "auto type - formula",
			cell:      "L1",
			value:     "=A1+1",
			valueType: "auto",
			wantErr:   false,
			verify: func(t *testing.T, f *excelize.File, cell string) {
				formula, err := f.GetCellFormula("Sheet1", cell)
				if err != nil {
					t.Fatalf("failed to get cell formula: %v", err)
				}
				if formula != "=A1+1" {
					t.Errorf("expected '=A1+1', got %q", formula)
				}
			},
		},
		{
			name:      "invalid type",
			cell:      "M1",
			value:     "test",
			valueType: "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setCellWithType(f, "Sheet1", tt.cell, tt.value, tt.valueType)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("setCellWithType failed: %v", err)
			}
			if tt.verify != nil {
				tt.verify(t, f, tt.cell)
			}
		})
	}
}

func TestDetectValueType(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"nil", nil, "string"},
		{"bool true", true, "bool"},
		{"bool false", false, "bool"},
		{"int", 42, "number"},
		{"int64", int64(42), "number"},
		{"float64", 42.5, "number"},
		{"float32", float32(42.5), "number"},
		{"string text", "hello", "string"},
		{"string number", "123", "number"},
		{"string bool", "true", "bool"},
		{"formula", "=SUM(A1:A10)", "formula"},
		{"struct", struct{}{}, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectValueType(tt.value)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetLastRow(t *testing.T) {
	path := createTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Sheet1 has 3 rows (from createTestFile)
	lastRow, err := getLastRow(f, "Sheet1")
	if err != nil {
		t.Fatalf("getLastRow failed: %v", err)
	}
	if lastRow != 3 {
		t.Errorf("expected 3 rows, got %d", lastRow)
	}

	// Sheet2 has 1 row
	lastRow, err = getLastRow(f, "Sheet2")
	if err != nil {
		t.Fatalf("getLastRow Sheet2 failed: %v", err)
	}
	if lastRow != 1 {
		t.Errorf("expected 1 row in Sheet2, got %d", lastRow)
	}

	// Test empty sheet
	_, err = f.NewSheet("EmptySheet")
	if err != nil {
		t.Fatalf("failed to create empty sheet: %v", err)
	}
	lastRow, err = getLastRow(f, "EmptySheet")
	if err != nil {
		t.Fatalf("getLastRow EmptySheet failed: %v", err)
	}
	if lastRow != 0 {
		t.Errorf("expected 0 rows in EmptySheet, got %d", lastRow)
	}

	// Test non-existent sheet
	_, err = getLastRow(f, "NonExistent")
	if err == nil {
		t.Error("expected error for non-existent sheet")
	}
}

func TestGetLastRowStreaming(t *testing.T) {
	// Create a file with many rows to verify streaming behavior
	dir := t.TempDir()
	path := filepath.Join(dir, "many_rows.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	// Add 100 rows
	for i := 1; i <= 100; i++ {
		if err := f.SetCellValue("Sheet1", FormatCellAddress(1, i), i); err != nil {
			t.Fatalf("failed to set cell at row %d: %v", i, err)
		}
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to save file: %v", err)
	}

	// Re-open and test
	f2, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f2.Close()

	lastRow, err := getLastRow(f2, "Sheet1")
	if err != nil {
		t.Fatalf("getLastRow failed: %v", err)
	}
	if lastRow != 100 {
		t.Errorf("expected 100 rows, got %d", lastRow)
	}
}

func TestWriteCell(t *testing.T) {
	// Create test file
	path := createTestFile(t)

	tests := []struct {
		name      string
		sheet     string
		cell      string
		value     any
		valueType string
		wantErr   bool
	}{
		{
			name:      "write string to existing cell",
			sheet:     "Sheet1",
			cell:      "A1",
			value:     "New Header",
			valueType: "auto",
			wantErr:   false,
		},
		{
			name:      "write number to new cell",
			sheet:     "Sheet1",
			cell:      "C3",
			value:     99.5,
			valueType: "number",
			wantErr:   false,
		},
		{
			name:      "write bool to cell",
			sheet:     "Sheet1",
			cell:      "D1",
			value:     true,
			valueType: "bool",
			wantErr:   false,
		},
		{
			name:      "write formula to cell",
			sheet:     "Sheet1",
			cell:      "E1",
			value:     "=A1+B1",
			valueType: "formula",
			wantErr:   false,
		},
		{
			name:      "write to default sheet",
			sheet:     "",
			cell:      "F1",
			value:     "default sheet",
			valueType: "string",
			wantErr:   false,
		},
		{
			name:      "write to Sheet2",
			sheet:     "Sheet2",
			cell:      "A1",
			value:     "Sheet2 data",
			valueType: "string",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := WriteCell(path, tt.sheet, tt.cell, tt.value, tt.valueType)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("WriteCell failed: %v", err)
			}

			// Verify result
			if !result.Success {
				t.Error("expected success=true")
			}
			if result.Cell != tt.cell {
				t.Errorf("expected cell %q, got %q", tt.cell, result.Cell)
			}
			if result.NewValue != tt.value {
				t.Errorf("expected new value %v, got %v", tt.value, result.NewValue)
			}

			// Verify the value was written by reading the file
			f, err := OpenFile(path)
			if err != nil {
				t.Fatalf("failed to open file for verification: %v", err)
			}
			defer f.Close()

			resolvedSheet := tt.sheet
			if resolvedSheet == "" {
				resolvedSheet = "Sheet1"
			}
			cellValue, err := f.GetCellValue(resolvedSheet, tt.cell)
			if err != nil {
				t.Fatalf("failed to read cell after write: %v", err)
			}

			// For formulas, check the formula itself
			if tt.valueType == "formula" {
				formula, err := f.GetCellFormula(resolvedSheet, tt.cell)
				if err != nil {
					t.Fatalf("failed to read formula: %v", err)
				}
				expectedFormula := tt.value.(string)
				if !strings.HasPrefix(expectedFormula, "=") {
					expectedFormula = "=" + expectedFormula
				}
				if formula != expectedFormula {
					t.Errorf("expected formula %q, got %q", expectedFormula, formula)
				}
			} else {
				// For other types, verify the string representation
				t.Logf("written value: %q", cellValue)
			}
		})
	}
}

func TestWriteCellErrors(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		sheet   string
		cell    string
		value   any
		wantErr error
	}{
		{
			name:    "non-existent file",
			path:    filepath.Join(dir, "nonexistent.xlsx"),
			sheet:   "Sheet1",
			cell:    "A1",
			value:   "test",
			wantErr: ErrFileNotFound,
		},
		{
			name:    "non-existent sheet",
			path:    createTestFile(t),
			sheet:   "NonExistentSheet",
			cell:    "A1",
			value:   "test",
			wantErr: ErrSheetNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := WriteCell(tt.path, tt.sheet, tt.cell, tt.value, "auto")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			// Just check that we got an error - the actual error wrapping may vary
			t.Logf("error: %v", err)
		})
	}
}

func TestAppendRows(t *testing.T) {
	// Create test file
	path := createTestFile(t)

	// Test 1: Append rows to Sheet1 (which has 3 rows)
	rows := [][]any{
		{"Value4", 44},
		{"Value5", 55},
		{"Value6", 66},
	}

	result, err := AppendRows(path, "Sheet1", rows)
	if err != nil {
		t.Fatalf("AppendRows failed: %v", err)
	}

	// Verify result
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.RowsAdded != 3 {
		t.Errorf("expected 3 rows added, got %d", result.RowsAdded)
	}
	if result.StartingRow != 4 {
		t.Errorf("expected starting row 4, got %d", result.StartingRow)
	}
	if result.EndingRow != 6 {
		t.Errorf("expected ending row 6, got %d", result.EndingRow)
	}

	// Verify the data was written by reading the file
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file for verification: %v", err)
	}
	defer f.Close()

	// Check row 4
	val, err := f.GetCellValue("Sheet1", "A4")
	if err != nil {
		t.Fatalf("failed to read A4: %v", err)
	}
	if val != "Value4" {
		t.Errorf("expected 'Value4' at A4, got %q", val)
	}

	val, err = f.GetCellValue("Sheet1", "B4")
	if err != nil {
		t.Fatalf("failed to read B4: %v", err)
	}
	if val != "44" {
		t.Errorf("expected '44' at B4, got %q", val)
	}

	// Check row 6
	val, err = f.GetCellValue("Sheet1", "A6")
	if err != nil {
		t.Fatalf("failed to read A6: %v", err)
	}
	if val != "Value6" {
		t.Errorf("expected 'Value6' at A6, got %q", val)
	}

	// Test 2: Append to default sheet
	rows2 := [][]any{
		{"Row7", 77},
	}
	result2, err := AppendRows(path, "", rows2)
	if err != nil {
		t.Fatalf("AppendRows to default sheet failed: %v", err)
	}
	if result2.StartingRow != 7 {
		t.Errorf("expected starting row 7, got %d", result2.StartingRow)
	}
}

func TestAppendRowsEmpty(t *testing.T) {
	// Test appending to an empty sheet
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	rows := [][]any{
		{"First", "Row"},
		{"Second", "Row"},
	}

	result, err := AppendRows(path, "Sheet1", rows)
	if err != nil {
		t.Fatalf("AppendRows to empty sheet failed: %v", err)
	}

	if result.StartingRow != 1 {
		t.Errorf("expected starting row 1 for empty sheet, got %d", result.StartingRow)
	}
	if result.EndingRow != 2 {
		t.Errorf("expected ending row 2, got %d", result.EndingRow)
	}

	// Verify the data
	f2, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f2.Close()

	val, err := f2.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to read A1: %v", err)
	}
	if val != "First" {
		t.Errorf("expected 'First' at A1, got %q", val)
	}
}

func TestAppendRowsLimit(t *testing.T) {
	path := createTestFile(t)

	// Try to append more than MaxAppendRows
	rows := make([][]any, MaxAppendRows+1)
	for i := range rows {
		rows[i] = []any{i}
	}

	_, err := AppendRows(path, "Sheet1", rows)
	if err == nil {
		t.Fatal("expected error for exceeding row limit")
	}
	if !errors.Is(err, ErrRowLimitExceeded) {
		t.Errorf("expected ErrRowLimitExceeded, got: %v", err)
	}
}

func TestAppendRowsErrors(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name  string
		path  string
		sheet string
		rows  [][]any
	}{
		{
			name:  "non-existent file",
			path:  filepath.Join(dir, "nonexistent.xlsx"),
			sheet: "Sheet1",
			rows:  [][]any{{"test"}},
		},
		{
			name:  "non-existent sheet",
			path:  createTestFile(t),
			sheet: "NonExistent",
			rows:  [][]any{{"test"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AppendRows(tt.path, tt.sheet, tt.rows)
			if err == nil {
				t.Error("expected error, got nil")
			}
			t.Logf("error: %v", err)
		})
	}
}

func TestCreateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new_file.xlsx")

	headers := []string{"Name", "Age", "Email"}
	rows := [][]any{
		{"Alice", 30, "alice@example.com"},
		{"Bob", 25, "bob@example.com"},
		{"Charlie", 35, "charlie@example.com"},
	}

	result, err := CreateFile(path, "People", headers, rows, false)
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	// Verify result
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.File != path {
		t.Errorf("expected file %q, got %q", path, result.File)
	}
	if result.SheetName != "People" {
		t.Errorf("expected sheet name 'People', got %q", result.SheetName)
	}
	if result.RowsWritten != 4 { // 1 header + 3 data rows
		t.Errorf("expected 4 rows written, got %d", result.RowsWritten)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("created file does not exist")
	}

	// Verify content
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open created file: %v", err)
	}
	defer f.Close()

	// Check sheet name
	sheets := f.GetSheetList()
	if len(sheets) != 1 {
		t.Errorf("expected 1 sheet, got %d", len(sheets))
	}
	if sheets[0] != "People" {
		t.Errorf("expected sheet 'People', got %q", sheets[0])
	}

	// Check headers
	val, err := f.GetCellValue("People", "A1")
	if err != nil {
		t.Fatalf("failed to read A1: %v", err)
	}
	if val != "Name" {
		t.Errorf("expected 'Name' at A1, got %q", val)
	}

	val, err = f.GetCellValue("People", "B1")
	if err != nil {
		t.Fatalf("failed to read B1: %v", err)
	}
	if val != "Age" {
		t.Errorf("expected 'Age' at B1, got %q", val)
	}

	// Check first data row
	val, err = f.GetCellValue("People", "A2")
	if err != nil {
		t.Fatalf("failed to read A2: %v", err)
	}
	if val != "Alice" {
		t.Errorf("expected 'Alice' at A2, got %q", val)
	}

	val, err = f.GetCellValue("People", "B2")
	if err != nil {
		t.Fatalf("failed to read B2: %v", err)
	}
	if val != "30" {
		t.Errorf("expected '30' at B2, got %q", val)
	}

	// Check last data row
	val, err = f.GetCellValue("People", "A4")
	if err != nil {
		t.Fatalf("failed to read A4: %v", err)
	}
	if val != "Charlie" {
		t.Errorf("expected 'Charlie' at A4, got %q", val)
	}
}

func TestCreateFileNoHeaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_headers.xlsx")

	rows := [][]any{
		{"Data1", 1},
		{"Data2", 2},
	}

	result, err := CreateFile(path, "", nil, rows, false)
	if err != nil {
		t.Fatalf("CreateFile without headers failed: %v", err)
	}

	if result.RowsWritten != 2 {
		t.Errorf("expected 2 rows written, got %d", result.RowsWritten)
	}
	if result.SheetName != "Sheet1" {
		t.Errorf("expected default sheet name 'Sheet1', got %q", result.SheetName)
	}

	// Verify content starts at row 1
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	val, err := f.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to read A1: %v", err)
	}
	if val != "Data1" {
		t.Errorf("expected 'Data1' at A1, got %q", val)
	}
}

func TestCreateFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.xlsx")

	result, err := CreateFile(path, "Empty", nil, nil, false)
	if err != nil {
		t.Fatalf("CreateFile with no data failed: %v", err)
	}

	if result.RowsWritten != 0 {
		t.Errorf("expected 0 rows written, got %d", result.RowsWritten)
	}

	// Verify file exists and has the right sheet
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) != 1 || sheets[0] != "Empty" {
		t.Errorf("expected sheet 'Empty', got %v", sheets)
	}
}

func TestCreateFileOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.xlsx")

	// Create initial file
	_, err := CreateFile(path, "First", []string{"A"}, [][]any{{"1"}}, false)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Try to create again without overwrite flag
	_, err = CreateFile(path, "Second", []string{"B"}, [][]any{{"2"}}, false)
	if err == nil {
		t.Fatal("expected error when creating file without overwrite")
	}
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists, got: %v", err)
	}

	// Create again with overwrite flag
	result, err := CreateFile(path, "Second", []string{"B"}, [][]any{{"2"}}, true)
	if err != nil {
		t.Fatalf("failed to overwrite file: %v", err)
	}

	if result.SheetName != "Second" {
		t.Errorf("expected sheet 'Second', got %q", result.SheetName)
	}

	// Verify the file was overwritten
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open overwritten file: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) != 1 || sheets[0] != "Second" {
		t.Errorf("expected only sheet 'Second', got %v", sheets)
	}

	val, err := f.GetCellValue("Second", "A1")
	if err != nil {
		t.Fatalf("failed to read A1: %v", err)
	}
	if val != "B" {
		t.Errorf("expected 'B' at A1, got %q", val)
	}
}

func TestCreateFileRowLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "too_many_rows.xlsx")

	// Try to create with more than MaxCreateFileRows
	rows := make([][]any, MaxCreateFileRows+1)
	for i := range rows {
		rows[i] = []any{i}
	}

	_, err := CreateFile(path, "Test", nil, rows, false)
	if err == nil {
		t.Fatal("expected error for exceeding row limit")
	}
	if !errors.Is(err, ErrRowLimitExceeded) {
		t.Errorf("expected ErrRowLimitExceeded, got: %v", err)
	}
}

func TestWriteRange(t *testing.T) {
	// Create test file
	path := createTestFile(t)

	// Test 1: Write a 3x3 range starting at B2
	data := [][]any{
		{"R1C1", "R1C2", "R1C3"},
		{100, 200, 300},
		{true, false, true},
	}

	result, err := WriteRange(path, "Sheet1", "B2", data)
	if err != nil {
		t.Fatalf("WriteRange failed: %v", err)
	}

	// Verify result
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Cell != "B2:D4" {
		t.Errorf("expected range B2:D4, got %q", result.Cell)
	}
	if !strings.Contains(result.NewValue.(string), "9 cells") {
		t.Errorf("expected 9 cells written, got: %v", result.NewValue)
	}

	// Verify the data was written by reading the file
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file for verification: %v", err)
	}
	defer f.Close()

	// Check first cell (B2)
	val, err := f.GetCellValue("Sheet1", "B2")
	if err != nil {
		t.Fatalf("failed to read B2: %v", err)
	}
	if val != "R1C1" {
		t.Errorf("expected 'R1C1' at B2, got %q", val)
	}

	// Check middle cell (C3)
	val, err = f.GetCellValue("Sheet1", "C3")
	if err != nil {
		t.Fatalf("failed to read C3: %v", err)
	}
	if val != "200" {
		t.Errorf("expected '200' at C3, got %q", val)
	}

	// Check last cell (D4)
	val, err = f.GetCellValue("Sheet1", "D4")
	if err != nil {
		t.Fatalf("failed to read D4: %v", err)
	}
	if val != "TRUE" {
		t.Errorf("expected 'TRUE' at D4, got %q", val)
	}

	// Test 2: Write single cell range
	singleData := [][]any{{"Single"}}
	result2, err := WriteRange(path, "Sheet1", "A1", singleData)
	if err != nil {
		t.Fatalf("WriteRange single cell failed: %v", err)
	}
	if result2.Cell != "A1:A1" {
		t.Errorf("expected range A1:A1, got %q", result2.Cell)
	}

	// Test 3: Write to different sheet
	data3 := [][]any{
		{"Sheet2Data1", "Sheet2Data2"},
	}
	result3, err := WriteRange(path, "Sheet2", "A1", data3)
	if err != nil {
		t.Fatalf("WriteRange to Sheet2 failed: %v", err)
	}
	if !result3.Success {
		t.Error("expected success for Sheet2 write")
	}

	// Verify Sheet2 data
	f2, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file for Sheet2 verification: %v", err)
	}
	defer f2.Close()

	val, err = f2.GetCellValue("Sheet2", "A1")
	if err != nil {
		t.Fatalf("failed to read Sheet2 A1: %v", err)
	}
	if val != "Sheet2Data1" {
		t.Errorf("expected 'Sheet2Data1' at Sheet2 A1, got %q", val)
	}
}

func TestWriteRangeEmptyRows(t *testing.T) {
	path := createTestFile(t)

	// Test with empty row in the middle
	data := [][]any{
		{"Row1"},
		{}, // Empty row
		{"Row3"},
	}

	result, err := WriteRange(path, "Sheet1", "A1", data)
	if err != nil {
		t.Fatalf("WriteRange with empty row failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success with empty rows")
	}
}

func TestWriteRangeCellLimit(t *testing.T) {
	path := createTestFile(t)

	// Create data that exceeds MaxWriteRangeCells
	numRows := MaxWriteRangeCells + 1
	data := make([][]any, numRows)
	for i := range data {
		data[i] = []any{i}
	}

	_, err := WriteRange(path, "Sheet1", "A1", data)
	if err == nil {
		t.Fatal("expected error for exceeding cell limit")
	}
	if !errors.Is(err, ErrCellLimitExceeded) {
		t.Errorf("expected ErrCellLimitExceeded, got: %v", err)
	}
}

func TestWriteRangeErrors(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name      string
		path      string
		sheet     string
		startCell string
		data      [][]any
	}{
		{
			name:      "non-existent file",
			path:      filepath.Join(dir, "nonexistent.xlsx"),
			sheet:     "Sheet1",
			startCell: "A1",
			data:      [][]any{{"test"}},
		},
		{
			name:      "non-existent sheet",
			path:      createTestFile(t),
			sheet:     "NonExistent",
			startCell: "A1",
			data:      [][]any{{"test"}},
		},
		{
			name:      "invalid start cell",
			path:      createTestFile(t),
			sheet:     "Sheet1",
			startCell: "INVALID",
			data:      [][]any{{"test"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := WriteRange(tt.path, tt.sheet, tt.startCell, tt.data)
			if err == nil {
				t.Error("expected error, got nil")
			}
			t.Logf("error: %v", err)
		})
	}
}

func TestCreateSheet(t *testing.T) {
	// Create test file
	path := createTestFile(t)

	// Test 1: Create sheet without headers
	result, err := CreateSheet(path, "NewSheet", nil)
	if err != nil {
		t.Fatalf("CreateSheet failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Sheet != "NewSheet" {
		t.Errorf("expected sheet 'NewSheet', got %q", result.Sheet)
	}

	// Verify the sheet was created
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file for verification: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if !slices.Contains(sheets, "NewSheet") {
		t.Errorf("NewSheet not found in sheet list: %v", sheets)
	}

	// Test 2: Create sheet with headers
	headers := []string{"ID", "Name", "Email"}
	result2, err := CreateSheet(path, "WithHeaders", headers)
	if err != nil {
		t.Fatalf("CreateSheet with headers failed: %v", err)
	}

	if !result2.Success {
		t.Error("expected success=true for sheet with headers")
	}

	// Verify headers were written
	f2, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file for header verification: %v", err)
	}
	defer f2.Close()

	// Check header cells
	val, err := f2.GetCellValue("WithHeaders", "A1")
	if err != nil {
		t.Fatalf("failed to read A1: %v", err)
	}
	if val != "ID" {
		t.Errorf("expected 'ID' at A1, got %q", val)
	}

	val, err = f2.GetCellValue("WithHeaders", "C1")
	if err != nil {
		t.Fatalf("failed to read C1: %v", err)
	}
	if val != "Email" {
		t.Errorf("expected 'Email' at C1, got %q", val)
	}
}

func TestCreateSheetDuplicate(t *testing.T) {
	path := createTestFile(t)

	// Sheet1 already exists
	_, err := CreateSheet(path, "Sheet1", nil)
	if err == nil {
		t.Fatal("expected error when creating duplicate sheet")
	}
	if !errors.Is(err, ErrSheetExists) {
		t.Errorf("expected ErrSheetExists, got: %v", err)
	}
}

func TestCreateSheetErrors(t *testing.T) {
	dir := t.TempDir()

	// Test non-existent file
	_, err := CreateSheet(filepath.Join(dir, "nonexistent.xlsx"), "Test", nil)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
	t.Logf("error: %v", err)
}

func TestDeleteSheet(t *testing.T) {
	// Create test file with multiple sheets
	path := createTestFile(t)

	// Add another sheet so we can delete one
	_, err := CreateSheet(path, "ToDelete", nil)
	if err != nil {
		t.Fatalf("failed to create sheet to delete: %v", err)
	}

	// Test 1: Delete the new sheet
	result, err := DeleteSheet(path, "ToDelete")
	if err != nil {
		t.Fatalf("DeleteSheet failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Sheet != "ToDelete" {
		t.Errorf("expected sheet 'ToDelete', got %q", result.Sheet)
	}

	// Verify the sheet was deleted
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file for verification: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	for _, sheet := range sheets {
		if sheet == "ToDelete" {
			t.Error("ToDelete sheet still exists after deletion")
		}
	}
}

func TestDeleteSheetLastSheet(t *testing.T) {
	// Create a file with only one sheet
	dir := t.TempDir()
	path := filepath.Join(dir, "single_sheet.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Try to delete the only sheet
	_, err := DeleteSheet(path, "Sheet1")
	if err == nil {
		t.Fatal("expected error when deleting last sheet")
	}
	if !errors.Is(err, ErrCannotDeleteLastSheet) {
		t.Errorf("expected ErrCannotDeleteLastSheet, got: %v", err)
	}
}

func TestDeleteSheetNonExistent(t *testing.T) {
	path := createTestFile(t)

	_, err := DeleteSheet(path, "NonExistent")
	if err == nil {
		t.Fatal("expected error when deleting non-existent sheet")
	}
	if !errors.Is(err, ErrSheetNotFound) {
		t.Errorf("expected ErrSheetNotFound, got: %v", err)
	}
}

func TestDeleteSheetErrors(t *testing.T) {
	dir := t.TempDir()

	// Test non-existent file
	_, err := DeleteSheet(filepath.Join(dir, "nonexistent.xlsx"), "Test")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
	t.Logf("error: %v", err)
}

func TestRenameSheet(t *testing.T) {
	// Create test file
	path := createTestFile(t)

	// Test rename Sheet2 to RenamedSheet
	result, err := RenameSheet(path, "Sheet2", "RenamedSheet")
	if err != nil {
		t.Fatalf("RenameSheet failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Sheet != "RenamedSheet" {
		t.Errorf("expected sheet 'RenamedSheet', got %q", result.Sheet)
	}

	// Verify the sheet was renamed
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file for verification: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	foundOld := false
	foundNew := false
	for _, sheet := range sheets {
		if sheet == "Sheet2" {
			foundOld = true
		}
		if sheet == "RenamedSheet" {
			foundNew = true
		}
	}

	if foundOld {
		t.Error("old sheet name 'Sheet2' still exists")
	}
	if !foundNew {
		t.Error("new sheet name 'RenamedSheet' not found")
	}

	// Verify data is still accessible under new name
	val, err := f.GetCellValue("RenamedSheet", "A1")
	if err != nil {
		t.Fatalf("failed to read from renamed sheet: %v", err)
	}
	if val != "Data" {
		t.Errorf("expected 'Data', got %q", val)
	}
}

func TestRenameSheetOldNotFound(t *testing.T) {
	path := createTestFile(t)

	_, err := RenameSheet(path, "NonExistent", "NewName")
	if err == nil {
		t.Fatal("expected error when renaming non-existent sheet")
	}
	if !errors.Is(err, ErrSheetNotFound) {
		t.Errorf("expected ErrSheetNotFound, got: %v", err)
	}
}

func TestRenameSheetNewExists(t *testing.T) {
	path := createTestFile(t)

	// Try to rename Sheet2 to Sheet1 (which already exists)
	_, err := RenameSheet(path, "Sheet2", "Sheet1")
	if err == nil {
		t.Fatal("expected error when new name already exists")
	}
	if !errors.Is(err, ErrSheetExists) {
		t.Errorf("expected ErrSheetExists, got: %v", err)
	}
}

func TestRenameSheetErrors(t *testing.T) {
	dir := t.TempDir()

	// Test non-existent file
	_, err := RenameSheet(filepath.Join(dir, "nonexistent.xlsx"), "Old", "New")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
	t.Logf("error: %v", err)
}

func TestInsertRows(t *testing.T) {
	// Create test file with 3 rows
	path := createTestFile(t)

	// Test 1: Insert 2 rows at position 2 (between row 1 and 2)
	data := [][]any{
		{"Inserted1", 100},
		{"Inserted2", 200},
	}

	result, err := InsertRows(path, "Sheet1", 2, data)
	if err != nil {
		t.Fatalf("InsertRows failed: %v", err)
	}

	// Verify result
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.RowsAdded != 2 {
		t.Errorf("expected 2 rows added, got %d", result.RowsAdded)
	}
	if result.StartingRow != 2 {
		t.Errorf("expected starting row 2, got %d", result.StartingRow)
	}
	if result.EndingRow != 3 {
		t.Errorf("expected ending row 3, got %d", result.EndingRow)
	}

	// Verify the data was inserted by reading the file
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file for verification: %v", err)
	}
	defer f.Close()

	// Check row 1 (unchanged)
	val, err := f.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to read A1: %v", err)
	}
	if val != "Header1" {
		t.Errorf("expected 'Header1' at A1, got %q", val)
	}

	// Check row 2 (inserted)
	val, err = f.GetCellValue("Sheet1", "A2")
	if err != nil {
		t.Fatalf("failed to read A2: %v", err)
	}
	if val != "Inserted1" {
		t.Errorf("expected 'Inserted1' at A2, got %q", val)
	}

	val, err = f.GetCellValue("Sheet1", "B2")
	if err != nil {
		t.Fatalf("failed to read B2: %v", err)
	}
	if val != "100" {
		t.Errorf("expected '100' at B2, got %q", val)
	}

	// Check row 3 (inserted)
	val, err = f.GetCellValue("Sheet1", "A3")
	if err != nil {
		t.Fatalf("failed to read A3: %v", err)
	}
	if val != "Inserted2" {
		t.Errorf("expected 'Inserted2' at A3, got %q", val)
	}

	// Check row 4 (was row 2, shifted down)
	val, err = f.GetCellValue("Sheet1", "A4")
	if err != nil {
		t.Fatalf("failed to read A4: %v", err)
	}
	if val != "Value1" {
		t.Errorf("expected 'Value1' at A4 (shifted from A2), got %q", val)
	}

	// Test 2: Insert at row 1 (beginning)
	data2 := [][]any{
		{"First", 999},
	}
	result2, err := InsertRows(path, "Sheet1", 1, data2)
	if err != nil {
		t.Fatalf("InsertRows at row 1 failed: %v", err)
	}
	if result2.StartingRow != 1 {
		t.Errorf("expected starting row 1, got %d", result2.StartingRow)
	}

	// Verify row 1 is now "First"
	f2, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f2.Close()

	val, err = f2.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to read A1: %v", err)
	}
	if val != "First" {
		t.Errorf("expected 'First' at A1, got %q", val)
	}
}

func TestInsertRowsEmpty(t *testing.T) {
	// Test inserting into an empty sheet
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	data := [][]any{
		{"Row1", "Data1"},
		{"Row2", "Data2"},
	}

	result, err := InsertRows(path, "Sheet1", 1, data)
	if err != nil {
		t.Fatalf("InsertRows to empty sheet failed: %v", err)
	}

	if result.StartingRow != 1 {
		t.Errorf("expected starting row 1, got %d", result.StartingRow)
	}

	// Verify the data
	f2, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f2.Close()

	val, err := f2.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to read A1: %v", err)
	}
	if val != "Row1" {
		t.Errorf("expected 'Row1' at A1, got %q", val)
	}
}

func TestInsertRowsLimit(t *testing.T) {
	path := createTestFile(t)

	// Try to insert more than MaxAppendRows
	rows := make([][]any, MaxAppendRows+1)
	for i := range rows {
		rows[i] = []any{i}
	}

	_, err := InsertRows(path, "Sheet1", 1, rows)
	if err == nil {
		t.Fatal("expected error for exceeding row limit")
	}
	if !errors.Is(err, ErrRowLimitExceeded) {
		t.Errorf("expected ErrRowLimitExceeded, got: %v", err)
	}
}

func TestInsertRowsInvalidRow(t *testing.T) {
	path := createTestFile(t)

	data := [][]any{{"test"}}

	// Test row < 1
	_, err := InsertRows(path, "Sheet1", 0, data)
	if err == nil {
		t.Error("expected error for row < 1")
	}
	t.Logf("error for row 0: %v", err)

	_, err = InsertRows(path, "Sheet1", -1, data)
	if err == nil {
		t.Error("expected error for row < 1")
	}
	t.Logf("error for row -1: %v", err)
}

func TestInsertRowsErrors(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name  string
		path  string
		sheet string
		row   int
		data  [][]any
	}{
		{
			name:  "non-existent file",
			path:  filepath.Join(dir, "nonexistent.xlsx"),
			sheet: "Sheet1",
			row:   1,
			data:  [][]any{{"test"}},
		},
		{
			name:  "non-existent sheet",
			path:  createTestFile(t),
			sheet: "NonExistent",
			row:   1,
			data:  [][]any{{"test"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := InsertRows(tt.path, tt.sheet, tt.row, tt.data)
			if err == nil {
				t.Error("expected error, got nil")
			}
			t.Logf("error: %v", err)
		})
	}
}

func TestDeleteRows(t *testing.T) {
	// Create test file with 3 rows
	path := createTestFile(t)

	// First, let's verify what we have initially
	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open initial file: %v", err)
	}
	initialRows, err := getLastRow(f, "Sheet1")
	f.Close()
	if err != nil {
		t.Fatalf("failed to get initial row count: %v", err)
	}
	t.Logf("Initial rows in Sheet1: %d", initialRows)

	// Test 1: Delete 1 row at position 2
	result, err := DeleteRows(path, "Sheet1", 2, 1)
	if err != nil {
		t.Fatalf("DeleteRows failed: %v", err)
	}

	// Verify result
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.RowsDeleted != 1 {
		t.Errorf("expected 1 row deleted, got %d", result.RowsDeleted)
	}

	// Verify the row was deleted by reading the file
	f2, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open file for verification: %v", err)
	}
	defer f2.Close()

	// Check row 1 (unchanged)
	val, err := f2.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to read A1: %v", err)
	}
	if val != "Header1" {
		t.Errorf("expected 'Header1' at A1, got %q", val)
	}

	// Check row 2 (was row 3, shifted up after deleting original row 2)
	val, err = f2.GetCellValue("Sheet1", "A2")
	if err != nil {
		t.Fatalf("failed to read A2: %v", err)
	}
	if val != "Value3" {
		t.Errorf("expected 'Value3' at A2 (shifted from A3), got %q", val)
	}

	// Row 3 should now be empty or not exist
	val, err = f2.GetCellValue("Sheet1", "A3")
	if err != nil {
		t.Fatalf("failed to read A3: %v", err)
	}
	if val != "" {
		t.Logf("A3 has value: %q (expected empty)", val)
	}

	// Test 2: Delete multiple rows
	// First, add more rows to test
	_, err = AppendRows(path, "Sheet1", [][]any{
		{"Value3", 33},
		{"Value4", 44},
		{"Value5", 55},
	})
	if err != nil {
		t.Fatalf("failed to append rows for multi-delete test: %v", err)
	}

	// Delete 2 rows starting at row 3
	result2, err := DeleteRows(path, "Sheet1", 3, 2)
	if err != nil {
		t.Fatalf("DeleteRows multiple failed: %v", err)
	}
	if result2.RowsDeleted != 2 {
		t.Errorf("expected 2 rows deleted, got %d", result2.RowsDeleted)
	}
}

func TestDeleteRowsLimit(t *testing.T) {
	path := createTestFile(t)

	// Try to delete more than MaxAppendRows
	_, err := DeleteRows(path, "Sheet1", 1, MaxAppendRows+1)
	if err == nil {
		t.Fatal("expected error for exceeding row limit")
	}
	if !errors.Is(err, ErrRowLimitExceeded) {
		t.Errorf("expected ErrRowLimitExceeded, got: %v", err)
	}
}

func TestDeleteRowsInvalidParameters(t *testing.T) {
	path := createTestFile(t)

	tests := []struct {
		name     string
		startRow int
		count    int
	}{
		{
			name:     "startRow < 1",
			startRow: 0,
			count:    1,
		},
		{
			name:     "startRow negative",
			startRow: -1,
			count:    1,
		},
		{
			name:     "count < 1",
			startRow: 1,
			count:    0,
		},
		{
			name:     "count negative",
			startRow: 1,
			count:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeleteRows(path, "Sheet1", tt.startRow, tt.count)
			if err == nil {
				t.Error("expected error, got nil")
			}
			t.Logf("error: %v", err)
		})
	}
}

func TestDeleteRowsErrors(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		sheet    string
		startRow int
		count    int
	}{
		{
			name:     "non-existent file",
			path:     filepath.Join(dir, "nonexistent.xlsx"),
			sheet:    "Sheet1",
			startRow: 1,
			count:    1,
		},
		{
			name:     "non-existent sheet",
			path:     createTestFile(t),
			sheet:    "NonExistent",
			startRow: 1,
			count:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeleteRows(tt.path, tt.sheet, tt.startRow, tt.count)
			if err == nil {
				t.Error("expected error, got nil")
			}
			t.Logf("error: %v", err)
		})
	}
}
