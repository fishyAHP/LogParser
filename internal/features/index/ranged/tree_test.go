package ranged

import (
	"reflect"
	"slices"
	"testing"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func TestRangeIterator(t *testing.T) {
	type record struct {
		id    domain.RecordID
		value domain.IntValue
	}

	tests := []struct {
		name    string
		records []record
		from    *domain.Bound
		to      *domain.Bound
		want    []domain.RecordID
	}{
		{
			name: "sorted by RecordID, not by field value",
			records: []record{
				{1, 400},
				{2, 300},
				{5, 100},
				{7, 200},
				{10, 100},
			},
			want: []domain.RecordID{1, 2, 5, 7, 10},
		},
		{
			name: "inclusive bounds",
			records: []record{
				{1, 40},
				{2, 20},
				{5, 10},
				{9, 30},
				{12, 50},
			},
			from: &domain.Bound{
				Value:     domain.IntValue(20),
				Inclusive: true,
			},
			to: &domain.Bound{
				Value:     domain.IntValue(40),
				Inclusive: true,
			},
			want: []domain.RecordID{1, 2, 9},
		},
		{
			name: "exclusive bounds",
			records: []record{
				{1, 10},
				{2, 20},
				{3, 30},
				{4, 40},
				{5, 50},
			},
			from: &domain.Bound{
				Value:     domain.IntValue(20),
				Inclusive: false,
			},
			to: &domain.Bound{
				Value:     domain.IntValue(40),
				Inclusive: false,
			},
			want: []domain.RecordID{3},
		},
		{
			name: "multiple records with the same value",
			records: []record{
				{1, 20},
				{3, 10},
				{6, 20},
				{8, 10},
				{12, 10},
			},
			want: []domain.RecordID{1, 3, 6, 8, 12},
		},
		{
			name: "lower bound only",
			records: []record{
				{1, 10},
				{2, 20},
				{3, 30},
				{4, 40},
			},
			from: &domain.Bound{
				Value:     domain.IntValue(25),
				Inclusive: true,
			},
			want: []domain.RecordID{3, 4},
		},
		{
			name: "upper bound only",
			records: []record{
				{1, 10},
				{2, 20},
				{3, 30},
				{4, 40},
			},
			to: &domain.Bound{
				Value:     domain.IntValue(25),
				Inclusive: true,
			},
			want: []domain.RecordID{1, 2},
		},
		{
			name: "single matching record",
			records: []record{
				{1, 10},
				{2, 20},
				{3, 30},
			},
			from: &domain.Bound{
				Value:     domain.IntValue(20),
				Inclusive: true,
			},
			to: &domain.Bound{
				Value:     domain.IntValue(20),
				Inclusive: true,
			},
			want: []domain.RecordID{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := New[domain.IntValue](
				func(a, b domain.IntValue) int {
					switch {
					case a < b:
						return -1
					case a > b:
						return 1
					default:
						return 0
					}
				},
			)

			for _, record := range tt.records {
				if err := idx.Add(record.value, record.id); err != nil {
					t.Fatalf(
						"Add(%v, %v): %v",
						record.value,
						record.id,
						err,
					)
				}
			}

			seq, err := idx.Range(tt.from, tt.to)
			if err != nil {
				t.Fatalf("Range: %v", err)
			}

			got := slices.Collect(seq)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}

			for i := 1; i < len(got); i++ {
				if got[i] <= got[i-1] {
					t.Errorf(
						"IDs must be unique and sorted: %v",
						got,
					)
					break
				}
			}

			again := slices.Collect(seq)

			if !reflect.DeepEqual(again, tt.want) {
				t.Errorf(
					"second traversal: got %v, want %v",
					again,
					tt.want,
				)
			}
		})
	}
}
