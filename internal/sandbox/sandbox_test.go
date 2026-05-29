package sandbox

import (
	"testing"
)

func TestStripAnsi_NoEscapes(t *testing.T) {
	input := "hello world"
	got := stripAnsi(input)
	if got != input {
		t.Errorf("stripAnsi(%q) = %q, want %q", input, got, input)
	}
}

func TestStripAnsi_WithColorCodes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple color",
			input:    "\033[32mReady\033[0m",
			expected: "Ready",
		},
		{
			name:     "bold text",
			input:    "\033[1mBold\033[0m",
			expected: "Bold",
		},
		{
			name:     "multiple colors",
			input:    "\033[31mred\033[0m \033[32mgreen\033[0m",
			expected: "red green",
		},
		{
			name:     "256-color escape sequence",
			input:    "\033[38;5;196mcolored\033[0m",
			expected: "colored",
		},
		{
			name:     "mixed content",
			input:    "Name: \033[1mmy-sandbox\033[0m  Status: \033[32mReady\033[0m",
			expected: "Name: my-sandbox  Status: Ready",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only escapes",
			input:    "\033[0m\033[1m\033[32m",
			expected: "",
		},
		{
			name:     "newlines preserved",
			input:    "line1\n\033[32mline2\033[0m\nline3",
			expected: "line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripAnsi(tt.input)
			if got != tt.expected {
				t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripAnsi_TabularOutput(t *testing.T) {
	// Simulate the kind of output `openshell sandbox list` would produce
	input := "\033[1mNAME\033[0m          \033[1mSTATUS\033[0m\n" +
		"my-sandbox    \033[32mReady\033[0m\n" +
		"other-sandbox \033[33mPending\033[0m\n"

	got := stripAnsi(input)
	expected := "NAME          STATUS\nmy-sandbox    Ready\nother-sandbox Pending\n"

	if got != expected {
		t.Errorf("stripAnsi() = %q, want %q", got, expected)
	}
}
