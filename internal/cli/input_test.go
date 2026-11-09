package cli

import (
	"testing"
)

func TestCompleteSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		complete bool
	}{
		{
			name:     "single line with semicolon",
			input:    "SELECT COUNT(*);",
			expected: "SELECT COUNT(*)",
			complete: true,
		},
		{
			name:     "single line without semicolon",
			input:    "SELECT COUNT(*)",
			expected: "",
			complete: false,
		},
		{
			name: "multiline",
			input: `SELECT timestamp, message
WHERE level="ERROR"
LIMIT 10;`,
			expected: `SELECT timestamp, message
WHERE level="ERROR"
LIMIT 10`,
			complete: true,
		},
		{
			name:     "semicolon inside quotes",
			input:    `SELECT message WHERE message="a;b";`,
			expected: `SELECT message WHERE message="a;b"`,
			complete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, complete, err := completeLSQL(tt.input)
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.expected {
				t.Errorf(
					"got SQL %q, want %q",
					got,
					tt.expected,
				)
			}

			if complete != tt.complete {
				t.Errorf(
					"got complete=%v, want %v",
					complete,
					tt.complete,
				)
			}
		})
	}
}
