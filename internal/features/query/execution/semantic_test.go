package execution

import (
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func TestSemanticAnalyzer_InvalidAggregateCombinations(t *testing.T) {
	scheme := &domain.Scheme{
		Parameters: []domain.Field{
			{
				Name:      "level",
				FieldType: domain.StringType,
				IndexType: domain.HashIndex,
			},
		},
	}

	analyzer := newSemanticAnalyzer(scheme)
	catalog := domain.NewCatalog()

	tests := []struct {
		name       string
		projection []SelectExpr
	}{
		{
			name: "field mixed with aggregate",
			projection: []SelectExpr{
				SelectedField("level"),
				SelectAggregate{
					Func:  Count,
					Field: "*",
				},
			},
		},
		{
			name: "aggregate followed by unknown field",
			projection: []SelectExpr{
				SelectAggregate{
					Func:  Count,
					Field: "*",
				},
				SelectedField("unknown"),
			},
		},
		{
			name: "wildcard mixed with field",
			projection: []SelectExpr{
				SelectWildCard{},
				SelectedField("level"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := analyzer.analyze(
				query{
					Projection: tt.projection,
				},
				catalog,
			)

			if err == nil {
				t.Fatal("expected semantic error, got nil")
			}
		})
	}
}
