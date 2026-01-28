package xlsx

import (
	"testing"
)

// TestStreamTailCellAllocationReduction verifies that Cell structs
// are only allocated for the final N rows, not all rows.
//
// The key test: varying tail size should change allocations,
// but varying total row count should not (much, only excelize overhead).
func TestStreamTailCellAllocationReduction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cell allocation test in short mode")
	}

	// Create test file with 10,000 rows
	path := createLargeTestFile(t, 10000)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Test different tail sizes on SAME file
	// If Cell allocation is working correctly, allocations should scale with tail size
	allocsTail10 := testing.AllocsPerRun(5, func() {
		rows, err := StreamTail(f, "Sheet1", 10)
		if err != nil {
			t.Fatalf("StreamTail failed: %v", err)
		}
		_ = len(rows)
	})

	allocsTail100 := testing.AllocsPerRun(5, func() {
		rows, err := StreamTail(f, "Sheet1", 100)
		if err != nil {
			t.Fatalf("StreamTail failed: %v", err)
		}
		_ = len(rows)
	})

	t.Logf("Allocations for tail=10 on 10k rows: %.0f", allocsTail10)
	t.Logf("Allocations for tail=100 on 10k rows: %.0f", allocsTail100)

	// The difference should be roughly (100-10) rows × ~4 allocs/row = ~360 allocs
	// (1 for []Cell slice + 3 for Cell structs per row with 3 columns)
	diff := allocsTail100 - allocsTail10
	t.Logf("Difference: %.0f allocations", diff)

	// Expected: ~360 allocs for 90 additional rows × 4 allocs/row
	// Actual acceptable range: 100-2000 (some variance is OK)
	if diff < 100 || diff > 2000 {
		t.Errorf("Allocation difference unexpected: %.0f (expected 100-2000)", diff)
		t.Errorf("This suggests Cell allocation is NOT scaling with tail size")
	} else {
		t.Logf("SUCCESS: Allocation difference %.0f is in expected range for 90 additional tail rows", diff)
	}

	// Additional sanity check: total allocations should be reasonable
	// excelize baseline is ~660k for 10k rows
	// With 100 tail rows × 4 allocs = 400 additional
	// Total should be around 660k + 400 = ~660.4k
	if allocsTail100 > 700000 {
		t.Errorf("Total allocations too high: %.0f (expected < 700000)", allocsTail100)
	}
}
