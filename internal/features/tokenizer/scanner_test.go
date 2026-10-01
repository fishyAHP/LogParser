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
			gotRune, gotOK := scanner.Peek()

			if gotRune != tt.wantRune {
				t.Errorf("Peek() rune = %q, want %q", gotRune, tt.wantRune)
			}

			if gotOK != tt.wantOK {
				t.Errorf("Peek() ok = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestScanner_PeekAfterAdvance(t *testing.T) {
	scanner := NewScanner("abc")

	if err := scanner.Advance(); err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	got, ok := scanner.Peek()

	if !ok {
		t.Fatal("Peek() ok = false, want true")
	}

	if got != 'b' {
		t.Errorf("Peek() = %q, want %q", got, 'b')
	}
}

func TestScanner_PeekAtEOF(t *testing.T) {
	scanner := NewScanner("abc")

	for scanner.Advance() == nil {
	}

	got, ok := scanner.Peek()

	if ok {
		t.Errorf("Peek() ok = true, want false")
	}

	if got != 0 {
		t.Errorf("Peek() rune = %q, want 0", got)
	}
}

func TestScanner_Advance(t *testing.T) {
	scanner := NewScanner("abc")

	for i := 0; i < len([]rune("abc")); i++ {
		if err := scanner.Advance(); err != nil {
			t.Fatalf("Advance() at step %d returned error: %v", i, err)
		}
	}

	err := scanner.Advance()

	if !errors.Is(err, EOF) {
		t.Errorf("Advance() error = %v, want EOF", err)
	}
}

func TestScanner_GetValue(t *testing.T) {
	scanner := NewScanner("ERROR|database")

	// Перемещаемся до separator.
	for i := 0; i < 5; i++ {
		if err := scanner.Advance(); err != nil {
			t.Fatalf("Advance() error = %v", err)
		}
	}

	got := scanner.GetValue()

	if got != "ERROR" {
		t.Errorf("GetValue() = %q, want %q", got, "ERROR")
	}
}

func TestScanner_GetValueMultipleTimes(t *testing.T) {
	scanner := NewScanner("ERROR|database")

	// ERROR
	for i := 0; i < 5; i++ {
		if err := scanner.Advance(); err != nil {
			t.Fatalf("Advance() error = %v", err)
		}
	}

	got := scanner.GetValue()

	if got != "ERROR" {
		t.Errorf("first GetValue() = %q, want %q", got, "ERROR")
	}

	// |
	if err := scanner.Advance(); err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	// database
	for i := 0; i < 8; i++ {
		if err := scanner.Advance(); err != nil {
			t.Fatalf("Advance() error = %v", err)
		}
	}

	got = scanner.GetValue()

	if got != "database" {
		t.Errorf("second GetValue() = %q, want %q", got, "database")
	}
}

func TestScanner_GetValueAtEOF(t *testing.T) {
	scanner := NewScanner("abc")

	for scanner.Advance() == nil {
	}

	got := scanner.GetValue()

	if got != "abc" {
		t.Errorf("GetValue() = %q, want %q", got, "abc")
	}
}

func TestScanner_EmptyInput(t *testing.T) {
	scanner := NewScanner("")

	_, ok := scanner.Peek()

	if ok {
		t.Error("Peek() ok = true for empty input, want false")
	}

	err := scanner.Advance()

	if !errors.Is(err, EOF) {
		t.Errorf("Advance() error = %v, want EOF", err)
	}

	got := scanner.GetValue()

	if got != "" {
		t.Errorf("GetValue() = %q, want empty string", got)
	}
}

func TestScanner_Unicode(t *testing.T) {
	scanner := NewScanner("Привет")

	for i := 0; i < len([]rune("Привет")); i++ {
		if err := scanner.Advance(); err != nil {
			t.Fatalf("Advance() error = %v", err)
		}
	}

	got := scanner.GetValue()

	if got != "Привет" {
		t.Errorf("GetValue() = %q, want %q", got, "Привет")
	}
}
