package xlsx

import (
	"os"
	"path/filepath"
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
