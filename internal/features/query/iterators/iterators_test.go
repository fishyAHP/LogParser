package iterators

import (
	"errors"
	"reflect"
	"testing"
)

func testSource(values ...string) Rows {
	return func(yield func([]byte, error) bool) {
		for _, value := range values {
			if !yield([]byte(value), nil) {
				return
			}
		}
	}
}

func collect(source Rows) ([]string, error) {
	var result []string

	for data, err := range source {
		if err != nil {
			return nil, err
		}

		result = append(result, string(data))
	}

	return result, nil
}

func TestOffset(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		offset uint64
		want   []string
	}{
		{
			name:   "zero offset",
			input:  []string{"a", "b", "c"},
			offset: 0,
			want:   []string{"a", "b", "c"},
		},
		{
			name:   "skip one",
			input:  []string{"a", "b", "c"},
			offset: 1,
			want:   []string{"b", "c"},
		},
		{
			name:   "skip two",
			input:  []string{"a", "b", "c"},
			offset: 2,
			want:   []string{"c"},
		},
		{
			name:   "offset equals length",
			input:  []string{"a", "b", "c"},
			offset: 3,
		},
		{
			name:   "offset exceeds length",
			input:  []string{"a", "b", "c"},
			offset: 10,
		},
		{
			name:   "empty source",
			input:  nil,
			offset: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := testSource(tt.input...)
			got, err := collect(Offset(source, tt.offset))

			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimit(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		limit uint64
		want  []string
	}{
		{
			name:  "zero limit",
			input: []string{"a", "b", "c"},
			limit: 0,
		},
		{
			name:  "limit one",
			input: []string{"a", "b", "c"},
			limit: 1,
			want:  []string{"a"},
		},
		{
			name:  "limit two",
			input: []string{"a", "b", "c"},
			limit: 2,
			want:  []string{"a", "b"},
		},
		{
			name:  "limit equals length",
			input: []string{"a", "b", "c"},
			limit: 3,
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "limit exceeds length",
			input: []string{"a", "b", "c"},
			limit: 10,
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "empty source",
			input: nil,
			limit: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collect(Limit(testSource(tt.input...), tt.limit))
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{
			name:  "empty source",
			input: nil,
			want:  `{"COUNT(*)":0}`,
		},
		{
			name:  "one record",
			input: []string{"a"},
			want:  `{"COUNT(*)":1}`,
		},
		{
			name:  "multiple records",
			input: []string{"a", "b", "c"},
			want:  `{"COUNT(*)":3}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collect(Count(testSource(tt.input...)))
			if err != nil {
				t.Fatal(err)
			}

			want := []string{tt.want}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestPaginationPipeline(t *testing.T) {
	source := testSource("a", "b", "c", "d", "e")

	rows := Offset(source, 2)
	rows = Limit(rows, 2)

	got, err := collect(rows)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"c", "d"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCountPipeline(t *testing.T) {
	source := testSource("a", "b", "c")

	rows := Count(source)
	rows = Limit(rows, 10)

	got, err := collect(rows)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{`{"COUNT(*)":3}`}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestIteratorErrorPropagation(t *testing.T) {
	expected := errors.New("storage failure")

	source := func(yield func([]byte, error) bool) {
		yield(nil, expected)
	}

	tests := []struct {
		name string
		wrap func(Rows) Rows
	}{
		{"offset", func(r Rows) Rows { return Offset(r, 2) }},
		{"limit", func(r Rows) Rows { return Limit(r, 2) }},
		{"count", Count},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := collect(tt.wrap(source))

			if !errors.Is(err, expected) {
				t.Errorf("got error %v, want %v", err, expected)
			}
		})
	}
}

func TestIteratorsRepeatedTraversal(t *testing.T) {
	tests := []struct {
		name string
		rows Rows
		want []string
	}{
		{
			name: "limit",
			rows: Limit(testSource("a", "b", "c"), 2),
			want: []string{"a", "b"},
		},
		{
			name: "offset",
			rows: Offset(testSource("a", "b", "c"), 1),
			want: []string{"b", "c"},
		},
		{
			name: "count",
			rows: Count(testSource("a", "b", "c")),
			want: []string{`{"COUNT(*)":3}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 2; i++ {
				got, err := collect(tt.rows)
				if err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf(
						"traversal %d: got %v, want %v",
						i+1,
						got,
						tt.want,
					)
				}
			}
		})
	}
}

func TestLimitStopsSource(t *testing.T) {
	var produced int

	source := func(yield func([]byte, error) bool) {
		for i := 0; i < 100; i++ {
			produced++

			if !yield([]byte("record"), nil) {
				return
			}
		}
	}

	got, err := collect(Limit(source, 2))
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}

	if produced != 2 {
		t.Errorf(
			"source produced %d records, expected exactly 2",
			produced,
		)
	}
}
