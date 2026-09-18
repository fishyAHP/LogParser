package tokenizer

import (
	"errors"
	"testing"
)

func TestScanner_Peek(t *testing.T) {
	scanner := NewScanner("abc")

	tests := []struct {
		name     string
		wantRune rune
		wantOK   bool
	}{
		{
			name:     "first character",
			wantRune: 'a',
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRune, gotOK := scanner.peek()

			if gotRune != tt.wantRune {
				t.Errorf("peek() rune = %q, want %q", gotRune, tt.wantRune)
			}

			if gotOK != tt.wantOK {
				t.Errorf("peek() ok = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestScanner_PeekAfterAdvance(t *testing.T) {
	scanner := NewScanner("abc")

	if err := scanner.advance(); err != nil {
		t.Fatalf("advance() error = %v", err)
	}

	got, ok := scanner.peek()

	if !ok {
		t.Fatal("peek() ok = false, want true")
	}

	if got != 'b' {
		t.Errorf("peek() = %q, want %q", got, 'b')
	}
}

func TestScanner_PeekAtEOF(t *testing.T) {
	scanner := NewScanner("abc")

	for scanner.advance() == nil {
	}

	got, ok := scanner.peek()

	if ok {
		t.Errorf("peek() ok = true, want false")
	}

	if got != 0 {
		t.Errorf("peek() rune = %q, want 0", got)
	}
}

func TestScanner_Advance(t *testing.T) {
	scanner := NewScanner("abc")

	for i := 0; i < len([]rune("abc")); i++ {
		if err := scanner.advance(); err != nil {
			t.Fatalf("advance() at step %d returned error: %v", i, err)
		}
	}

	err := scanner.advance()

	if !errors.Is(err, EOF) {
		t.Errorf("advance() error = %v, want EOF", err)
	}
}

func TestScanner_GetValue(t *testing.T) {
	scanner := NewScanner("ERROR|database")

	// Перемещаемся до separator.
	for i := 0; i < 5; i++ {
		if err := scanner.advance(); err != nil {
			t.Fatalf("advance() error = %v", err)
		}
	}

	got := scanner.getValue()

	if got != "ERROR" {
		t.Errorf("getValue() = %q, want %q", got, "ERROR")
	}
}

func TestScanner_GetValueMultipleTimes(t *testing.T) {
	scanner := NewScanner("ERROR|database")

	// ERROR
	for i := 0; i < 5; i++ {
		if err := scanner.advance(); err != nil {
			t.Fatalf("advance() error = %v", err)
		}
	}

	got := scanner.getValue()

	if got != "ERROR" {
		t.Errorf("first getValue() = %q, want %q", got, "ERROR")
	}

	// |
	if err := scanner.advance(); err != nil {
		t.Fatalf("advance() error = %v", err)
	}

	// database
	for i := 0; i < 8; i++ {
		if err := scanner.advance(); err != nil {
			t.Fatalf("advance() error = %v", err)
		}
	}

	got = scanner.getValue()

	if got != "database" {
		t.Errorf("second getValue() = %q, want %q", got, "database")
	}
}

func TestScanner_GetValueAtEOF(t *testing.T) {
	scanner := NewScanner("abc")

	for scanner.advance() == nil {
	}

	got := scanner.getValue()

	if got != "abc" {
		t.Errorf("getValue() = %q, want %q", got, "abc")
	}
}

func TestScanner_EmptyInput(t *testing.T) {
	scanner := NewScanner("")

	_, ok := scanner.peek()

	if ok {
		t.Error("peek() ok = true for empty input, want false")
	}

	err := scanner.advance()

	if !errors.Is(err, EOF) {
		t.Errorf("advance() error = %v, want EOF", err)
	}

	got := scanner.getValue()

	if got != "" {
		t.Errorf("getValue() = %q, want empty string", got)
	}
}

func TestScanner_Unicode(t *testing.T) {
	scanner := NewScanner("Привет")

	for i := 0; i < len([]rune("Привет")); i++ {
		if err := scanner.advance(); err != nil {
			t.Fatalf("advance() error = %v", err)
		}
	}

	got := scanner.getValue()

	if got != "Привет" {
		t.Errorf("getValue() = %q, want %q", got, "Привет")
	}
}
