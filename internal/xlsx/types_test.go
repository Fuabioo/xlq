package xlsx

import (
	"errors"
	"testing"
)

func TestParseRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  error
		startCol int
		startRow int
		endCol   int
		endRow   int
	}{
		{
			name:     "single cell A1",
			input:    "A1",
			wantErr:  nil,
			startCol: 1,
			startRow: 1,
			endCol:   1,
			endRow:   1,
		},
		{
			name:     "simple range B2:D5",
			input:    "B2:D5",
			wantErr:  nil,
			startCol: 2,
			startRow: 2,
			endCol:   4,
			endRow:   5,
		},
		{
			name:     "large column range AA100:AB200",
			input:    "AA100:AB200",
			wantErr:  nil,
			startCol: 27,
			startRow: 100,
			endCol:   28,
			endRow:   200,
		},
		{
			name:     "reversed range should normalize D5:B2",
			input:    "D5:B2",
			wantErr:  nil,
			startCol: 2,
			startRow: 2,
			endCol:   4,
			endRow:   5,
		},
		{
			name:     "lowercase should work a1:b2",
			input:    "a1:b2",
			wantErr:  nil,
			startCol: 1,
			startRow: 1,
			endCol:   2,
			endRow:   2,
		},
		{
			name:     "with spaces should work",
			input:    " A1:B2 ",
			wantErr:  nil,
			startCol: 1,
			startRow: 1,
			endCol:   2,
			endRow:   2,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "incomplete range",
			input:   "A1:B",
			wantErr: ErrInvalidRange,
		},
		{
			name:    "too many colons",
			input:   "A1:B2:C3",
			wantErr: ErrInvalidRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ParseRange(tt.input)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ParseRange(%q) expected error %v, got nil", tt.input, tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseRange(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRange(%q) unexpected error: %v", tt.input, err)
				return
			}

			if r.StartCol != tt.startCol || r.StartRow != tt.startRow ||
				r.EndCol != tt.endCol || r.EndRow != tt.endRow {
				t.Errorf("ParseRange(%q) = {%d,%d,%d,%d}, want {%d,%d,%d,%d}",
					tt.input,
					r.StartCol, r.StartRow, r.EndCol, r.EndRow,
					tt.startCol, tt.startRow, tt.endCol, tt.endRow)
			}
		})
	}
}

func TestColumnConversion(t *testing.T) {
	tests := []struct {
		name string
		num  int
	}{
		{"A", 1},
		{"B", 2},
		{"Z", 26},
		{"AA", 27},
		{"AB", 28},
		{"AZ", 52},
		{"BA", 53},
		{"ZZ", 702},
		{"AAA", 703},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test name to number
			got := ColumnNameToNumber(tt.name)
			if got != tt.num {
				t.Errorf("ColumnNameToNumber(%q) = %d, want %d", tt.name, got, tt.num)
			}

			// Test number to name
			gotName := ColumnNumberToName(tt.num)
			if gotName != tt.name {
				t.Errorf("ColumnNumberToName(%d) = %q, want %q", tt.num, gotName, tt.name)
			}
		})
	}
}

func TestColumnNameToNumberCaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"a", 1},
		{"A", 1},
		{"aa", 27},
		{"AA", 27},
		{"aA", 27},
		{"Aa", 27},
	}

	for _, tt := range tests {
		got := ColumnNameToNumber(tt.input)
		if got != tt.want {
			t.Errorf("ColumnNameToNumber(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseCellAddress(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantCol int
		wantRow int
		wantErr error
	}{
		{
			name:    "simple A1",
			addr:    "A1",
			wantCol: 1,
			wantRow: 1,
			wantErr: nil,
		},
		{
			name:    "B10",
			addr:    "B10",
			wantCol: 2,
			wantRow: 10,
			wantErr: nil,
		},
		{
			name:    "AA100",
			addr:    "AA100",
			wantCol: 27,
			wantRow: 100,
			wantErr: nil,
		},
		{
			name:    "lowercase a1",
			addr:    "a1",
			wantCol: 1,
			wantRow: 1,
			wantErr: nil,
		},
		{
			name:    "with spaces",
			addr:    " B5 ",
			wantCol: 2,
			wantRow: 5,
			wantErr: nil,
		},
		{
			name:    "invalid format",
			addr:    "invalid",
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "only letter",
			addr:    "A",
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "only number",
			addr:    "1",
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "zero row",
			addr:    "A0",
			wantErr: ErrInvalidAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, row, err := ParseCellAddress(tt.addr)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ParseCellAddress(%q) expected error %v, got nil", tt.addr, tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseCellAddress(%q) error = %v, want %v", tt.addr, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseCellAddress(%q) unexpected error: %v", tt.addr, err)
				return
			}

			if col != tt.wantCol || row != tt.wantRow {
				t.Errorf("ParseCellAddress(%q) = (%d, %d), want (%d, %d)",
					tt.addr, col, row, tt.wantCol, tt.wantRow)
			}
		})
	}
}

func TestFormatCellAddress(t *testing.T) {
	tests := []struct {
		col  int
		row  int
		want string
	}{
		{1, 1, "A1"},
		{2, 10, "B10"},
		{26, 1, "Z1"},
		{27, 100, "AA100"},
		{52, 200, "AZ200"},
	}

	for _, tt := range tests {
		got := FormatCellAddress(tt.col, tt.row)
		if got != tt.want {
			t.Errorf("FormatCellAddress(%d, %d) = %q, want %q",
				tt.col, tt.row, got, tt.want)
		}
	}
}

func TestCellRangeContains(t *testing.T) {
	r := &CellRange{
		StartCol: 2, // B
		StartRow: 2,
		EndCol:   4, // D
		EndRow:   5,
	}

	tests := []struct {
		col  int
		row  int
		want bool
	}{
		{2, 2, true},  // B2 - start
		{4, 5, true},  // D5 - end
		{3, 3, true},  // C3 - middle
		{1, 2, false}, // A2 - outside
		{5, 3, false}, // E3 - outside
		{3, 1, false}, // C1 - outside
		{3, 6, false}, // C6 - outside
	}

	for _, tt := range tests {
		got := r.Contains(tt.col, tt.row)
		if got != tt.want {
			t.Errorf("CellRange{B2:D5}.Contains(%d, %d) = %v, want %v",
				tt.col, tt.row, got, tt.want)
		}
	}
}

func TestCellRangeString(t *testing.T) {
	tests := []struct {
		name string
		r    *CellRange
		want string
	}{
		{
			name: "single cell",
			r: &CellRange{
				StartCol: 1,
				StartRow: 1,
				EndCol:   1,
				EndRow:   1,
			},
			want: "A1",
		},
		{
			name: "range",
			r: &CellRange{
				StartCol: 2,
				StartRow: 2,
				EndCol:   4,
				EndRow:   5,
			},
			want: "B2:D5",
		},
		{
			name: "large range",
			r: &CellRange{
				StartCol: 27,
				StartRow: 100,
				EndCol:   28,
				EndRow:   200,
			},
			want: "AA100:AB200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.String()
			if got != tt.want {
				t.Errorf("CellRange.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
