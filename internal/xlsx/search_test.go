package xlsx

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func createSearchTestFile(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "search.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	// Sheet1 with various data
	if err := f.SetCellValue("Sheet1", "A1", "Hello World"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "B1", "hello"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "A2", "Goodbye"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "B2", "Test123"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "A3", "hello again"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}

	// Create Sheet2
	if _, err := f.NewSheet("Sheet2"); err != nil {
		t.Fatalf("failed to create sheet: %v", err)
	}
	if err := f.SetCellValue("Sheet2", "A1", "Another Hello"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet2", "B1", "Data"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	return path
}

func TestSearchBasic(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Search for "hello" case-insensitive
	results, err := SearchSimple(f, "hello", true)
	if err != nil {
		t.Fatalf("SearchSimple failed: %v", err)
	}

	// Should find: "Hello World", "hello", "hello again", "Another Hello"
	if len(results) != 4 {
		t.Errorf("expected 4 results, got %d", len(results))
		for _, r := range results {
			t.Logf("  found: %s/%s = %q", r.Sheet, r.Address, r.Value)
		}
	}
}

func TestSearchCaseSensitive(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Case-sensitive search for "hello"
	results, err := SearchSimple(f, "hello", false)
	if err != nil {
		t.Fatalf("SearchSimple failed: %v", err)
	}

	// Should find: "hello", "hello again"
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSearchInSheet(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Search only in Sheet1
	results, err := SearchInSheet(f, "Sheet1", "hello", true)
	if err != nil {
		t.Fatalf("SearchInSheet failed: %v", err)
	}

	// Should find: "Hello World", "hello", "hello again" (not "Another Hello" from Sheet2)
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// All results should be from Sheet1
	for _, r := range results {
		if r.Sheet != "Sheet1" {
			t.Errorf("unexpected sheet: %s", r.Sheet)
		}
	}
}

func TestSearchRegex(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Regex search for pattern containing digits
	results, err := SearchRegex(f, `\d+`, false)
	if err != nil {
		t.Fatalf("SearchRegex failed: %v", err)
	}

	// Should find: "Test123"
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Value != "Test123" {
		t.Errorf("expected 'Test123', got %q", results[0].Value)
	}
}

func TestSearchMaxResults(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	ch, err := Search(context.Background(), f, "hello", SearchOptions{
		CaseInsensitive: true,
		MaxResults:      2,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	results, err := CollectSearchResults(ch)
	if err != nil {
		t.Fatalf("CollectSearchResults failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results (max), got %d", len(results))
	}
}

func TestSearchNoResults(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	results, err := SearchSimple(f, "nonexistent", false)
	if err != nil {
		t.Fatalf("SearchSimple failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchEmptyPattern(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	_, err = SearchSimple(f, "", false)
	if err == nil {
		t.Error("expected error for empty pattern")
	}
}

func TestSearchInvalidRegex(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	ch, err := Search(context.Background(), f, "[invalid", SearchOptions{Regex: true})
	if err == nil {
		// If channel was returned, drain it
		for range ch {
		}
		t.Error("expected error for invalid regex")
	}
}

func TestSearchResultFields(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	results, err := SearchInSheet(f, "Sheet1", "Hello World", false)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Sheet != "Sheet1" {
		t.Errorf("Sheet = %q, want 'Sheet1'", r.Sheet)
	}
	if r.Address != "A1" {
		t.Errorf("Address = %q, want 'A1'", r.Address)
	}
	if r.Value != "Hello World" {
		t.Errorf("Value = %q, want 'Hello World'", r.Value)
	}
	if r.Row != 1 {
		t.Errorf("Row = %d, want 1", r.Row)
	}
	if r.Col != 1 {
		t.Errorf("Col = %d, want 1", r.Col)
	}
}

func TestSearchNilFile(t *testing.T) {
	_, err := SearchSimple(nil, "test", false)
	if err == nil {
		t.Error("expected error for nil file")
	}
}

func TestSearchInvalidSheet(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	_, err = SearchInSheet(f, "NonExistentSheet", "test", false)
	if err == nil {
		t.Error("expected error for invalid sheet")
	}
}

func TestSearchRegexCaseInsensitive(t *testing.T) {
	path := createSearchTestFile(t)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Case-insensitive regex for "HELLO" (should match "Hello" and "hello")
	results, err := SearchRegex(f, "HELLO", true)
	if err != nil {
		t.Fatalf("SearchRegex failed: %v", err)
	}

	// Should find: "Hello World", "hello", "hello again", "Another Hello"
	if len(results) != 4 {
		t.Errorf("expected 4 results, got %d", len(results))
	}
}

func TestSearchWithEmptyCells(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	// Sparse data with empty cells
	if err := f.SetCellValue("Sheet1", "A1", ""); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "A2", "data"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "A3", ""); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	fRead, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer fRead.Close()

	results, err := SearchSimple(fRead, "data", false)
	if err != nil {
		t.Fatalf("SearchSimple failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}
