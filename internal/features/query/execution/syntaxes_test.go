package execution

import (
	"reflect"
	"testing"
)

func TestSyntaxes_Projection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []SelectExpr
		wantErr bool
	}{
		{
			name:  "wildcard",
			input: "SELECT *",
			want: []SelectExpr{
				SelectWildCard{},
			},
		},
		{
			name:  "single field",
			input: "SELECT level",
			want: []SelectExpr{
				SelectedField("level"),
			},
		},
		{
			name:  "multiple fields",
			input: "SELECT level, pid",
			want: []SelectExpr{
				SelectedField("level"),
				SelectedField("pid"),
			},
		},
		{
			name:  "count all",
			input: "SELECT COUNT(*)",
			want: []SelectExpr{
				SelectAggregate{
					Func:  Count,
					Field: "*",
				},
			},
		},
		{
			name:  "field and count",
			input: "SELECT level, COUNT(*)",
			want: []SelectExpr{
				SelectedField("level"),
				SelectAggregate{
					Func:  Count,
					Field: "*",
				},
			},
		},
		{
			name:  "count field",
			input: "SELECT COUNT(pid)",
			want: []SelectExpr{
				SelectAggregate{
					Func:  Count,
					Field: "pid",
				},
			},
		},
		{
			name:    "unclosed count",
			input:   "SELECT COUNT(",
			wantErr: true,
		},
		{
			name:    "trailing comma",
			input:   "SELECT level,",
			wantErr: true,
		},
		{
			name:    "empty select",
			input:   "SELECT",
			wantErr: true,
		},
		{
			name:    "double comma",
			input:   "SELECT level,,pid",
			wantErr: true,
		},
		{
			name:  "case insensitive",
			input: "sElEcT cOuNt(*)",
			want: []SelectExpr{
				SelectAggregate{
					Func:  Count,
					Field: "*",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer()

			lexemes, err := lexer.parse(tt.input)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("lexer.Parse() error = %v", err)
			}

			syntax := newSyntaxes(lexemes)

			got, err := syntax.query()

			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"Query(%q) expected error, got nil",
						tt.input,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("Query(%q) error = %v", tt.input, err)
			}

			if !reflect.DeepEqual(got.Projection, tt.want) {
				t.Errorf(
					"Query(%q) projection:\ngot  = %#v\nwant = %#v",
					tt.input,
					got.Projection,
					tt.want,
				)
			}
		})
	}
}
