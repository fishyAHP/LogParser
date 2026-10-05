package app

import (
	"strings"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func newDefault() *Service {
	service, _ := NewService(&domain.Scheme{
		Separator: ',',
		Parameters: []domain.Field{
			{
				Name:      "level",
				FieldType: domain.StringType,
				IndexType: domain.HashIndex,
			},
		},
	},
		domain.JSON,
		".storage/logs",
	)

	return service
}

func TestService_IngestReader(t *testing.T) {
	service := newDefault()
	input := strings.NewReader(`
{"level":"INFO","pid":10}
{"level":"DEBUG","pid":100}
{"level":"DEBUG","pid":1}
`)

	err := service.IngestReader(input)
	if err != nil {
		t.Fatal(err)
	}

	logs, err := service.Query(`level = "DEBUG"`)
	if err != nil {
		t.Fatal(err)
	}

	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
}
