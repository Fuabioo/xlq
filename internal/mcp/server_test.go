package mcp

import (
	"testing"
)

func TestNewServer(t *testing.T) {
	srv := New()
	if srv == nil {
		t.Error("New() returned nil")
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
