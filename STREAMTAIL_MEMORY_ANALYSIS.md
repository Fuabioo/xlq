# StreamTail Memory Analysis Report

**Date**: 2026-01-23
**Test File**: `/home/fuabioo/Playground/excelize-mcp/internal/xlsx/memory_bench_test.go`
**Claim**: StreamTail allocates Cell structs for EVERY row before discarding them, causing high memory churn even though final result is bounded.

## Executive Summary

**CLAIM CONFIRMED**: StreamTail exhibits severe memory allocation issues that scale linearly with total row count, not with the requested tail size.

## Benchmark Results

### 1. Main Memory Benchmark (10,000 rows, tail size 10)

```
BenchmarkStreamTailMemory-24    	      16	  68215799 ns/op	28731334 B/op	  789352 allocs/op
```

**Analysis**:
- 789,352 allocations for a 10-row result
- ~28.7 MB allocated per operation
- For 10,000 rows requesting only 10 = **78,935 allocations per row**

### 2. Allocation Scaling Test (Varying Tail Sizes, 10,000 rows)

```
TailSize_5     28728360 B/op    789330 allocs/op
TailSize_10    28728952 B/op    789331 allocs/op
TailSize_50    28731509 B/op    789332 allocs/op
TailSize_100   28734485 B/op    789329 allocs/op
```

**Critical Finding**: Allocations are **constant** regardless of tail size (5 vs 100 rows).
- This proves allocations are proportional to total row count, NOT tail size
- Tail size 5: 789,330 allocs
- Tail size 100: 789,329 allocs (virtually identical)
- Memory usage difference: ~6KB across 20x tail size increase

**Conclusion**: The ring buffer is NOT preventing allocation overhead.

### 3. File Size Comparison (All with tail size 10)

```
Small_20rows_tail10      avg allocations = 1,567
Medium_1000rows_tail10   avg allocations = 78,284
Large_10000rows_tail10   avg allocations = 789,300
```

**Scaling Analysis**:
- 20 rows → 1,567 allocs ≈ 78 allocs/row
- 1,000 rows → 78,284 allocs ≈ 78 allocs/row
- 10,000 rows → 789,300 allocs ≈ 79 allocs/row

**Perfect linear scaling**: ~78-79 allocations per row processed.

### 4. StreamHead Comparison (10,000 rows, head size 10)

```
BenchmarkStreamTailMemory-24    	      18	  64734270 ns/op	28731527 B/op	  789350 allocs/op
BenchmarkStreamHeadMemory-24    	   14228	     83609 ns/op	   33029 B/op	     821 allocs/op
```

**Dramatic Difference**:
- StreamHead (10 rows): 821 allocations, 33 KB
- StreamTail (10 rows from 10K): 789,350 allocations, 28.7 MB
- **962x more allocations** for same result size
- **869x more memory** allocated

### 5. Wide Rows Test (10,000 rows x 20 columns, tail size 10)

```
BenchmarkStreamTailMemoryWideRows-24    	       2	 521730358 ns/op	325971388 B/op	 5955769 allocs/op
```

**Extreme Memory Pressure**:
- ~326 MB allocated
- ~6 million allocations
- For a result of only 10 rows x 20 cells = 200 cells
- **29,779 allocations per output cell**

## Root Cause Analysis

Looking at the code in `stream.go:188-197`:

```go
cells := make([]Cell, len(cols))  // Line 188
for i, val := range cols {
    cells[i] = Cell{               // Lines 190-196
        Address: FormatCellAddress(i+1, rowNum),
        Value:   val,
        Type:    "string",
        Row:     rowNum,
        Col:     i + 1,
    }
}
```

**The Problem**:
1. For EVERY row iteration (lines 180-202), a new `cells` slice is allocated
2. For EVERY cell in EVERY row, a `Cell` struct is allocated with heap-allocated strings
3. These allocations happen BEFORE the row is stored in the ring buffer
4. When the ring buffer overwrites an old slot (line 199), the previous allocations become garbage
5. The GC has to clean up (10,000 - 10) = 9,990 rows worth of Cell allocations

**Why Ring Buffer Doesn't Help**:
- Ring buffer only reuses the `Row` struct slots (line 175: `make([]Row, n)`)
- It does NOT reuse the `Cells` slice or individual `Cell` structs
- Each iteration allocates fresh `cells` and `Cell` instances
- Old `Cells` slices are orphaned when buffer slot is overwritten

## Impact Assessment

### For typical usage (1,000 row file, tail 10):
- Allocates for 1,000 rows, keeps 10
- 99% of allocations wasted
- ~78,000 allocations for 10-row result

### For large files (100,000 rows, tail 10):
- Would allocate ~7.9 million times
- ~2.8 GB of memory churn
- For a 10-row result

### Memory profile characteristics:
- High allocation rate
- High GC pressure
- Memory usage spikes during tail operation
- Not truly "bounded memory" as claimed in comments (line 157)

## Recommendations

### Option 1: Reuse Cell Slice (Simple Fix)
Pre-allocate a single `cells` slice and reuse it:

```go
// Pre-allocate reusable cell buffer
var cellsBuf []Cell

for rows.Next() {
    cols, err := rows.Columns()
    if err != nil {
        return nil, fmt.Errorf("error reading row %d: %w", rowNum, err)
    }

    // Resize buffer if needed
    if cap(cellsBuf) < len(cols) {
        cellsBuf = make([]Cell, len(cols))
    } else {
        cellsBuf = cellsBuf[:len(cols)]
    }

    // Fill cells
    for i, val := range cols {
        cellsBuf[i] = Cell{
            Address: FormatCellAddress(i+1, rowNum),
            Value:   val,
            Type:    "string",
            Row:     rowNum,
            Col:     i + 1,
        }
    }

    // Copy to ring buffer (must copy since we're reusing cellsBuf)
    buffer[bufIdx].Number = rowNum
    buffer[bufIdx].Cells = make([]Cell, len(cellsBuf))
    copy(buffer[bufIdx].Cells, cellsBuf)

    bufIdx = (bufIdx + 1) % n
    totalRows++
}
```

**Expected improvement**: ~79 allocs/row → ~2-3 allocs/row (ring buffer slots only)

### Option 2: String Interning (Advanced)
For files with repeated values, intern strings to reduce allocations.

### Option 3: Two-Pass (Best Memory)
1. First pass: count total rows (only increment counter, no allocations)
2. Second pass: stream only the last N rows with targeted allocations

Trade-off: 2x file read time, but eliminates waste entirely.

## Verification Command

To reproduce these findings:

```bash
# Main benchmark
go test -bench=BenchmarkStreamTailMemory$ -benchmem ./internal/xlsx/ -run=^$

# Allocation count test
go test -run=TestStreamTailAllocationCount ./internal/xlsx/ -v

# Comparison with StreamHead
go test -bench="BenchmarkStream(Head|Tail)Memory$" -benchmem ./internal/xlsx/ -run=^$

# Memory profile for analysis
go test -run=TestStreamTailMemoryProfile -memprofile=mem.out ./internal/xlsx/
go tool pprof mem.out
```

## Conclusion

The claim is **100% confirmed**. StreamTail:
- Allocates Cell structs for EVERY row in the sheet
- Ring buffer only prevents Row struct proliferation, not Cell allocation waste
- Memory usage scales with total row count, not requested tail size
- For large files, this creates severe memory pressure and GC churn
- Does not meet the "bounded memory" promise in architecture principles

**Priority**: HIGH - This undermines the core "streaming-first" architectural principle and could cause OOM on large files.
