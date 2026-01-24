package xlsx

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

// SearchOptions configures search behavior
type SearchOptions struct {
	CaseInsensitive bool   // Case-insensitive matching
	Sheet           string // Limit to specific sheet (empty = all sheets)
	Regex           bool   // Treat pattern as regex
	MaxResults      int    // Maximum results (0 = unlimited)
}

// SearchResultStream wraps a search result with potential error
type SearchResultStream struct {
	Result *SearchResult
	Err    error
}

// Search searches for cells matching a pattern across one or all sheets
func Search(ctx context.Context, f *excelize.File, pattern string, opts SearchOptions) (<-chan SearchResultStream, error) {
	if f == nil {
		return nil, fmt.Errorf("file handle is nil")
	}

	if pattern == "" {
		return nil, fmt.Errorf("search pattern cannot be empty")
	}

	// Compile regex or create literal matcher
	var matcher func(string) bool
	if opts.Regex {
		flags := ""
		if opts.CaseInsensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		matcher = re.MatchString
	} else {
		if opts.CaseInsensitive {
			patternLower := strings.ToLower(pattern)
			matcher = func(s string) bool {
				return strings.Contains(strings.ToLower(s), patternLower)
			}
		} else {
			matcher = func(s string) bool {
				return strings.Contains(s, pattern)
			}
		}
	}

	// Determine which sheets to search
	var sheetsToSearch []string
	if opts.Sheet != "" {
		sheetName, err := ResolveSheetName(f, opts.Sheet)
		if err != nil {
			return nil, err
		}
		sheetsToSearch = []string{sheetName}
	} else {
		sheets, err := GetSheets(f)
		if err != nil {
			return nil, err
		}
		sheetsToSearch = sheets
	}

	ch := make(chan SearchResultStream)

	go func() {
		defer close(ch)

		resultCount := 0
		for _, sheet := range sheetsToSearch {
			rows, err := f.Rows(sheet)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				case ch <- SearchResultStream{Err: fmt.Errorf("failed to read sheet %s: %w", sheet, err)}:
					return
				}
			}

			rowNum := 0
			for rows.Next() {
				// Check context before processing row
				select {
				case <-ctx.Done():
					rows.Close()
					return
				default:
				}

				rowNum++

				cols, err := rows.Columns()
				if err != nil {
					rows.Close()
					select {
					case <-ctx.Done():
						return
					case ch <- SearchResultStream{Err: fmt.Errorf("error at row %d: %w", rowNum, err)}:
						return
					}
				}

				for colIdx, val := range cols {
					if val != "" && matcher(val) {
						result := &SearchResult{
							Sheet:   sheet,
							Address: FormatCellAddress(colIdx+1, rowNum),
							Value:   val,
							Row:     rowNum,
							Col:     colIdx + 1,
						}
						select {
						case <-ctx.Done():
							rows.Close()
							return
						case ch <- SearchResultStream{Result: result}:
						}

						resultCount++
						if opts.MaxResults > 0 && resultCount >= opts.MaxResults {
							rows.Close()
							return
						}
					}
				}
			}

			if err := rows.Error(); err != nil {
				rows.Close()
				select {
				case <-ctx.Done():
					return
				case ch <- SearchResultStream{Err: fmt.Errorf("row iteration error in sheet %s: %w", sheet, err)}:
					return
				}
			}
			rows.Close()
		}
	}()

	return ch, nil
}

// CollectSearchResults collects all search results into a slice
func CollectSearchResults(ch <-chan SearchResultStream) ([]SearchResult, error) {
	var results []SearchResult
	for stream := range ch {
		if stream.Err != nil {
			return nil, stream.Err
		}
		if stream.Result != nil {
			results = append(results, *stream.Result)
		}
	}
	return results, nil
}

// SearchSimple is a convenience function for simple searches
func SearchSimple(f *excelize.File, pattern string, caseInsensitive bool) ([]SearchResult, error) {
	ch, err := Search(context.Background(), f, pattern, SearchOptions{
		CaseInsensitive: caseInsensitive,
		Regex:           false,
	})
	if err != nil {
		return nil, err
	}
	return CollectSearchResults(ch)
}

// SearchInSheet searches only within a specific sheet
func SearchInSheet(f *excelize.File, sheet, pattern string, caseInsensitive bool) ([]SearchResult, error) {
	ch, err := Search(context.Background(), f, pattern, SearchOptions{
		Sheet:           sheet,
		CaseInsensitive: caseInsensitive,
		Regex:           false,
	})
	if err != nil {
		return nil, err
	}
	return CollectSearchResults(ch)
}

// SearchRegex searches using a regex pattern
func SearchRegex(f *excelize.File, pattern string, caseInsensitive bool) ([]SearchResult, error) {
	ch, err := Search(context.Background(), f, pattern, SearchOptions{
		CaseInsensitive: caseInsensitive,
		Regex:           true,
	})
	if err != nil {
		return nil, err
	}
	return CollectSearchResults(ch)
}
