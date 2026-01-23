package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// createTestFile creates a simple xlsx file for testing
func createTestFile(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.xlsx")

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	sheet := "Sheet1"

	// Add headers
	headers := []string{"Name", "Age", "City"}
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			t.Fatal(err)
		}
	}

	// Add data
	data := [][]interface{}{
		{"Alice", 30, "New York"},
		{"Bob", 25, "Boston"},
		{"Charlie", 35, "Chicago"},
	}

	for rowIdx, row := range data {
		for colIdx, val := range row {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.SetCellValue(sheet, cell, val); err != nil {
				t.Fatal(err)
			}
		}
	}

	if err := f.SaveAs(testFile); err != nil {
		t.Fatal(err)
	}

	return testFile
}

// captureOutput captures stdout while executing a function
func captureOutput(t *testing.T, f func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	// Execute function
	f()

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	return buf.String()
}

func TestSheetsCommand(t *testing.T) {
	testFile := createTestFile(t)

	output := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"sheets", testFile})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("sheets command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Sheet1") {
		t.Errorf("Expected output to contain 'Sheet1', got: %s", output)
	}
}

func TestInfoCommand(t *testing.T) {
	testFile := createTestFile(t)

	output := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"info", testFile, "Sheet1"})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("info command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Sheet1") {
		t.Errorf("Expected output to contain 'Sheet1', got: %s", output)
	}
	if !strings.Contains(output, "rows") {
		t.Errorf("Expected output to contain 'rows', got: %s", output)
	}
}

func TestHeadCommand(t *testing.T) {
	testFile := createTestFile(t)

	output := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"head", testFile, "Sheet1", "-n", "2"})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("head command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Name") {
		t.Errorf("Expected output to contain 'Name', got: %s", output)
	}
}

func TestTailCommand(t *testing.T) {
	testFile := createTestFile(t)

	output := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"tail", testFile, "Sheet1", "-n", "2"})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("tail command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Bob") || !strings.Contains(output, "Charlie") {
		t.Errorf("Expected output to contain last rows, got: %s", output)
	}
}

func TestCellCommand(t *testing.T) {
	testFile := createTestFile(t)

	output := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"cell", testFile, "Sheet1", "A2"})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("cell command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Alice") {
		t.Errorf("Expected output to contain 'Alice', got: %s", output)
	}
}

func TestSearchCommand(t *testing.T) {
	testFile := createTestFile(t)

	output := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"search", testFile, "Alice"})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("search command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Alice") {
		t.Errorf("Expected output to contain 'Alice', got: %s", output)
	}
}

func TestReadCommand(t *testing.T) {
	testFile := createTestFile(t)

	output := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"read", testFile, "Sheet1", "A1:B2"})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("read command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Name") {
		t.Errorf("Expected output to contain 'Name', got: %s", output)
	}
	if !strings.Contains(output, "Alice") {
		t.Errorf("Expected output to contain 'Alice', got: %s", output)
	}
}

func TestFormatFlag(t *testing.T) {
	testFile := createTestFile(t)

	tests := []struct {
		format   string
		expected string
	}{
		{"json", "["},
		{"csv", "Name,Age,City"},
		{"tsv", "Name\tAge\tCity"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			output := captureOutput(t, func() {
				rootCmd.SetArgs([]string{"head", testFile, "Sheet1", "-n", "1", "--format", tt.format})
				if err := rootCmd.Execute(); err != nil {
					t.Errorf("head command with format %s failed: %v", tt.format, err)
				}
			})

			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.expected, output)
			}
		})
	}
}

func TestInvalidFile(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	rootCmd.SetArgs([]string{"sheets", "nonexistent.xlsx"})
	err = rootCmd.Execute()

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}

	// Read stderr
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
}

// Reset root command after tests
func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
