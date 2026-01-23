package xlsx

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// createTestFile creates a minimal xlsx file for testing
func createTestFile(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	// Default sheet is Sheet1
	if err := f.SetCellValue("Sheet1", "A1", "Header1"); err != nil {
		t.Fatalf("failed to set cell: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "B1", "Header2"); err != nil {
		t.Fatalf("failed to set cell: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "A2", "Value1"); err != nil {
		t.Fatalf("failed to set cell: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "B2", 42); err != nil {
		t.Fatalf("failed to set cell: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "A3", "Value3"); err != nil {
		t.Fatalf("failed to set cell: %v", err)
	}

	// Create second sheet
	_, err := f.NewSheet("Sheet2")
	if err != nil {
		t.Fatalf("failed to create sheet: %v", err)
	}
	if err := f.SetCellValue("Sheet2", "A1", "Data"); err != nil {
		t.Fatalf("failed to set cell: %v", err)
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	return path
}

func TestOpenFile(t *testing.T) {
	path := createTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Test with non-existent file
	_, err = OpenFile("/nonexistent/file.xlsx")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestGetSheets(t *testing.T) {
	path := createTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	sheets, err := GetSheets(f)
	if err != nil {
		t.Fatalf("GetSheets failed: %v", err)
	}

	if len(sheets) != 2 {
		t.Errorf("expected 2 sheets, got %d", len(sheets))
	}

	// Test with nil file
	_, err = GetSheets(nil)
	if err == nil {
		t.Error("expected error for nil file")
	}
}

func TestGetSheetInfo(t *testing.T) {
	path := createTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	info, err := GetSheetInfo(f, "Sheet1")
	if err != nil {
		t.Fatalf("GetSheetInfo failed: %v", err)
	}

	if info.Name != "Sheet1" {
		t.Errorf("expected name 'Sheet1', got %q", info.Name)
	}

	if info.Rows != 3 {
		t.Errorf("expected 3 rows, got %d", info.Rows)
	}

	if info.Cols != 2 {
		t.Errorf("expected 2 cols, got %d", info.Cols)
	}

	if len(info.Headers) != 2 || info.Headers[0] != "Header1" {
		t.Errorf("headers mismatch: %v", info.Headers)
	}

	// Test case-insensitive sheet name
	info2, err := GetSheetInfo(f, "sheet1")
	if err != nil {
		t.Fatalf("case-insensitive lookup failed: %v", err)
	}
	if info2.Name != "Sheet1" {
		t.Errorf("expected normalized name 'Sheet1', got %q", info2.Name)
	}

	// Test non-existent sheet
	_, err = GetSheetInfo(f, "NonExistent")
	if err == nil {
		t.Error("expected error for non-existent sheet")
	}

	// Test with nil file
	_, err = GetSheetInfo(nil, "Sheet1")
	if err == nil {
		t.Error("expected error for nil file")
	}
}

func TestGetCell(t *testing.T) {
	path := createTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	cell, err := GetCell(f, "Sheet1", "A1")
	if err != nil {
		t.Fatalf("GetCell failed: %v", err)
	}

	if cell.Value != "Header1" {
		t.Errorf("expected 'Header1', got %q", cell.Value)
	}

	if cell.Address != "A1" {
		t.Errorf("expected address 'A1', got %q", cell.Address)
	}

	if cell.Row != 1 || cell.Col != 1 {
		t.Errorf("expected row=1, col=1, got row=%d, col=%d", cell.Row, cell.Col)
	}

	// Test number cell
	numCell, err := GetCell(f, "Sheet1", "B2")
	if err != nil {
		t.Fatalf("GetCell B2 failed: %v", err)
	}
	if numCell.Type != "number" {
		t.Errorf("expected type 'number', got %q", numCell.Type)
	}

	// Test invalid address
	_, err = GetCell(f, "Sheet1", "invalid")
	if err == nil {
		t.Error("expected error for invalid address")
	}

	// Test non-existent sheet
	_, err = GetCell(f, "NonExistent", "A1")
	if err == nil {
		t.Error("expected error for non-existent sheet")
	}

	// Test with nil file
	_, err = GetCell(nil, "Sheet1", "A1")
	if err == nil {
		t.Error("expected error for nil file")
	}

	// Test case-insensitive sheet name
	cell2, err := GetCell(f, "sheet1", "A1")
	if err != nil {
		t.Fatalf("case-insensitive GetCell failed: %v", err)
	}
	if cell2.Value != "Header1" {
		t.Errorf("expected 'Header1', got %q", cell2.Value)
	}
}

func TestGetDefaultSheet(t *testing.T) {
	path := createTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	defaultSheet, err := GetDefaultSheet(f)
	if err != nil {
		t.Fatalf("GetDefaultSheet failed: %v", err)
	}

	if defaultSheet == "" {
		t.Error("expected non-empty default sheet name")
	}

	// Verify it's a valid sheet
	if !SheetExists(f, defaultSheet) {
		t.Errorf("default sheet %q does not exist", defaultSheet)
	}
}

func TestDetectCellType(t *testing.T) {
	path := createTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	tests := []struct {
		name     string
		sheet    string
		addr     string
		expected string
	}{
		{"string cell", "Sheet1", "A1", "string"},
		{"number cell", "Sheet1", "B2", "number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cell, err := GetCell(f, tt.sheet, tt.addr)
			if err != nil {
				t.Fatalf("GetCell failed: %v", err)
			}
			if cell.Type != tt.expected {
				t.Errorf("expected type %q, got %q", tt.expected, cell.Type)
			}
		})
	}

	// Test empty cell
	emptyCell, err := GetCell(f, "Sheet1", "C1")
	if err != nil {
		t.Fatalf("GetCell for empty cell failed: %v", err)
	}
	if emptyCell.Type != "empty" {
		t.Errorf("expected type 'empty', got %q", emptyCell.Type)
	}
}

func TestSheetExists(t *testing.T) {
	path := createTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	if !SheetExists(f, "Sheet1") {
		t.Error("Sheet1 should exist")
	}

	if !SheetExists(f, "sheet1") { // case insensitive
		t.Error("sheet1 should match Sheet1")
	}

	if SheetExists(f, "NonExistent") {
		t.Error("NonExistent should not exist")
	}
}

func TestResolveSheetName(t *testing.T) {
	path := createTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Empty returns default
	name, err := ResolveSheetName(f, "")
	if err != nil {
		t.Fatalf("ResolveSheetName('') failed: %v", err)
	}
	if name == "" {
		t.Error("expected non-empty default sheet name")
	}

	// Case-insensitive resolution
	name, err = ResolveSheetName(f, "sheet2")
	if err != nil {
		t.Fatalf("ResolveSheetName('sheet2') failed: %v", err)
	}
	if name != "Sheet2" {
		t.Errorf("expected 'Sheet2', got %q", name)
	}

	// Non-existent
	_, err = ResolveSheetName(f, "NonExistent")
	if err == nil {
		t.Error("expected error for non-existent sheet")
	}

	// Exact match
	name, err = ResolveSheetName(f, "Sheet1")
	if err != nil {
		t.Fatalf("ResolveSheetName('Sheet1') failed: %v", err)
	}
	if name != "Sheet1" {
		t.Errorf("expected 'Sheet1', got %q", name)
	}
}
