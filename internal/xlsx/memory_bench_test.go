package xlsx

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// BenchmarkStreamTailMemory measures allocation behavior of StreamTail
// to verify the claim that it allocates Cell structs for EVERY row
// even though only the last N rows are kept in the ring buffer.
//
// Expected behavior (ring buffer working correctly):
// - Should allocate ~10 rows worth of Cell structs (requested tail size)
// - Total allocations should be bounded regardless of total row count
//
// Bad behavior (if claim is true):
// - Would allocate 10,000 rows worth of Cell structs
// - Memory churn proportional to total row count
func BenchmarkStreamTailMemory(b *testing.B) {
	// Create test file with 10,000 rows
	path := createLargeTestFile(&testing.T{}, 10000)

	f, err := OpenFile(path)
	if err != nil {
		b.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rows, err := StreamTail(f, "Sheet1", 10)
		if err != nil {
			b.Fatalf("StreamTail failed: %v", err)
		}

		// Verify we got the correct tail
		if len(rows) != 10 {
			b.Errorf("expected 10 rows, got %d", len(rows))
		}

		// Verify last row is actually row 10000
		if len(rows) > 0 && rows[len(rows)-1].Number != 10000 {
			b.Errorf("expected last row number 10000, got %d", rows[len(rows)-1].Number)
		}
	}
}

// BenchmarkStreamTailMemorySmall provides a baseline with a small file
// to compare allocation patterns
func BenchmarkStreamTailMemorySmall(b *testing.B) {
	// Create test file with only 20 rows
	path := createLargeTestFile(&testing.T{}, 20)

	f, err := OpenFile(path)
	if err != nil {
		b.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rows, err := StreamTail(f, "Sheet1", 10)
		if err != nil {
			b.Fatalf("StreamTail failed: %v", err)
		}

		if len(rows) != 10 {
			b.Errorf("expected 10 rows, got %d", len(rows))
		}
	}
}

// BenchmarkStreamTailMemoryVaryingTailSize tests different tail sizes
// to observe allocation scaling
func BenchmarkStreamTailMemoryVaryingTailSize(b *testing.B) {
	path := createLargeTestFile(&testing.T{}, 10000)

	f, err := OpenFile(path)
	if err != nil {
		b.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Test different tail sizes to see if allocations scale with tail size (good)
	// or with total row count (bad)
	tailSizes := []int{5, 10, 50, 100}

	for _, size := range tailSizes {
		b.Run(fmt.Sprintf("TailSize_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rows, err := StreamTail(f, "Sheet1", size)
				if err != nil {
					b.Fatalf("StreamTail failed: %v", err)
				}

				if len(rows) != size {
					b.Errorf("expected %d rows, got %d", size, len(rows))
				}
			}
		})
	}
}

// BenchmarkStreamHeadMemory provides a comparison point
// StreamHead should only allocate for the requested rows
func BenchmarkStreamHeadMemory(b *testing.B) {
	path := createLargeTestFile(&testing.T{}, 10000)

	f, err := OpenFile(path)
	if err != nil {
		b.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ch, err := StreamHead(context.Background(), f, "Sheet1", 10)
		if err != nil {
			b.Fatalf("StreamHead failed: %v", err)
		}

		rows, err := CollectRows(ch)
		if err != nil {
			b.Fatalf("CollectRows failed: %v", err)
		}

		if len(rows) != 10 {
			b.Errorf("expected 10 rows, got %d", len(rows))
		}
	}
}

// TestStreamTailMemoryProfile creates a memory profile test
// that can be run with: go test -run=TestStreamTailMemoryProfile -memprofile=mem.out
func TestStreamTailMemoryProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory profile test in short mode")
	}

	// Create large file to exaggerate memory behavior
	path := createLargeTestFile(t, 10000)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Run multiple times to accumulate allocations in profile
	for i := 0; i < 10; i++ {
		rows, err := StreamTail(f, "Sheet1", 10)
		if err != nil {
			t.Fatalf("StreamTail failed: %v", err)
		}

		// Verify correctness
		if len(rows) != 10 {
			t.Errorf("expected 10 rows, got %d", len(rows))
		}

		if rows[len(rows)-1].Number != 10000 {
			t.Errorf("expected last row 10000, got %d", rows[len(rows)-1].Number)
		}
	}

	t.Logf("Memory profile test completed. Analyze with: go tool pprof mem.out")
}

// TestStreamTailAllocationCount is a unit test that checks allocation counts
// using testing.AllocsPerRun
func TestStreamTailAllocationCount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping allocation count test in short mode")
	}

	// Create test files with different sizes
	testCases := []struct {
		name      string
		totalRows int
		tailSize  int
	}{
		{"Small_20rows_tail10", 20, 10},
		{"Medium_1000rows_tail10", 1000, 10},
		{"Large_10000rows_tail10", 10000, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := createLargeTestFile(t, tc.totalRows)

			f, err := OpenFile(path)
			if err != nil {
				t.Fatalf("OpenFile failed: %v", err)
			}
			defer f.Close()

			// Measure allocations per run
			avgAllocs := testing.AllocsPerRun(5, func() {
				rows, err := StreamTail(f, "Sheet1", tc.tailSize)
				if err != nil {
					t.Fatalf("StreamTail failed: %v", err)
				}
				_ = rows // Use the result to prevent optimization
			})

			t.Logf("%s: avg allocations = %.0f", tc.name, avgAllocs)

			// The claim is that StreamTail allocates for EVERY row
			// If true, allocations would scale with totalRows
			// If false (ring buffer working), allocations should be relatively constant

			// This is informational - we'll compare across test cases
			// to see if allocations scale with totalRows (bad) or remain bounded (good)
		})
	}
}

// createTestFileWithWideRows creates a file with many columns to test Cell allocation
func createTestFileWithWideRows(t *testing.T, numRows, numCols int) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "wide.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	// Create rows with many columns
	for row := 1; row <= numRows; row++ {
		for col := 1; col <= numCols; col++ {
			value := fmt.Sprintf("R%dC%d", row, col)
			addr := FormatCellAddress(col, row)
			if err := f.SetCellValue("Sheet1", addr, value); err != nil {
				t.Fatalf("failed to set cell %s: %v", addr, err)
			}
		}
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	return path
}

// BenchmarkStreamTailMemoryWideRows tests memory with wide rows (many cells per row)
// This should exaggerate the allocation issue if it exists
func BenchmarkStreamTailMemoryWideRows(b *testing.B) {
	// 10,000 rows x 20 columns = 200,000 cells
	// If all are allocated, that's significant memory
	path := createTestFileWithWideRows(&testing.T{}, 10000, 20)

	f, err := OpenFile(path)
	if err != nil {
		b.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rows, err := StreamTail(f, "Sheet1", 10)
		if err != nil {
			b.Fatalf("StreamTail failed: %v", err)
		}

		if len(rows) != 10 {
			b.Errorf("expected 10 rows, got %d", len(rows))
		}

		// Each row should have 20 cells
		if len(rows) > 0 && len(rows[0].Cells) != 20 {
			b.Errorf("expected 20 cells per row, got %d", len(rows[0].Cells))
		}
	}
}
