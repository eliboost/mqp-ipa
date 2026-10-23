package instance

import (
	"testing"
)

func TestParseSizeToBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"2Gi", 2 * 1024 * 1024 * 1024, false},
		{"500Mi", 500 * 1024 * 1024, false},
		{"1000", 1000, false},
		{"10K", 10 * 1024, false},
		{"7M", 7 * 1024 * 1024, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseSizeToBytes(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSizeToBytes(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if got != tt.expected {
			t.Errorf("ParseSizeToBytes(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}
