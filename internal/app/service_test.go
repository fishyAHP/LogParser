package app

import (
	"strings"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func newDefault(t *testing.T) *Service {
	t.Helper()

	service, err := NewService(&domain.Scheme{
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
		t.TempDir(),
	)

	if err != nil {
		t.Errorf("new service: %v", err)
	}

	t.Cleanup(func() {
		if err := service.Close(); err != nil {
			t.Errorf("close service: %v", err)
		}
	})

	return service
}

func TestService_IngestReader(t *testing.T) {
	service := newDefault(t)
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

func TestService_RebuildIndexesAfterRestart(t *testing.T) {
	dir := t.TempDir()

	scheme := &domain.Scheme{
		Parameters: []domain.Field{
			{
				Name:      "timestamp",
				FieldType: domain.TimeType,
				IndexType: domain.RangeIndex,
			},
			{
				Name:      "level",
				FieldType: domain.StringType,
				IndexType: domain.HashIndex,
			},
			{
				Name:      "pid",
				FieldType: domain.IntType,
				IndexType: domain.RangeIndex,
			},
			{
				Name:      "message",
				FieldType: domain.StringType,
				IndexType: domain.TextIndex,
			},
		},
	}

	service1, err := NewService(
		scheme,
		domain.JSON,
		dir,
	)
	if err != nil {
		t.Fatalf("create first service: %v", err)
	}

	logs := [][]byte{
		[]byte(`{"timestamp":"2026-10-05T10:00:00Z","level":"INFO","pid":100,"message":"server started"}`),
		[]byte(`{"timestamp":"2026-10-05T10:01:00Z","level":"ERROR","pid":200,"message":"database failed"}`),
		[]byte(`{"timestamp":"2026-10-05T10:02:00Z","level":"ERROR","pid":300,"message":"network failed"}`),
	}

	for _, raw := range logs {
		if err := service1.Ingest(raw); err != nil {
			t.Fatalf("ingest: %v", err)
		}
	}

	// До restart должно быть две ERROR-записи.
	before, err := service1.Query(`level = "ERROR"`)
	if err != nil {
		t.Fatalf("query before restart: %v", err)
	}

	if len(before) != 2 {
		t.Fatalf(
			"before restart: expected 2 records, got %d",
			len(before),
		)
	}

	if err := service1.Close(); err != nil {
		t.Fatalf("close first service: %v", err)
	}

	service2, err := NewService(
		scheme,
		domain.JSON,
		dir,
	)
	if err != nil {
		t.Fatalf("create second service: %v", err)
	}
	defer service2.Close()

	// Новый IndexService был пустым.
	// NewService должен был восстановить его из Storage.
	after, err := service2.Query(`level = "ERROR"`)
	if err != nil {
		t.Fatalf("query after restart: %v", err)
	}

	if len(after) != 2 {
		t.Fatalf(
			"after restart: expected 2 records, got %d",
			len(after),
		)
	}

	result, err := service2.Query(`pid >= 200`)
	if err != nil {
		t.Fatalf("range query after restart: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf(
			"expected 2 records with pid >= 200, got %d",
			len(result),
		)
	}

	newLog := []byte(
		`{"timestamp":"2026-10-05T10:03:00Z","level":"ERROR","pid":400,"message":"disk failed"}`,
	)

	if err := service2.Ingest(newLog); err != nil {
		t.Fatalf("ingest after restart: %v", err)
	}

	result, err = service2.Query(`pid >= 200`)
	if err != nil {
		t.Fatalf("range query after new ingest: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf(
			"expected 3 records after new ingest, got %d",
			len(result),
		)
	}
}
