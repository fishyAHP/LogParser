package cli

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestBasicInput_ReadLine(t *testing.T) {
	input := NewBasicInput(strings.NewReader(
		"query Select Count(*)\n" +
			"query SELECT timestamp\n" +
			"exit\n",
	))

	expected := []string{
		"query Select Count(*)",
		"query SELECT timestamp",
		"exit",
	}

	for _, want := range expected {
		got, err := input.ReadLine()
		if err != nil {
			t.Fatalf("ReadLine: %v", err)
		}

		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}

	_, err := input.ReadLine()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}
