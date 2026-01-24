package xlsx

import (
	"runtime"
	"testing"
	"time"
)

// TestGoroutineLeakStreamRows tests if StreamRows leaks goroutines when receiver stops early.
//
// The claim: StreamRows spawns a goroutine that writes to an unbuffered channel.
// If the receiver stops reading early (abandons the channel), the goroutine blocks
// forever on the channel send operation.
//
// This test proves or disproves this claim empirically.
func TestGoroutineLeakStreamRows(t *testing.T) {
	path := createLargeTestFile(t, 1000) // Create file with 1000 rows

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Force garbage collection and get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baselineGoroutines)

	// Number of leak attempts
	const leakAttempts = 10

	for range leakAttempts {
		// Start streaming (this spawns a goroutine)
		ch, err := StreamRows(f, "Sheet1", 1, 1000)
		if err != nil {
			t.Fatalf("StreamRows failed: %v", err)
		}

		// Read only 1 row, then abandon the channel
		result := <-ch
		if result.Err != nil {
			t.Fatalf("Error reading first row: %v", result.Err)
		}
		if result.Row == nil {
			t.Fatal("Expected row, got nil")
		}

		// Abandon the channel (don't read remaining 999 rows)
		// The goroutine is trying to send row #2 to the unbuffered channel
		// but nobody is receiving, so it should block forever
		_ = ch // Don't read any more
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

// TestGoroutineLeakStreamRange tests if StreamRange leaks goroutines when receiver stops early.
func TestGoroutineLeakStreamRange(t *testing.T) {
	path := createLargeTestFile(t, 1000)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Force garbage collection and get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baselineGoroutines)

	const leakAttempts = 10

	for range leakAttempts {
		// Stream a large range
		ch, err := StreamRange(f, "Sheet1", "A1:C1000")
		if err != nil {
			t.Fatalf("StreamRange failed: %v", err)
		}

		// Read only 1 row, then abandon
		result := <-ch
		if result.Err != nil {
			t.Fatalf("Error reading first row: %v", result.Err)
		}
		if result.Row == nil {
			t.Fatal("Expected row, got nil")
		}

		// Abandon the channel
		_ = ch
	}

	// Give goroutines a chance to exit
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	afterLeakGoroutines := runtime.NumGoroutine()
	t.Logf("After leak attempts: %d goroutines", afterLeakGoroutines)

	leakedGoroutines := afterLeakGoroutines - baselineGoroutines
	t.Logf("Leaked goroutines: %d", leakedGoroutines)

	if leakedGoroutines >= leakAttempts-2 {
		t.Errorf("CONFIRMED LEAK: %d goroutines leaked after %d abandoned channels (expected ~0)",
			leakedGoroutines, leakAttempts)
		t.Error("The goroutines are blocked forever on unbuffered channel sends")
	} else {
		t.Logf("NO LEAK: Only %d goroutines remain after %d abandoned channels",
			leakedGoroutines, leakAttempts)
	}
}

// TestGoroutineNoLeakFullConsumption verifies no leak when channel is fully consumed.
// This is the control test - proper usage should not leak.
func TestGoroutineNoLeakFullConsumption(t *testing.T) {
	path := createLargeTestFile(t, 100)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baselineGoroutines)

	const iterations = 10

	for range iterations {
		ch, err := StreamRows(f, "Sheet1", 1, 100)
		if err != nil {
			t.Fatalf("StreamRows failed: %v", err)
		}

		// Fully consume the channel (proper usage)
		_, err = CollectRows(ch)
		if err != nil {
			t.Fatalf("CollectRows failed: %v", err)
		}
	}

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	afterGoroutines := runtime.NumGoroutine()
	t.Logf("After full consumption: %d goroutines", afterGoroutines)

	leaked := afterGoroutines - baselineGoroutines
	t.Logf("Goroutines delta: %d", leaked)

	// With full consumption, we should not leak goroutines
	if leaked > 2 { // Allow small tolerance for test framework
		t.Errorf("Unexpected leak even with full consumption: %d goroutines remain", leaked)
	} else {
		t.Logf("PASS: No leak with proper usage (delta: %d)", leaked)
	}
}

// TestGoroutineLeakTiming demonstrates the exact moment the goroutine blocks.
// This test provides detailed insight into the blocking behavior.
func TestGoroutineLeakTiming(t *testing.T) {
	path := createLargeTestFile(t, 10)

	f, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	ch, err := StreamRows(f, "Sheet1", 1, 10)
	if err != nil {
		t.Fatalf("StreamRows failed: %v", err)
	}

	// Immediately after StreamRows, goroutine is spawned
	time.Sleep(50 * time.Millisecond)
	afterSpawn := runtime.NumGoroutine()
	t.Logf("After spawn: baseline=%d, current=%d, delta=%d",
		baseline, afterSpawn, afterSpawn-baseline)

	// Read one row
	<-ch
	time.Sleep(50 * time.Millisecond)
	afterFirstRead := runtime.NumGoroutine()
	t.Logf("After reading 1 row: %d goroutines (delta: %d)",
		afterFirstRead, afterFirstRead-baseline)

	// Abandon channel - goroutine tries to send row 2 and blocks
	time.Sleep(200 * time.Millisecond)
	afterAbandon := runtime.NumGoroutine()
	t.Logf("After abandon: %d goroutines (delta: %d)",
		afterAbandon, afterAbandon-baseline)

	// Wait longer to see if it cleans up
	time.Sleep(1 * time.Second)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	afterWait := runtime.NumGoroutine()
	t.Logf("After 1s wait + GC: %d goroutines (delta: %d)",
		afterWait, afterWait-baseline)

	if afterWait-baseline >= 1 {
		t.Logf("LEAK CONFIRMED: Goroutine did not exit after 1+ seconds")
		t.Logf("The goroutine is blocked on: ch <- RowResult{Row: &Row{Number: 2, ...}}")
		t.Logf("Root cause: Unbuffered channel with no receiver")
	} else {
		t.Logf("No leak detected in this test")
	}
}

// BenchmarkGoroutineLeakMemory checks if leaked goroutines consume memory over time.
// This benchmark demonstrates the practical impact of the leak.
func BenchmarkGoroutineLeakMemory(b *testing.B) {
	path := createLargeTestFile(&testing.T{}, 1000)

	f, err := OpenFile(path)
	if err != nil {
		b.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ch, err := StreamRows(f, "Sheet1", 1, 1000)
		if err != nil {
			b.Fatalf("StreamRows failed: %v", err)
		}

		// Read only 1 row then abandon (leak pattern)
		<-ch
		// Abandoned - goroutine blocked forever
	}

	// Report goroutine count
	b.StopTimer()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	b.Logf("Final goroutine count: %d (leaked: ~%d)", runtime.NumGoroutine(), b.N)
}
