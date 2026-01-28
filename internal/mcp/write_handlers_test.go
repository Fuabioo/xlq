package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleWriteCell(t *testing.T) {
	// Create a temporary test directory in current working directory
	tmpDir := filepath.Join("testdata", "tmp_write_cell_test")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test_write_cell.xlsx")

	// Create initial file
	_, err := xlsx.CreateFile(testFile, "Sheet1", nil, nil, false)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create server
	srv := New()

	// Create a mock request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "write_cell",
			Arguments: map[string]any{
				"file":  testFile,
				"sheet": "Sheet1",
				"cell":  "A1",
				"value": "Hello, World!",
				"type":  "auto",
			},
		},
	}

	// Call handler
	result, err := srv.handleWriteCell(context.Background(), request)
	if err != nil {
		t.Fatalf("handleWriteCell returned error: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.IsError {
		t.Errorf("expected success, got error: %+v", result)
	}

	// Parse the JSON result
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content is not TextContent type")
	}

	var writeResult xlsx.WriteResult
	if err := json.Unmarshal([]byte(textContent.Text), &writeResult); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	// Verify the result
	if !writeResult.Success {
		t.Error("expected success to be true")
	}
	if writeResult.Cell != "A1" {
		t.Errorf("expected cell A1, got %s", writeResult.Cell)
	}
	if writeResult.NewValue != "Hello, World!" {
		t.Errorf("expected value 'Hello, World!', got %v", writeResult.NewValue)
	}

	// Verify the file was actually updated
	f, err := xlsx.OpenFile(testFile)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	cellValue, err := xlsx.GetCell(f, "Sheet1", "A1")
	if err != nil {
		t.Fatalf("failed to get cell: %v", err)
	}
	if cellValue.Value != "Hello, World!" {
		t.Errorf("expected cell value 'Hello, World!', got %s", cellValue.Value)
	}
}

func TestHandleAppendRows(t *testing.T) {
	// Create a temporary test directory in current working directory
	tmpDir := filepath.Join("testdata", "tmp_append_rows_test")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test_append_rows.xlsx")

	// Create initial file with headers
	headers := []string{"Name", "Age", "City"}
	initialRows := [][]any{
		{"Alice", 30, "New York"},
		{"Bob", 25, "San Francisco"},
	}
	_, err := xlsx.CreateFile(testFile, "Sheet1", headers, initialRows, false)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create server
	srv := New()

	// Create a mock request
	newRows := [][]any{
		{"Charlie", 35, "Los Angeles"},
		{"Diana", 28, "Seattle"},
	}
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "append_rows",
			Arguments: map[string]any{
				"file":  testFile,
				"sheet": "Sheet1",
				"rows":  newRows,
			},
		},
	}

	// Call handler
	result, err := srv.handleAppendRows(context.Background(), request)
	if err != nil {
		t.Fatalf("handleAppendRows returned error: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.IsError {
		t.Errorf("expected success, got error: %+v", result)
	}

	// Parse the JSON result
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content is not TextContent type")
	}

	var appendResult xlsx.AppendResult
	if err := json.Unmarshal([]byte(textContent.Text), &appendResult); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	// Verify the result
	if !appendResult.Success {
		t.Error("expected success to be true")
	}
	if appendResult.RowsAdded != 2 {
		t.Errorf("expected 2 rows added, got %d", appendResult.RowsAdded)
	}
	// Should start at row 4 (1 header + 2 initial rows + 1)
	if appendResult.StartingRow != 4 {
		t.Errorf("expected starting row 4, got %d", appendResult.StartingRow)
	}
	if appendResult.EndingRow != 5 {
		t.Errorf("expected ending row 5, got %d", appendResult.EndingRow)
	}
}

func TestHandleCreateFile(t *testing.T) {
	// Create a temporary test directory in current working directory
	tmpDir := filepath.Join("testdata", "tmp_create_file_test")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test_create_file.xlsx")

	// Create server
	srv := New()

	// Create a mock request
	headers := []string{"Product", "Price", "Quantity"}
	rows := [][]any{
		{"Widget", 19.99, 100},
		{"Gadget", 29.99, 50},
	}
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "create_file",
			Arguments: map[string]any{
				"file":       testFile,
				"sheet_name": "Inventory",
				"overwrite":  false,
				"headers":    headers,
				"rows":       rows,
			},
		},
	}

	// Call handler
	result, err := srv.handleCreateFile(context.Background(), request)
	if err != nil {
		t.Fatalf("handleCreateFile returned error: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.IsError {
		t.Errorf("expected success, got error: %+v", result)
	}

	// Parse the JSON result
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content is not TextContent type")
	}

	var createResult xlsx.CreateFileResult
	if err := json.Unmarshal([]byte(textContent.Text), &createResult); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	// Verify the result
	if !createResult.Success {
		t.Error("expected success to be true")
	}
	if createResult.SheetName != "Inventory" {
		t.Errorf("expected sheet name 'Inventory', got %s", createResult.SheetName)
	}
	if createResult.RowsWritten != 3 { // 1 header + 2 data rows
		t.Errorf("expected 3 rows written, got %d", createResult.RowsWritten)
	}

	// Verify the file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("file was not created")
	}

	// Verify file contents
	f, err := xlsx.OpenFile(testFile)
	if err != nil {
		t.Fatalf("failed to open created file: %v", err)
	}
	defer f.Close()

	// Check sheet info
	info, err := xlsx.GetSheetInfo(f, "Inventory")
	if err != nil {
		t.Fatalf("failed to get sheet info: %v", err)
	}
	if info.Rows != 3 {
		t.Errorf("expected 3 rows, got %d", info.Rows)
	}
	if info.Cols != 3 {
		t.Errorf("expected 3 columns, got %d", info.Cols)
	}
}

func TestHandleWriteCellErrors(t *testing.T) {
	srv := New()

	tests := []struct {
		name    string
		request mcp.CallToolRequest
		errText string
	}{
		{
			name: "File outside allowed path",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "write_cell",
					Arguments: map[string]any{
						"file":  "/tmp/test.xlsx",
						"sheet": "Sheet1",
						"cell":  "A1",
						"value": "test",
					},
				},
			},
			errText: "denied",
		},
		{
			name: "Blocked path (.env)",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "write_cell",
					Arguments: map[string]any{
						"file":  ".env",
						"sheet": "Sheet1",
						"cell":  "A1",
						"value": "test",
					},
				},
			},
			errText: "denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := srv.handleWriteCell(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("handler returned error (should return error in result): %v", err)
			}
			if !result.IsError {
				t.Error("expected error result")
			}
		})
	}
}

func TestHandleAppendRowsErrors(t *testing.T) {
	srv := New()

	// Create a temporary test directory in current working directory
	tmpDir := filepath.Join("testdata", "tmp_append_errors_test")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		request mcp.CallToolRequest
		errText string
	}{
		{
			name: "Empty rows",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "append_rows",
					Arguments: map[string]any{
						"file":  filepath.Join(tmpDir, "test.xlsx"),
						"sheet": "Sheet1",
						"rows":  [][]any{},
					},
				},
			},
			errText: "no rows provided",
		},
		{
			name: "Too many rows",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "append_rows",
					Arguments: map[string]any{
						"file":  filepath.Join(tmpDir, "test.xlsx"),
						"sheet": "Sheet1",
						"rows":  make([][]any, 1001), // Exceeds MaxAppendRows
					},
				},
			},
			errText: "too many rows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := srv.handleAppendRows(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("handler returned error (should return error in result): %v", err)
			}
			if !result.IsError {
				t.Error("expected error result")
			}
		})
	}
}
