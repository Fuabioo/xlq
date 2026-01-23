package xlsx

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// createLargeTestFile creates a test file with many rows
func createLargeTestFile(t *testing.T, numRows int) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "large.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	// Create rows
	for i := 1; i <= numRows; i++ {
		if err := f.SetCellValue("Sheet1", FormatCellAddress(1, i), i); err != nil {
			t.Fatalf("failed to set cell A%d: %v", i, err)
		}
		if err := f.SetCellValue("Sheet1", FormatCellAddress(2, i), i*10); err != nil {
			t.Fatalf("failed to set cell B%d: %v", i, err)
		}
		if err := f.SetCellValue("Sheet1", FormatCellAddress(3, i), i*100); err != nil {
			t.Fatalf("failed to set cell C%d: %v", i, err)
		}
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	return path
}

func TestStreamRows(t *testing.T) {
	path := createLargeTestFile(t, 100)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Stream rows 10-20
	ch, err := StreamRows(f, "Sheet1", 10, 20)
	if err != nil {
		t.Fatalf("StreamRows failed: %v", err)
	}

	rows, err := CollectRows(ch)
	if err != nil {
		t.Fatalf("CollectRows failed: %v", err)
	}

	if len(rows) != 11 { // rows 10 through 20 inclusive
		t.Errorf("expected 11 rows, got %d", len(rows))
	}

	if rows[0].Number != 10 {
		t.Errorf("expected first row number 10, got %d", rows[0].Number)
	}

	if rows[10].Number != 20 {
		t.Errorf("expected last row number 20, got %d", rows[10].Number)
	}
}

func TestStreamRowsToEnd(t *testing.T) {
	path := createLargeTestFile(t, 50)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Stream from row 45 to end (endRow = 0)
	ch, err := StreamRows(f, "Sheet1", 45, 0)
	if err != nil {
		t.Fatalf("StreamRows failed: %v", err)
	}

	rows, err := CollectRows(ch)
	if err != nil {
		t.Fatalf("CollectRows failed: %v", err)
	}

	if len(rows) != 6 { // rows 45-50
		t.Errorf("expected 6 rows, got %d", len(rows))
	}

	if rows[0].Number != 45 {
		t.Errorf("expected first row number 45, got %d", rows[0].Number)
	}
}

func TestStreamRange(t *testing.T) {
	path := createLargeTestFile(t, 50)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	ch, err := StreamRange(f, "Sheet1", "B5:C10")
	if err != nil {
		t.Fatalf("StreamRange failed: %v", err)
	}

	rows, err := CollectRows(ch)
	if err != nil {
		t.Fatalf("CollectRows failed: %v", err)
	}

	if len(rows) != 6 { // rows 5-10
		t.Errorf("expected 6 rows, got %d", len(rows))
	}

	// Each row should have 2 cells (B and C)
	if len(rows[0].Cells) != 2 {
		t.Errorf("expected 2 cells per row, got %d", len(rows[0].Cells))
	}

	// First cell should be B5
	if rows[0].Cells[0].Address != "B5" {
		t.Errorf("expected address B5, got %s", rows[0].Cells[0].Address)
	}

	// Verify cell B5 value (row 5, column 2 = 5*10 = 50)
	if rows[0].Cells[0].Value != "50" {
		t.Errorf("expected value 50, got %s", rows[0].Cells[0].Value)
	}
}

func TestStreamRangeSingleCell(t *testing.T) {
	path := createLargeTestFile(t, 10)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Single cell range
	ch, err := StreamRange(f, "Sheet1", "B5")
	if err != nil {
		t.Fatalf("StreamRange failed: %v", err)
	}

	rows, err := CollectRows(ch)
	if err != nil {
		t.Fatalf("CollectRows failed: %v", err)
	}

	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}

	if len(rows[0].Cells) != 1 {
		t.Errorf("expected 1 cell, got %d", len(rows[0].Cells))
	}

	if rows[0].Cells[0].Address != "B5" {
		t.Errorf("expected address B5, got %s", rows[0].Cells[0].Address)
	}
}

func TestStreamHead(t *testing.T) {
	path := createLargeTestFile(t, 50)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	ch, err := StreamHead(f, "Sheet1", 5)
	if err != nil {
		t.Fatalf("StreamHead failed: %v", err)
	}

	rows, err := CollectRows(ch)
	if err != nil {
		t.Fatalf("CollectRows failed: %v", err)
	}

	if len(rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(rows))
	}

	if rows[0].Number != 1 {
		t.Errorf("expected first row number 1, got %d", rows[0].Number)
	}

	if rows[4].Number != 5 {
		t.Errorf("expected last row number 5, got %d", rows[4].Number)
	}
}

func TestStreamHeadDefault(t *testing.T) {
	path := createLargeTestFile(t, 50)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Pass 0 to test default behavior (should default to 10)
	ch, err := StreamHead(f, "Sheet1", 0)
	if err != nil {
		t.Fatalf("StreamHead failed: %v", err)
	}

	rows, err := CollectRows(ch)
	if err != nil {
		t.Fatalf("CollectRows failed: %v", err)
	}

	if len(rows) != 10 {
		t.Errorf("expected 10 rows (default), got %d", len(rows))
	}
}

func TestStreamTail(t *testing.T) {
	path := createLargeTestFile(t, 50)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	rows, err := StreamTail(f, "Sheet1", 5)
	if err != nil {
		t.Fatalf("StreamTail failed: %v", err)
	}

	if len(rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(rows))
	}

	// Last row should be row 50
	if rows[4].Number != 50 {
		t.Errorf("expected last row number 50, got %d", rows[4].Number)
	}

	// First of tail should be row 46
	if rows[0].Number != 46 {
		t.Errorf("expected first tail row number 46, got %d", rows[0].Number)
	}
}

func TestStreamTailSmallFile(t *testing.T) {
	path := createLargeTestFile(t, 3) // Only 3 rows

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	rows, err := StreamTail(f, "Sheet1", 10) // Request more than available
	if err != nil {
		t.Fatalf("StreamTail failed: %v", err)
	}

	if len(rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(rows))
	}

	// Should return rows in order 1, 2, 3
	if rows[0].Number != 1 {
		t.Errorf("expected first row number 1, got %d", rows[0].Number)
	}
	if rows[2].Number != 3 {
		t.Errorf("expected last row number 3, got %d", rows[2].Number)
	}
}

func TestStreamTailDefault(t *testing.T) {
	path := createLargeTestFile(t, 50)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Pass 0 to test default behavior (should default to 10)
	rows, err := StreamTail(f, "Sheet1", 0)
	if err != nil {
		t.Fatalf("StreamTail failed: %v", err)
	}

	if len(rows) != 10 {
		t.Errorf("expected 10 rows (default), got %d", len(rows))
	}

	// Last row should be row 50
	if rows[9].Number != 50 {
		t.Errorf("expected last row number 50, got %d", rows[9].Number)
	}
}

func TestStreamTailEmptySheet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	f2, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f2.Close()

	rows, err := StreamTail(f2, "Sheet1", 5)
	if err != nil {
		t.Fatalf("StreamTail failed: %v", err)
	}

	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

func TestRowsToStringSlice(t *testing.T) {
	rows := []Row{
		{Number: 1, Cells: []Cell{{Value: "a"}, {Value: "b"}}},
		{Number: 2, Cells: []Cell{{Value: "c"}, {Value: "d"}}},
	}

	result := RowsToStringSlice(rows)

	if len(result) != 2 {
		t.Errorf("expected 2 rows, got %d", len(result))
	}

	if result[0][0] != "a" || result[0][1] != "b" {
		t.Errorf("unexpected first row: %v", result[0])
	}

	if result[1][0] != "c" || result[1][1] != "d" {
		t.Errorf("unexpected second row: %v", result[1])
	}
}

func TestStreamRowsToStrings(t *testing.T) {
	path := createLargeTestFile(t, 10)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	result, err := StreamRowsToStrings(f, "Sheet1", 1, 3)
	if err != nil {
		t.Fatalf("StreamRowsToStrings failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 rows, got %d", len(result))
	}

	// Verify content of first row
	if len(result[0]) != 3 {
		t.Errorf("expected 3 columns, got %d", len(result[0]))
	}

	if result[0][0] != "1" {
		t.Errorf("expected first cell to be '1', got '%s'", result[0][0])
	}
}

func TestStreamRowsInvalidSheet(t *testing.T) {
	path := createLargeTestFile(t, 10)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	_, err = StreamRows(f, "NonExistentSheet", 1, 10)
	if err == nil {
		t.Error("expected error for non-existent sheet, got nil")
	}
}

func TestStreamRangeInvalidRange(t *testing.T) {
	path := createLargeTestFile(t, 10)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	_, err = StreamRange(f, "Sheet1", "INVALID")
	if err == nil {
		t.Error("expected error for invalid range, got nil")
	}
}

func TestCollectRowsWithError(t *testing.T) {
	// This test verifies error handling in the channel
	// We can't easily simulate this without mocking, but we can test the logic
	ch := make(chan RowResult)

	go func() {
		ch <- RowResult{Row: &Row{Number: 1, Cells: []Cell{{Value: "a"}}}}
		ch <- RowResult{Err: fmt.Errorf("test error")}
		close(ch)
	}()

	_, err := CollectRows(ch)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestStreamRowsDefaultSheet(t *testing.T) {
	path := createLargeTestFile(t, 10)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Pass empty sheet name to test default sheet resolution
	ch, err := StreamRows(f, "", 1, 5)
	if err != nil {
		t.Fatalf("StreamRows with default sheet failed: %v", err)
	}

	rows, err := CollectRows(ch)
	if err != nil {
		t.Fatalf("CollectRows failed: %v", err)
	}

	if len(rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(rows))
	}
}

// Benchmark tests
func BenchmarkStreamRows(b *testing.B) {
	path := createLargeTestFile(&testing.T{}, 1000)

	f, err := OpenFile(path)
	if err != nil {
		b.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ch, err := StreamRows(f, "Sheet1", 1, 100)
		if err != nil {
			b.Fatalf("StreamRows failed: %v", err)
		}

		_, err = CollectRows(ch)
		if err != nil {
			b.Fatalf("CollectRows failed: %v", err)
		}
	}
}

func BenchmarkStreamTail(b *testing.B) {
	path := createLargeTestFile(&testing.T{}, 1000)

	f, err := OpenFile(path)
	if err != nil {
		b.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := StreamTail(f, "Sheet1", 10)
		if err != nil {
			b.Fatalf("StreamTail failed: %v", err)
		}
	}
}
