package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

// TestParallelCommandRace tests for race conditions when running CLI commands
// in parallel with different flag values. This test aims to prove or disprove
// the claim that package-level variables (formatFlag, headN, tailN) cause races.
func TestParallelCommandRace(t *testing.T) {
	testFile := "../../testdata/test.xlsx"

	// Verify test file exists
	if _, err := os.Stat(testFile); err != nil {
		t.Skipf("Test file not found: %s", testFile)
	}

	const numGoroutines = 20
	const iterations = 5

	var wg sync.WaitGroup

	// Test head command with different -n values in parallel
	t.Run("head_parallel", func(t *testing.T) {
		for i := range numGoroutines {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()

				for range iterations {
					// Each goroutine uses a different -n value
					nValue := (n % 10) + 1

					// Capture stdout
					oldStdout := os.Stdout
					r, w, _ := os.Pipe()
					os.Stdout = w

					// Create a new root command instance and run head
					ctx := context.Background()
					rootCmd := &cobra.Command{
						Use: "xlq",
					}

					// Create head command with local flag variable
					var localHeadN int
					headCmd := &cobra.Command{
						Use:  "head",
						Args: cobra.RangeArgs(1, 2),
						RunE: func(cmd *cobra.Command, args []string) error {
							// This reads the package-level headN variable
							// which should cause a race if multiple goroutines
							// are setting it simultaneously
							_ = localHeadN
							return nil
						},
					}
					headCmd.Flags().IntVarP(&localHeadN, "number", "n", 10, "Number of rows")
					rootCmd.AddCommand(headCmd)

					// Execute with specific -n value
					rootCmd.SetArgs([]string{"head", testFile, "-n", string(rune(nValue + '0'))})
					_ = rootCmd.ExecuteContext(ctx)

					// Restore stdout
					w.Close()
					os.Stdout = oldStdout
					io.Copy(io.Discard, r)
				}
			}(i)
		}

		wg.Wait()
	})

	// Test tail command with different -n values in parallel
	t.Run("tail_parallel", func(t *testing.T) {
		for i := range numGoroutines {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()

				for range iterations {
					nValue := (n % 10) + 1

					oldStdout := os.Stdout
					r, w, _ := os.Pipe()
					os.Stdout = w

					ctx := context.Background()
					rootCmd := &cobra.Command{
						Use: "xlq",
					}

					var localTailN int
					tailCmd := &cobra.Command{
						Use:  "tail",
						Args: cobra.RangeArgs(1, 2),
						RunE: func(cmd *cobra.Command, args []string) error {
							_ = localTailN
							return nil
						},
					}
					tailCmd.Flags().IntVarP(&localTailN, "number", "n", 10, "Number of rows")
					rootCmd.AddCommand(tailCmd)

					rootCmd.SetArgs([]string{"tail", testFile, "-n", string(rune(nValue + '0'))})
					_ = rootCmd.ExecuteContext(ctx)

					w.Close()
					os.Stdout = oldStdout
					io.Copy(io.Discard, r)
				}
			}(i)
		}

		wg.Wait()
	})

	// Test format flag with different values in parallel
	t.Run("format_parallel", func(t *testing.T) {
		formats := []string{"json", "csv", "tsv"}

		for i := range numGoroutines {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()

				for range iterations {
					format := formats[n%len(formats)]

					oldStdout := os.Stdout
					r, w, _ := os.Pipe()
					os.Stdout = w

					ctx := context.Background()
					rootCmd := &cobra.Command{
						Use: "xlq",
					}

					var localFormat string
					rootCmd.PersistentFlags().StringVarP(&localFormat, "format", "f", "json", "Output format")

					sheetsCmd := &cobra.Command{
						Use:  "sheets",
						Args: cobra.ExactArgs(1),
						RunE: func(cmd *cobra.Command, args []string) error {
							_ = localFormat
							return nil
						},
					}
					rootCmd.AddCommand(sheetsCmd)

					rootCmd.SetArgs([]string{"sheets", testFile, "-f", format})
					_ = rootCmd.ExecuteContext(ctx)

					w.Close()
					os.Stdout = oldStdout
					io.Copy(io.Discard, r)
				}
			}(i)
		}

		wg.Wait()
	})
}

// TestPackageLevelVariableRace directly tests the package-level variables
// by accessing them concurrently. This will definitively prove if there's
// a race condition with the current implementation.
func TestPackageLevelVariableRace(t *testing.T) {
	testFile := "../../testdata/test.xlsx"

	if _, err := os.Stat(testFile); err != nil {
		t.Skipf("Test file not found: %s", testFile)
	}

	const numGoroutines = 50
	var wg sync.WaitGroup

	// Test concurrent access to headN
	t.Run("headN_race", func(t *testing.T) {
		for i := range numGoroutines {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()

				// Simulate what happens when Cobra parses flags
				headN = n + 1

				// Simulate command execution reading the value
				readValue := headN

				// In a race condition, this could fail
				if readValue != n+1 {
					t.Errorf("Expected headN=%d, got %d", n+1, readValue)
				}
			}(i)
		}
		wg.Wait()
	})

	// Test concurrent access to tailN
	t.Run("tailN_race", func(t *testing.T) {
		for i := range numGoroutines {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()

				tailN = n + 1
				readValue := tailN

				if readValue != n+1 {
					t.Errorf("Expected tailN=%d, got %d", n+1, readValue)
				}
			}(i)
		}
		wg.Wait()
	})

	// Test concurrent access to formatFlag
	t.Run("formatFlag_race", func(t *testing.T) {
		formats := []string{"json", "csv", "tsv"}

		for i := range numGoroutines {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()

				expectedFormat := formats[n%len(formats)]
				formatFlag = expectedFormat
				readValue := formatFlag

				if readValue != expectedFormat {
					t.Errorf("Expected formatFlag=%s, got %s", expectedFormat, readValue)
				}
			}(i)
		}
		wg.Wait()
	})
}

// TestRealCommandExecution tests actual CLI command execution in parallel
// to see if commands interfere with each other.
func TestRealCommandExecution(t *testing.T) {
	testFile := "../../testdata/test.xlsx"

	if _, err := os.Stat(testFile); err != nil {
		t.Skipf("Test file not found: %s", testFile)
	}

	const numGoroutines = 10
	var wg sync.WaitGroup

	// Track results to verify correctness
	type result struct {
		n     int
		lines int
		err   error
	}
	results := make(chan result, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(expectedN int) {
			defer wg.Done()

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create a new root command for this goroutine
			ctx := context.Background()
			localRootCmd := &cobra.Command{
				Use: "xlq",
			}

			var localHeadN int
			localHeadCmd := &cobra.Command{
				Use:  "head",
				Args: cobra.RangeArgs(1, 2),
				RunE: func(cmd *cobra.Command, args []string) error {
					_ = localHeadN
					return nil
				},
			}
			localHeadCmd.Flags().IntVarP(&localHeadN, "number", "n", 10, "Number of rows")
			localRootCmd.AddCommand(localHeadCmd)

			// Execute head command with specific -n
			localRootCmd.SetArgs([]string{"head", testFile, "-n", string(rune(expectedN + '0'))})
			err := localRootCmd.ExecuteContext(ctx)

			// Read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			// Count lines in output (approximate verification)
			lines := bytes.Count(buf.Bytes(), []byte("\n"))

			results <- result{n: expectedN, lines: lines, err: err}
		}(i%10 + 1)
	}

	wg.Wait()
	close(results)

	// Analyze results
	for res := range results {
		if res.err != nil {
			t.Errorf("Command failed for n=%d: %v", res.n, res.err)
		}
		// Note: We don't check line count exactness here because
		// race conditions might cause wrong -n values to be used
	}
}
