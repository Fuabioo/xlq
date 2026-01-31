package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveFilePath(t *testing.T) {
	tests := []struct {
		name        string
		basepath    string
		file        string
		expected    string
		shouldError bool
		errorMsg    string
	}{
		{
			name:     "empty basepath returns file unchanged",
			basepath: "",
			file:     "test.xlsx",
			expected: "test.xlsx",
		},
		{
			name:     "absolute file ignores basepath",
			basepath: "/tmp/base",
			file:     "/absolute/path/test.xlsx",
			expected: "/absolute/path/test.xlsx",
		},
		{
			name:     "relative file joined with basepath",
			basepath: "/tmp/base",
			file:     "test.xlsx",
			expected: filepath.Join("/tmp/base", "test.xlsx"),
		},
		{
			name:     "relative file with subdirectory",
			basepath: "/tmp/base",
			file:     "subdir/test.xlsx",
			expected: filepath.Join("/tmp/base", "subdir/test.xlsx"),
		},
		{
			name:     "basepath with trailing slash",
			basepath: "/tmp/base/",
			file:     "test.xlsx",
			expected: filepath.Join("/tmp/base/", "test.xlsx"),
		},
		{
			name:     "both empty",
			basepath: "",
			file:     "",
			expected: "",
		},
		{
			name:     "safe relative path with .. staying inside",
			basepath: "/tmp/base",
			file:     "sub/../file.xlsx",
			expected: filepath.Join("/tmp/base", "file.xlsx"),
		},
		{
			name:        "path traversal with ../../",
			basepath:    "/tmp/base",
			file:        "../../etc/passwd",
			shouldError: true,
			errorMsg:    "path traversal denied",
		},
		{
			name:        "path traversal with multiple ../",
			basepath:    "/tmp/base",
			file:        "../../../etc/shadow",
			shouldError: true,
			errorMsg:    "path traversal denied",
		},
		{
			name:        "path traversal single ..",
			basepath:    "/tmp/base",
			file:        "..",
			shouldError: true,
			errorMsg:    "path traversal denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveFilePath(tt.basepath, tt.file)
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got result %q", result)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ResolveFilePath(%q, %q) = %q, want %q",
					tt.basepath, tt.file, result, tt.expected)
			}
		})
	}
}

func TestGetBasepathFromCmd(t *testing.T) {
	t.Run("flag value takes precedence", func(t *testing.T) {
		parent := &cobra.Command{Use: "root"}
		parent.PersistentFlags().StringP("basepath", "b", "", "")
		child := &cobra.Command{Use: "child", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
		parent.AddCommand(child)

		parent.SetArgs([]string{"child", "--basepath", "/from/flag"})
		t.Setenv("XLQ_BASEPATH", "/from/env")

		if err := parent.Execute(); err != nil {
			t.Fatalf("execute failed: %v", err)
		}

		result := GetBasepathFromCmd(child)
		if result != "/from/flag" {
			t.Errorf("expected /from/flag, got %q", result)
		}
	})

	t.Run("env fallback when flag empty", func(t *testing.T) {
		parent := &cobra.Command{Use: "root"}
		parent.PersistentFlags().StringP("basepath", "b", "", "")
		child := &cobra.Command{Use: "child", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
		parent.AddCommand(child)

		parent.SetArgs([]string{"child"})
		t.Setenv("XLQ_BASEPATH", "/from/env")

		if err := parent.Execute(); err != nil {
			t.Fatalf("execute failed: %v", err)
		}

		result := GetBasepathFromCmd(child)
		if result != "/from/env" {
			t.Errorf("expected /from/env, got %q", result)
		}
	})

	t.Run("empty when both unset", func(t *testing.T) {
		parent := &cobra.Command{Use: "root"}
		parent.PersistentFlags().StringP("basepath", "b", "", "")
		child := &cobra.Command{Use: "child", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
		parent.AddCommand(child)

		parent.SetArgs([]string{"child"})
		t.Setenv("XLQ_BASEPATH", "")

		if err := parent.Execute(); err != nil {
			t.Fatalf("execute failed: %v", err)
		}

		result := GetBasepathFromCmd(child)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("no basepath flag registered", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		t.Setenv("XLQ_BASEPATH", "/from/env")

		result := GetBasepathFromCmd(cmd)
		if result != "/from/env" {
			t.Errorf("expected /from/env, got %q", result)
		}
	})
}

func TestSheetsCommandWithBasepath(t *testing.T) {
	testFile := createTestFile(t)
	dir := filepath.Dir(testFile)
	base := filepath.Base(testFile)

	output := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"--basepath", dir, "sheets", base})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("sheets command with --basepath failed: %v", err)
		}
	})

	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestSheetsCommandWithBasepathEnv(t *testing.T) {
	testFile := createTestFile(t)
	dir := filepath.Dir(testFile)
	base := filepath.Base(testFile)

	t.Setenv("XLQ_BASEPATH", dir)

	output := captureOutput(t, func() {
		// Explicitly reset basepath flag to empty so env var takes effect
		rootCmd.SetArgs([]string{"--basepath", "", "sheets", base})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("sheets command with XLQ_BASEPATH env failed: %v", err)
		}
	})

	if output == "" {
		t.Error("expected non-empty output")
	}
}
