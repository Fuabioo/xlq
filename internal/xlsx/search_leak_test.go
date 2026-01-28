package xlsx

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// TestGoroutineLeakSearch tests if Search leaks goroutines when receiver stops early.
func TestGoroutineLeakSearch(t *testing.T) {
	// Use existing createSearchTestFile from search_test.go
	dir := t.TempDir()
	path := dir + "/search_leak.xlsx"

	// Create test file manually to avoid import issues
	f := excelize.NewFile()
	defer f.Close()

	// Create test data with "hello" in various cells
	if err := f.SetCellValue("Sheet1", "A1", "hello world"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "A2", "Hello there"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "B1", "data"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}
	if err := f.SetCellValue("Sheet1", "B2", "HELLO"); err != nil {
		t.Fatalf("failed to set cell value: %v", err)
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to save test file: %v", err)
	}
	f.Close()

	// Now open and test
	fRead, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer fRead.Close()

	// Force garbage collection and get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baselineGoroutines)

	// Number of leak attempts
	const leakAttempts = 10

	for range leakAttempts {
		// Create a cancelable context
		ctx, cancel := context.WithCancel(context.Background())

		// Start searching (this spawns a goroutine)
		// Use a pattern that will match many cells
		ch, err := Search(ctx, fRead, "hello", SearchOptions{
			CaseInsensitive: true,
			MaxResults:      100,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Read only 1 result, then abandon the channel
		result := <-ch
		if result.Err != nil {
			t.Fatalf("Error reading first result: %v", result.Err)
		}
		if result.Result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Cancel the context to signal the goroutine to exit
		cancel()

		// Abandon the channel (don't read remaining results)
		_ = ch
	}

	// Give goroutines a chance to exit (if they could)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	afterLeakGoroutines := runtime.NumGoroutine()
	t.Logf("After leak attempts: %d goroutines", afterLeakGoroutines)

	leakedGoroutines := afterLeakGoroutines - baselineGoroutines
	t.Logf("Leaked goroutines: %d", leakedGoroutines)

	// If goroutines leaked, we expect approximately leakAttempts new goroutines
	// We allow some tolerance for test framework goroutines
	if leakedGoroutines >= leakAttempts-2 {
		t.Errorf("CONFIRMED LEAK: %d goroutines leaked after %d abandoned channels (expected ~0)",
			leakedGoroutines, leakAttempts)
		t.Error("The goroutines are blocked forever on unbuffered channel sends")
	} else {
		t.Logf("NO LEAK: Only %d goroutines remain after %d abandoned channels",
			leakedGoroutines, leakAttempts)
	}
}
