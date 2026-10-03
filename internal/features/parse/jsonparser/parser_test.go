package jsonparser

import (
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func TestParseStringValueWithoutJSONQuotes(t *testing.T) {
	parser := New(&domain.Scheme{
		Parameters: []domain.Field{
			{
				Name:      "level",
				FieldType: domain.StringType,
			},
		},
	})

	entry, err := parser.Parse([]byte(`{"level":"DEBUG"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entry.Values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(entry.Values))
	}

	level, ok := entry.Values[0].(domain.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", entry.Values[0])
	}

	if level != domain.StringValue("DEBUG") {
		t.Fatalf("expected DEBUG without JSON quotes, got %q", level)
	}
}
