package app

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func newAggregationService(t *testing.T) *Service {
	t.Helper()

	scheme := &domain.Scheme{
		Separator: ',',
		Parameters: []domain.Field{
			{
				Name:      "pid",
				FieldType: domain.IntType,
				IndexType: domain.RangeIndex,
			},
			{
				Name:      "level",
				FieldType: domain.StringType,
				IndexType: domain.HashIndex,
			},
		},
	}

	service, err := NewService(
		scheme,
		domain.JSON,
		t.TempDir(),
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	t.Cleanup(func() {
		if err := service.Close(); err != nil {
			t.Errorf("close service: %v", err)
		}
	})

	logs := []string{
		`{"pid":1,"level":"INFO"}`,
		`{"pid":2,"level":"ERROR"}`,
		`{"pid":3,"level":"WARN"}`,
		`{"pid":4,"level":"ERROR"}`,
	}

	for _, log := range logs {
		if err := service.Ingest([]byte(log)); err != nil {
			t.Fatalf("ingest %s: %v", log, err)
		}
	}

	return service
}

func executeAggregation(
	t *testing.T,
	service *Service,
	query string,
) ([]map[string]any, error) {
	t.Helper()

	rows, err := service.Query(query)
	if err != nil {
		return nil, err
	}

	var results []map[string]any

	for data, err := range rows {
		if err != nil {
			return nil, err
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("unmarshal result: %w", err)
		}

		results = append(results, result)
	}

	return results, nil
}

func TestAggregationIntegration(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected map[string]any
	}{
		{
			name:  "COUNT all",
			query: "SELECT COUNT(*)",
			expected: map[string]any{
				"COUNT(*)": float64(4),
			},
		},
		{
			name:  "COUNT field",
			query: "SELECT COUNT(pid)",
			expected: map[string]any{
				"COUNT(pid)": float64(4),
			},
		},
		{
			name:  "SUM",
			query: "SELECT SUM(pid)",
			expected: map[string]any{
				"SUM(pid)": float64(10),
			},
		},
		{
			name:  "AVG",
			query: "SELECT AVG(pid)",
			expected: map[string]any{
				"AVG(pid)": float64(2.5),
			},
		},
		{
			name:  "MIN",
			query: "SELECT MIN(pid)",
			expected: map[string]any{
				"MIN(pid)": float64(1),
			},
		},
		{
			name:  "MAX",
			query: "SELECT MAX(pid)",
			expected: map[string]any{
				"MAX(pid)": float64(4),
			},
		},
		{
			name:  "string MIN MAX",
			query: "SELECT MIN(level), MAX(level)",
			expected: map[string]any{
				"MIN(level)": "ERROR",
				"MAX(level)": "WARN",
			},
		},
		{
			name: "multiple aggregations",
			query: "SELECT COUNT(*), COUNT(pid), SUM(pid), " +
				"AVG(pid), MIN(pid), MAX(pid)",
			expected: map[string]any{
				"COUNT(*)":   float64(4),
				"COUNT(pid)": float64(4),
				"SUM(pid)":   float64(10),
				"AVG(pid)":   float64(2.5),
				"MIN(pid)":   float64(1),
				"MAX(pid)":   float64(4),
			},
		},
		{
			name:  "WHERE hash index",
			query: `SELECT COUNT(*), AVG(pid) WHERE level="ERROR"`,
			expected: map[string]any{
				"COUNT(*)": float64(2),
				"AVG(pid)": float64(3),
			},
		},
		{
			name:  "WHERE range index",
			query: "SELECT COUNT(*), SUM(pid) WHERE pid>=3",
			expected: map[string]any{
				"COUNT(*)": float64(2),
				"SUM(pid)": float64(7),
			},
		},
		{
			name: "WHERE AND",
			query: `SELECT COUNT(*), MIN(pid), MAX(pid) ` +
				`WHERE level="ERROR" AND pid>=3`,
			expected: map[string]any{
				"COUNT(*)": float64(1),
				"MIN(pid)": float64(4),
				"MAX(pid)": float64(4),
			},
		},
		{
			name: "WHERE OR",
			query: `SELECT COUNT(*), SUM(pid), AVG(pid) ` +
				`WHERE level="ERROR" OR level="WARN"`,
			expected: map[string]any{
				"COUNT(*)": float64(3),
				"SUM(pid)": float64(9),
				"AVG(pid)": float64(3),
			},
		},
		{
			name: "WHERE OR no duplicates",
			query: `SELECT COUNT(*), SUM(pid) ` +
				`WHERE level="ERROR" OR pid>=3`,
			expected: map[string]any{
				"COUNT(*)": float64(3),
				"SUM(pid)": float64(9),
			},
		},
		{
			name:  "LIMIT one aggregate row",
			query: "SELECT COUNT(*) LIMIT 1",
			expected: map[string]any{
				"COUNT(*)": float64(4),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newAggregationService(t)

			results, err := executeAggregation(
				t,
				service,
				tt.query,
			)
			if err != nil {
				t.Fatalf("query failed: %v", err)
			}

			if len(results) != 1 {
				t.Fatalf(
					"expected 1 result row, got %d",
					len(results),
				)
			}

			got := results[0]

			if len(got) != len(tt.expected) {
				t.Fatalf(
					"expected %d fields, got %d: %v",
					len(tt.expected),
					len(got),
					got,
				)
			}

			for field, want := range tt.expected {
				actual, ok := got[field]
				if !ok {
					t.Errorf("missing field %q", field)
					continue
				}

				switch expected := want.(type) {
				case float64:
					number, ok := actual.(float64)
					if !ok {
						t.Errorf(
							"%s: expected number, got %T",
							field,
							actual,
						)
						continue
					}

					if math.Abs(number-expected) > 1e-9 {
						t.Errorf(
							"%s: expected %v, got %v",
							field,
							expected,
							number,
						)
					}

				default:
					if actual != want {
						t.Errorf(
							"%s: expected %v, got %v",
							field,
							want,
							actual,
						)
					}
				}
			}
		})
	}
}

func TestAggregationInvalidQueries(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "AVG string",
			query: "SELECT AVG(level)",
		},
		{
			name:  "SUM string",
			query: "SELECT SUM(level)",
		},
		{
			name:  "unknown field",
			query: "SELECT AVG(unknown_field)",
		},
		{
			name:  "mixed aggregate and field",
			query: "SELECT pid, AVG(pid)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newAggregationService(t)

			_, err := executeAggregation(
				t,
				service,
				tt.query,
			)

			if err == nil {
				t.Errorf(
					"expected error for query: %s",
					tt.query,
				)
			}
		})
	}
}

func TestAggregationEmptyResult(t *testing.T) {
	service := newAggregationService(t)

	t.Run("COUNT empty", func(t *testing.T) {
		results, err := executeAggregation(
			t,
			service,
			"SELECT COUNT(*) WHERE pid<0",
		)
		if err != nil {
			t.Fatal(err)
		}

		if len(results) != 1 {
			t.Fatalf(
				"expected 1 row, got %d",
				len(results),
			)
		}

		if results[0]["COUNT(*)"] != float64(0) {
			t.Errorf(
				"expected COUNT(*) = 0, got %v",
				results[0]["COUNT(*)"],
			)
		}
	})

	t.Run("AVG empty", func(t *testing.T) {
		_, err := executeAggregation(
			t,
			service,
			"SELECT AVG(pid) WHERE pid<0",
		)

		if err == nil {
			t.Error("expected error for empty AVG")
		}
	})

	t.Run("MIN empty", func(t *testing.T) {
		_, err := executeAggregation(
			t,
			service,
			"SELECT MIN(pid) WHERE pid<0",
		)

		if err == nil {
			t.Error("expected error for empty MIN")
		}
	})
}

func TestAggregationLimitOffset(t *testing.T) {
	service := newAggregationService(t)

	tests := []struct {
		name         string
		query        string
		expectedRows int
	}{
		{
			name:         "LIMIT 0",
			query:        "SELECT COUNT(*) LIMIT 0",
			expectedRows: 0,
		},
		{
			name:         "OFFSET 1",
			query:        "SELECT COUNT(*) OFFSET 1",
			expectedRows: 0,
		},
		{
			name:         "LIMIT 1 OFFSET 0",
			query:        "SELECT COUNT(*) LIMIT 1 OFFSET 0",
			expectedRows: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := executeAggregation(
				t,
				service,
				tt.query,
			)
			if err != nil {
				t.Fatal(err)
			}

			if len(results) != tt.expectedRows {
				t.Errorf(
					"expected %d rows, got %d",
					tt.expectedRows,
					len(results),
				)
			}
		})
	}
}
