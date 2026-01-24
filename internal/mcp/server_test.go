package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestNewServer(t *testing.T) {
	srv := New()
	if srv == nil {
		t.Fatal("New() returned nil")
		return
	}
	if srv.mcpServer == nil {
		t.Error("mcpServer is nil")
	}
}

func TestJsonResult(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		shouldErr bool
	}{
		{
			name:      "simple string slice",
			input:     []string{"a", "b", "c"},
			shouldErr: false,
		},
		{
			name:      "map",
			input:     map[string]string{"key": "value"},
			shouldErr: false,
		},
		{
			name:      "nil",
			input:     nil,
			shouldErr: false,
		},
		{
			name:      "struct",
			input:     struct{ Name string }{"test"},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jsonResult(tt.input)
			if tt.shouldErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("result is nil")
				}
			}
		})
	}
}

func TestJsonResultOutputLimit(t *testing.T) {
	// Create a large string that exceeds MaxOutputBytes (5MB)
	largeData := strings.Repeat("x", MaxOutputBytes+1000)

	result, err := jsonResult(largeData)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	// Check if the result is an error message about size
	if !result.IsError {
		t.Error("expected IsError to be true for large output")
	}
}

func TestJsonResultWithMetadata(t *testing.T) {
	tests := []struct {
		name         string
		data         interface{}
		rowsReturned int
		truncated    bool
		limit        int
	}{
		{
			name:         "basic metadata",
			data:         [][]string{{"a", "b"}, {"c", "d"}},
			rowsReturned: 2,
			truncated:    false,
			limit:        1000,
		},
		{
			name:         "truncated result",
			data:         [][]string{{"a", "b"}},
			rowsReturned: 1,
			truncated:    true,
			limit:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jsonResultWithMetadata(tt.data, tt.rowsReturned, tt.truncated, tt.limit)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("result is nil")
				return
			}

			// Verify the result contains our expected structure
			if len(result.Content) == 0 {
				t.Fatal("result has no content")
			}

			// Extract text from the content by type assertion to TextContent
			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("content is not TextContent type, got %T", result.Content[0])
			}

			text := textContent.Text
			if !strings.Contains(text, "metadata") {
				t.Error("result doesn't contain metadata")
			}
			if !strings.Contains(text, "rows_returned") {
				t.Error("result doesn't contain rows_returned")
			}
			if !strings.Contains(text, "truncated") {
				t.Error("result doesn't contain truncated")
			}
			if !strings.Contains(text, "limit") {
				t.Error("result doesn't contain limit")
			}

			// Also verify we can parse it as JSON
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(text), &parsed); err != nil {
				t.Errorf("result is not valid JSON: %v", err)
			}
		})
	}
}

func TestLimitsConstants(t *testing.T) {
	// Verify that constants are sensible
	if DefaultRowLimit <= 0 {
		t.Error("DefaultRowLimit must be positive")
	}
	if MaxRowLimit <= DefaultRowLimit {
		t.Error("MaxRowLimit must be greater than DefaultRowLimit")
	}
	if DefaultHeadRows <= 0 {
		t.Error("DefaultHeadRows must be positive")
	}
	if MaxHeadRows <= DefaultHeadRows {
		t.Error("MaxHeadRows must be greater than DefaultHeadRows")
	}
	if DefaultTailRows <= 0 {
		t.Error("DefaultTailRows must be positive")
	}
	if MaxTailRows <= DefaultTailRows {
		t.Error("MaxTailRows must be greater than DefaultTailRows")
	}
	if DefaultSearchResults <= 0 {
		t.Error("DefaultSearchResults must be positive")
	}
	if MaxSearchResults <= DefaultSearchResults {
		t.Error("MaxSearchResults must be greater than DefaultSearchResults")
	}
	if MaxOutputBytes <= 0 {
		t.Error("MaxOutputBytes must be positive")
	}

	// Verify actual values match requirements
	if DefaultRowLimit != 1000 {
		t.Errorf("DefaultRowLimit should be 1000, got %d", DefaultRowLimit)
	}
	if MaxRowLimit != 10000 {
		t.Errorf("MaxRowLimit should be 10000, got %d", MaxRowLimit)
	}
	if DefaultHeadRows != 10 {
		t.Errorf("DefaultHeadRows should be 10, got %d", DefaultHeadRows)
	}
	if MaxHeadRows != 5000 {
		t.Errorf("MaxHeadRows should be 5000, got %d", MaxHeadRows)
	}
	if DefaultTailRows != 10 {
		t.Errorf("DefaultTailRows should be 10, got %d", DefaultTailRows)
	}
	if MaxTailRows != 5000 {
		t.Errorf("MaxTailRows should be 5000, got %d", MaxTailRows)
	}
	if DefaultSearchResults != 100 {
		t.Errorf("DefaultSearchResults should be 100, got %d", DefaultSearchResults)
	}
	if MaxSearchResults != 1000 {
		t.Errorf("MaxSearchResults should be 1000, got %d", MaxSearchResults)
	}
	if MaxOutputBytes != 5*1024*1024 {
		t.Errorf("MaxOutputBytes should be 5MB, got %d", MaxOutputBytes)
	}
}
