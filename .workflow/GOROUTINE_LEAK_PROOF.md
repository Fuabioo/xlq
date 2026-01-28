# Goroutine Leak - Empirical Proof

**Date**: 2026-01-23
**File**: `/home/fuabioo/Playground/excelize-mcp/internal/xlsx/stream.go`
**Functions**: `StreamRows`, `StreamRange`

## Claim

> StreamRows and StreamRange spawn goroutines that write to unbuffered channels. If the receiver stops reading early, the goroutine blocks forever.

## Verdict: **CONFIRMED**

The goroutine leak claim has been **empirically proven** through comprehensive testing.

## Evidence

### Test 1: StreamRows Leak Detection

```
Baseline goroutines: 2
After 10 abandoned channels: 12 goroutines
Leaked goroutines: 10
```

**Result**: Each abandoned channel leaked exactly 1 goroutine (100% leak rate)

### Test 2: StreamRange Leak Detection

```
Baseline goroutines: 12
After 10 abandoned channels: 22 goroutines
Leaked goroutines: 10
```

**Result**: Identical leak pattern - 100% leak rate

### Test 3: Proper Usage Control Test

```
Baseline goroutines: 2
After 10 full consumptions: 2 goroutines
Delta: 0
```

**Result**: When channels are fully consumed, NO leak occurs

### Test 4: Timing Analysis

```
After spawn: delta=1 (goroutine created)
After reading 1 row: delta=1 (goroutine still running)
After abandon: delta=1 (goroutine blocked)
After 1s wait + GC: delta=1 (goroutine STILL blocked)
```

**Result**: Goroutine remains blocked indefinitely, even after garbage collection

## Root Cause Analysis

### The Problem

In `stream.go` lines 29 and 94:

```go
ch := make(chan RowResult)  // UNBUFFERED CHANNEL
```

The goroutine spawned in lines 31-72 writes to this unbuffered channel:

```go
go func() {
    defer close(ch)
    defer rows.Close()

    for rows.Next() {
        // ... process row ...
        ch <- RowResult{Row: &Row{...}}  // BLOCKS if no receiver
    }
}()
```

### Why It Leaks

1. **Unbuffered channel**: Requires synchronous sender/receiver
2. **Sender blocks**: On `ch <- RowResult{...}` if receiver stopped reading
3. **No timeout**: Goroutine waits forever for a receiver that never comes
4. **No context**: No way to cancel the goroutine
5. **Deferred cleanup doesn't run**: `defer close(ch)` and `defer rows.Close()` never execute because the goroutine is blocked before reaching them

### The Blocking Point

When a receiver abandons the channel after reading N rows, the goroutine is blocked at:

```go
ch <- RowResult{Row: &Row{Number: N+1, ...}}  // Stuck here forever
```

The goroutine is waiting for someone to receive row N+1, but the receiver is gone.

## Real-World Impact

### Memory Leak

Each leaked goroutine holds:
- Goroutine stack (minimum 2KB, can grow)
- Row iterator state (`excelize.Rows`)
- File handle references
- Cell data for the row it's trying to send

### Accumulation Example

```
After 100 early-abandoned reads: ~100 leaked goroutines
After 1,000 early-abandoned reads: ~1,000 leaked goroutines
After 10,000 early-abandoned reads: ~10,000 leaked goroutines + file handles
```

### When Does This Happen?

Common scenarios:
1. **Error handling**: Code reads a few rows, encounters error, returns early
2. **Search/find**: Looking for first matching row, then stops
3. **Sampling**: Reading only first N rows for preview
4. **Context cancellation**: Request timeout/cancellation
5. **Limit operations**: Using `CollectRowsWithLimit` but channel still produces all rows

## Test Files

Created: `/home/fuabioo/Playground/excelize-mcp/internal/xlsx/goroutine_leak_test.go`

Tests:
- `TestGoroutineLeakStreamRows` - Proves StreamRows leaks
- `TestGoroutineLeakStreamRange` - Proves StreamRange leaks
- `TestGoroutineNoLeakFullConsumption` - Proves proper usage is safe
- `TestGoroutineLeakTiming` - Shows exact blocking behavior
- `BenchmarkGoroutineLeakMemory` - Demonstrates memory impact

## Running the Proof

```bash
# Prove the leak
go test -v -run TestGoroutineLeak ./internal/xlsx/

# Show proper usage is safe
go test -v -run TestGoroutineNoLeakFullConsumption ./internal/xlsx/

# See detailed timing
go test -v -run TestGoroutineLeakTiming ./internal/xlsx/
```

## Conclusion

The goroutine leak claim is **100% confirmed** through empirical testing. The current implementation of `StreamRows` and `StreamRange` will leak exactly one goroutine per abandoned channel, with no cleanup mechanism.

### Proof Summary

- 10/10 abandoned channels leaked goroutines (100% leak rate)
- Goroutines remain blocked indefinitely (tested up to 1+ seconds)
- Proper channel consumption shows zero leaks (control test passed)
- Leak persists through garbage collection (no automatic cleanup)

**Status**: Critical bug confirmed, requires fix before production use.
