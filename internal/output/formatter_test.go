package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{
			name:    "json lowercase",
			format:  "json",
			wantErr: false,
		},
		{
			name:    "json uppercase",
			format:  "JSON",
			wantErr: false,
		},
		{
			name:    "csv",
			format:  "csv",
			wantErr: false,
		},
		{
			name:    "tsv",
			format:  "tsv",
			wantErr: false,
		},
		{
			name:    "empty defaults to json",
			format:  "",
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFormatter(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFormatter(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestJSONFormatter_FormatValue(t *testing.T) {
	f := &JSONFormatter{}

	tests := []struct {
		name     string
		input    interface{}
		contains string
		wantErr  bool
	}{
		{
			name:     "string map",
			input:    map[string]string{"name": "Sheet1", "rows": "100"},
			contains: `"name":"Sheet1"`,
			wantErr:  false,
		},
		{
			name:     "string value",
			input:    "test",
			contains: `"test"`,
			wantErr:  false,
		},
		{
			name:     "number",
			input:    42,
			contains: "42",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := f.FormatValue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(string(out), tt.contains) {
				t.Errorf("output missing expected content %q: %s", tt.contains, out)
			}
		})
	}
}

func TestJSONFormatter_FormatSlice(t *testing.T) {
	f := &JSONFormatter{}

	tests := []struct {
		name       string
		input      interface{}
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "2d string slice",
			input:      [][]string{{"a", "b"}, {"c", "d"}},
			wantPrefix: "[[",
			wantErr:    false,
		},
		{
			name:       "1d string slice",
			input:      []string{"x", "y", "z"},
			wantPrefix: "[",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := f.FormatSlice(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.HasPrefix(string(out), tt.wantPrefix) {
				t.Errorf("expected prefix %q: %s", tt.wantPrefix, out)
			}
		})
	}
}

func TestJSONFormatter_Streaming(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	err := f.WriteHeader(&buf)
	if err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}

	err = f.WriteSeparator(&buf)
	if err != nil {
		t.Fatalf("WriteSeparator failed: %v", err)
	}
	buf.Write([]byte(`"item1"`))

	err = f.WriteSeparator(&buf)
	if err != nil {
		t.Fatalf("WriteSeparator failed: %v", err)
	}
	buf.Write([]byte(`"item2"`))

	err = f.WriteFooter(&buf)
	if err != nil {
		t.Fatalf("WriteFooter failed: %v", err)
	}

	expected := `["item1","item2"]` + "\n"
	if buf.String() != expected {
		t.Errorf("streaming output = %q, want %q", buf.String(), expected)
	}
}

func TestCSVFormatter_FormatValue(t *testing.T) {
	f := &CSVFormatter{}

	tests := []struct {
		name     string
		input    interface{}
		contains string
		wantErr  bool
	}{
		{
			name:     "simple row",
			input:    []string{"value1", "value2"},
			contains: "value1,value2",
			wantErr:  false,
		},
		{
			name:     "row with comma",
			input:    []string{"value1", "value2", "with,comma"},
			contains: `"with,comma"`,
			wantErr:  false,
		},
		{
			name:     "row with quote",
			input:    []string{"value1", `value"with"quotes`},
			contains: `"value""with""quotes"`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := f.FormatValue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(string(out), tt.contains) {
				t.Errorf("CSV should contain %q: %s", tt.contains, out)
			}
		})
	}
}

func TestCSVFormatter_FormatSlice(t *testing.T) {
	f := &CSVFormatter{}

	rows := [][]string{{"a", "b"}, {"c", "d"}}
	out, err := f.FormatSlice(rows)
	if err != nil {
		t.Fatalf("FormatSlice failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %s", len(lines), out)
	}

	if !strings.Contains(lines[0], "a,b") {
		t.Errorf("first line should contain 'a,b': %s", lines[0])
	}
	if !strings.Contains(lines[1], "c,d") {
		t.Errorf("second line should contain 'c,d': %s", lines[1])
	}
}

func TestTSVFormatter_FormatValue(t *testing.T) {
	f := &TSVFormatter{}

	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "simple row",
			input:    []string{"value1", "value2", "value3"},
			expected: "value1\tvalue2\tvalue3\n",
			wantErr:  false,
		},
		{
			name:     "single value",
			input:    []string{"test"},
			expected: "test\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := f.FormatValue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(out) != tt.expected {
				t.Errorf("got %q, want %q", out, tt.expected)
			}
		})
	}
}

func TestTSVFormatter_FormatSlice(t *testing.T) {
	f := &TSVFormatter{}

	rows := [][]string{{"a", "b"}, {"c", "d"}}
	out, err := f.FormatSlice(rows)
	if err != nil {
		t.Fatalf("FormatSlice failed: %v", err)
	}

	if !strings.Contains(string(out), "a\tb") {
		t.Errorf("TSV output missing tab in first row: %s", out)
	}
	if !strings.Contains(string(out), "c\td") {
		t.Errorf("TSV output missing tab in second row: %s", out)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestFormatRows(t *testing.T) {
	rows := [][]string{{"h1", "h2"}, {"v1", "v2"}}

	tests := []struct {
		name      string
		format    string
		checkFunc func(string) bool
		wantErr   bool
	}{
		{
			name:   "json format",
			format: "json",
			checkFunc: func(s string) bool {
				return strings.HasPrefix(s, "[[")
			},
			wantErr: false,
		},
		{
			name:   "csv format",
			format: "csv",
			checkFunc: func(s string) bool {
				lines := strings.Split(strings.TrimSpace(s), "\n")
				return len(lines) == 2
			},
			wantErr: false,
		},
		{
			name:   "tsv format",
			format: "tsv",
			checkFunc: func(s string) bool {
				return strings.Contains(s, "\t")
			},
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := FormatRows(tt.format, rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatRows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !tt.checkFunc(string(out)) {
				t.Errorf("FormatRows(%q) output check failed: %s", tt.format, out)
			}
		})
	}
}

func TestFormatSingle(t *testing.T) {
	data := map[string]interface{}{"name": "test", "count": 42}

	tests := []struct {
		name     string
		format   string
		contains string
		wantErr  bool
	}{
		{
			name:     "json single object",
			format:   "json",
			contains: `"name":"test"`,
			wantErr:  false,
		},
		{
			name:     "empty format defaults to json",
			format:   "",
			contains: `"name":"test"`,
			wantErr:  false,
		},
		{
			name:    "invalid format",
			format:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := FormatSingle(tt.format, data)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatSingle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(string(out), tt.contains) {
				t.Errorf("JSON output missing expected content %q: %s", tt.contains, out)
			}
		})
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		wantLen   int
		wantFirst string
		wantErr   bool
	}{
		{
			name:      "string slice passthrough",
			input:     []string{"a", "b"},
			wantLen:   2,
			wantFirst: "a",
			wantErr:   false,
		},
		{
			name:      "interface slice",
			input:     []interface{}{"x", 123, true},
			wantLen:   3,
			wantFirst: "x",
			wantErr:   false,
		},
		{
			name:      "map",
			input:     map[string]interface{}{"key": "value"},
			wantLen:   1,
			wantFirst: "value",
			wantErr:   false,
		},
		{
			name:      "single value",
			input:     "test",
			wantLen:   1,
			wantFirst: "test",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := toStringSlice(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toStringSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(out) != tt.wantLen {
					t.Errorf("expected length %d, got %d: %v", tt.wantLen, len(out), out)
				}
				if len(out) > 0 && !strings.Contains(out[0], tt.wantFirst) {
					t.Errorf("expected first item to contain %q, got %q", tt.wantFirst, out[0])
				}
			}
		})
	}
}

func TestToStringSliceSlice(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantLen int
		wantErr bool
	}{
		{
			name:    "2d string slice passthrough",
			input:   [][]string{{"a", "b"}, {"c", "d"}},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "interface slice",
			input:   []interface{}{[]string{"a", "b"}, []string{"c", "d"}},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "single row",
			input:   []string{"a", "b"},
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := toStringSliceSlice(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("toStringSliceSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(out) != tt.wantLen {
				t.Errorf("expected %d rows, got %d", tt.wantLen, len(out))
			}
		})
	}
}

func TestCSVFormatter_NoHeaderFooter(t *testing.T) {
	f := &CSVFormatter{}
	var buf bytes.Buffer

	err := f.WriteHeader(&buf)
	if err != nil {
		t.Errorf("WriteHeader should not error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("CSV WriteHeader should write nothing, got: %s", buf.String())
	}

	buf.Reset()
	err = f.WriteFooter(&buf)
	if err != nil {
		t.Errorf("WriteFooter should not error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("CSV WriteFooter should write nothing, got: %s", buf.String())
	}

	err = f.WriteSeparator(&buf)
	if err != nil {
		t.Errorf("WriteSeparator should not error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("CSV WriteSeparator should write nothing, got: %s", buf.String())
	}
}

func TestTSVFormatter_NoHeaderFooter(t *testing.T) {
	f := &TSVFormatter{}
	var buf bytes.Buffer

	err := f.WriteHeader(&buf)
	if err != nil {
		t.Errorf("WriteHeader should not error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("TSV WriteHeader should write nothing, got: %s", buf.String())
	}

	buf.Reset()
	err = f.WriteFooter(&buf)
	if err != nil {
		t.Errorf("WriteFooter should not error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("TSV WriteFooter should write nothing, got: %s", buf.String())
	}

	err = f.WriteSeparator(&buf)
	if err != nil {
		t.Errorf("WriteSeparator should not error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("TSV WriteSeparator should write nothing, got: %s", buf.String())
	}
}
