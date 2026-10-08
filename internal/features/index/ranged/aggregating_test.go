package ranged

import (
	"errors"
	"math"
	"testing"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

func TestSumAvg(t *testing.T) {
	tests := []struct {
		name     string
		values   []domain.IntValue
		selected []domain.RecordID
		wantSum  float64
		wantAvg  float64
	}{
		{
			name:     "integers",
			values:   []domain.IntValue{10, 20, 30},
			selected: []domain.RecordID{1, 2, 3},
			wantSum:  60,
			wantAvg:  20,
		},
		{
			name:     "duplicates",
			values:   []domain.IntValue{10, 10, 10, 20},
			selected: []domain.RecordID{1, 2, 3, 4},
			wantSum:  50,
			wantAvg:  12.5,
		},
		{
			name:     "filtered",
			values:   []domain.IntValue{10, 20, 30},
			selected: []domain.RecordID{1, 3},
			wantSum:  40,
			wantAvg:  20,
		},
		{
			name:     "negative values",
			values:   []domain.IntValue{-10, 20, -30},
			selected: []domain.RecordID{1, 2, 3},
			wantSum:  -20,
			wantAvg:  -20.0 / 3,
		},
		{
			name:     "single value",
			values:   []domain.IntValue{42},
			selected: []domain.RecordID{1},
			wantSum:  42,
			wantAvg:  42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := New[domain.IntValue](func(a, b domain.IntValue) int {
				switch {
				case a < b:
					return -1
				case a > b:
					return 1
				default:
					return 0
				}
			})

			for j, value := range tt.values {
				err := idx.Add(value, domain.RecordID(j+1))
				if err != nil {
					t.Fatalf("Add: %v", err)
				}
			}

			posting := structs.NewPostingLists(len(tt.selected))
			for _, id := range tt.selected {
				posting.Add(id)
			}

			sum, err := idx.Sum(posting)
			if err != nil {
				t.Fatalf("Sum: %v", err)
			}

			avg, err := idx.Avg(posting)
			if err != nil {
				t.Fatalf("Avg: %v", err)
			}

			assertFloatEqual(t, sum, tt.wantSum)
			assertFloatEqual(t, avg, tt.wantAvg)
		})
	}
}

func assertFloatEqual(t *testing.T, got domain.Value, want float64) {
	t.Helper()

	value, ok := got.(domain.FloatValue)
	if !ok {
		t.Fatalf("expected FloatValue, got %T", got)
	}

	if math.Abs(float64(value)-want) > 1e-9 {
		t.Errorf("got %v, want %v", value, want)
	}
}

func TestSumAvgFloat(t *testing.T) {
	idx := New[domain.FloatValue](func(a, b domain.FloatValue) int {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	})

	values := []domain.FloatValue{1.5, 2.5, 3.0}

	posting := structs.NewPostingLists(len(values))

	for j, value := range values {
		id := domain.RecordID(j + 1)

		if err := idx.Add(value, id); err != nil {
			t.Fatalf("Add: %v", err)
		}

		posting.Add(id)
	}

	sum, err := idx.Sum(posting)
	if err != nil {
		t.Fatalf("Sum: %v", err)
	}

	avg, err := idx.Avg(posting)
	if err != nil {
		t.Fatalf("Avg: %v", err)
	}

	assertFloatEqual(t, sum, 7)
	assertFloatEqual(t, avg, 7.0/3)
}

func TestMinMax(t *testing.T) {
	tests := []struct {
		name     string
		selected []domain.RecordID
		wantMin  domain.IntValue
		wantMax  domain.IntValue
	}{
		{
			name:     "all records",
			selected: []domain.RecordID{1, 2, 3, 4, 5},
			wantMin:  10,
			wantMax:  50,
		},
		{
			name:     "filtered",
			selected: []domain.RecordID{2, 3, 4},
			wantMin:  20,
			wantMax:  40,
		},
		{
			name:     "single record",
			selected: []domain.RecordID{3},
			wantMin:  30,
			wantMax:  30,
		},
		{
			name:     "non consecutive records",
			selected: []domain.RecordID{2, 5},
			wantMin:  20,
			wantMax:  50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := New[domain.IntValue](func(a, b domain.IntValue) int {
				switch {
				case a < b:
					return -1
				case a > b:
					return 1
				default:
					return 0
				}
			})

			values := []domain.IntValue{10, 20, 30, 40, 50}

			for j, value := range values {
				if err := idx.Add(value, domain.RecordID(j+1)); err != nil {
					t.Fatalf("Add: %v", err)
				}
			}

			posting := structs.NewPostingLists(len(tt.selected))
			for _, id := range tt.selected {
				posting.Add(id)
			}

			minValue, err := idx.Min(posting)
			if err != nil {
				t.Fatalf("Min: %v", err)
			}

			maxValue, err := idx.Max(posting)
			if err != nil {
				t.Fatalf("Max: %v", err)
			}

			if minValue != tt.wantMin {
				t.Errorf("Min: got %v, want %v", minValue, tt.wantMin)
			}

			if maxValue != tt.wantMax {
				t.Errorf("Max: got %v, want %v", maxValue, tt.wantMax)
			}
		})
	}
}

func TestAggregationEmpty(t *testing.T) {
	idx := New[domain.IntValue](func(a, b domain.IntValue) int {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	})

	if err := idx.Add(domain.IntValue(10), 1); err != nil {
		t.Fatalf("Add: %v", err)
	}

	tests := []struct {
		name string
		fn   func(*structs.PostingList) (domain.Value, error)
	}{
		{"Sum", idx.Sum},
		{"Avg", idx.Avg},
		{"Min", idx.Min},
		{"Max", idx.Max},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posting := structs.NewPostingLists(0)

			_, err := tt.fn(posting)

			if !errors.Is(err, common.ErrRecordNotFound) {
				t.Errorf(
					"expected ErrRecordNotFound, got %v",
					err,
				)
			}
		})
	}
}

func TestAggregationUnsupportedType(t *testing.T) {
	idx := New[domain.TimeValue](func(a, b domain.TimeValue) int {
		return time.Time(a).Compare(time.Time(b))
	})

	value := domain.TimeValue(time.Now())

	if err := idx.Add(value, 1); err != nil {
		t.Fatalf("Add: %v", err)
	}

	posting := structs.NewPostingLists(1)
	posting.Add(1)

	tests := []struct {
		name string
		fn   func(*structs.PostingList) (domain.Value, error)
	}{
		{"Sum", idx.Sum},
		{"Avg", idx.Avg},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fn(posting)

			if !errors.Is(err, common.ErrUnsupportedType) {
				t.Errorf(
					"expected ErrUnsupportedType, got %v",
					err,
				)
			}
		})
	}
}
