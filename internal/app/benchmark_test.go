package app

import (
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func BenchmarkService_Query(b *testing.B) {
	service, err := NewService(&domain.Scheme{
		Separator: ',',
		Parameters: []domain.Field{
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
	},
		domain.JSON,
		"../../.storage/logs",
	)

	if err != nil {
		b.Fatalf("new service: %v", err)
	}

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "Exact",
			query: `SELECT * WHERE level = "ERROR"`,
		},
		{
			name:  "Intersect",
			query: `SELECT * WHERE level = "ERROR" AND pid > 100`,
		},
		{
			name:  "IntersectWithLimit",
			query: `SELECT * WHERE level = "ERROR" AND pid > 100 LIMIT 10`,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				seq, err := service.Query(tt.query)
				if err != nil {
					b.Errorf("query: %v", err)
				}

				for _, err := range seq {
					if err != nil {
						b.Errorf("iterator: %v", err)
					}
				}
			}
		})
	}

	b.Cleanup(func() {
		if err := service.Close(); err != nil {
			b.Errorf("close service: %v", err)
		}
	})
}
