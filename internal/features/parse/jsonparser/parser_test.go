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

	parsed, err := parser.Parse([]byte(`{"level":"DEBUG"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(parsed.Entry.Values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(parsed.Entry.Values))
	}

	level, ok := parsed.Entry.Values[0].(domain.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", parsed.Entry.Values[0])
	}

	if level != domain.StringValue("DEBUG") {
		t.Fatalf("expected DEBUG without JSON quotes, got %q", level)
	}
}
