package aggregation

import (
	"fmt"
	"math"
	"testing"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

func defaultSumAggregator(
	t *testing.T,
	typ domain.DataType,
) *SumAggregator {
	aggr, err := NewSumAggregator(typ)
	if err != nil {
		t.Fatalf("new sum aggregator: %v", err)
	}

	return aggr
}

func defaultAvgAggregator(
	t *testing.T,
	typ domain.DataType,
) *AvgAggregator {
	aggr, err := NewAvgAggregator(typ)
	if err != nil {
		t.Fatalf("new avg aggregator: %v", err)
	}

	return aggr
}

func defaultMinAggregator(
	t *testing.T,
	typ domain.DataType,
) *ExtremumAggregator {
	aggr, err := NewMinAggregator(typ)
	if err != nil {
		t.Fatalf("new avg aggregator: %v", err)
	}

	return aggr
}

func defaultMaxAggregator(
	t *testing.T,
	typ domain.DataType,
) *ExtremumAggregator {
	aggr, err := NewMinAggregator(typ)
	if err != nil {
		t.Fatalf("new avg aggregator: %v", err)
	}

	return aggr
}

func TestSumAggregatorPositive(t *testing.T) {
	var cases = []struct {
		name   string
		typ    domain.DataType
		adds   []domain.Value
		result domain.Value
	}{
		{
			name:   "add positive ints",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(10), domain.IntValue(20), domain.IntValue(30)},
			result: domain.IntValue(60),
		},
		{
			name:   "add negative ints",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(-10), domain.IntValue(-20), domain.IntValue(-30)},
			result: domain.IntValue(-60),
		},
		{
			name:   "add positive and negative ints",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(100), domain.IntValue(-40), domain.IntValue(20)},
			result: domain.IntValue(80),
		},
		{
			name:   "add zero ints",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(0), domain.IntValue(0), domain.IntValue(0)},
			result: domain.IntValue(0),
		},
		{
			name:   "add positive floats",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(1.5), domain.FloatValue(2.25), domain.FloatValue(3.25)},
			result: domain.FloatValue(7.0),
		},
		{
			name:   "add negative floats",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(-1.5), domain.FloatValue(-2.25), domain.FloatValue(-3.25)},
			result: domain.FloatValue(-7.0),
		},
		{
			name:   "add positive and negative floats",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(10.5), domain.FloatValue(-4.25), domain.FloatValue(2.75)},
			result: domain.FloatValue(9.0),
		},
		{
			name:   "add zero floats",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(0), domain.FloatValue(0.0), domain.FloatValue(0)},
			result: domain.FloatValue(0),
		},
		{
			name:   "add large ints without overflow",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(math.MaxInt64 - 100), domain.IntValue(100)},
			result: domain.IntValue(math.MaxInt64),
		},
		{
			name:   "empty sequence",
			typ:    domain.IntType,
			adds:   []domain.Value{},
			result: domain.IntValue(0),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			aggr := defaultSumAggregator(t, tt.typ)

			for _, add := range tt.adds {
				if err := aggr.Add(add); err != nil {
					t.Fatalf("aggregate add: %v", err)
				}
			}

			res, err := aggr.Result()
			if err != nil {
				t.Fatalf("aggregate result: %v", err)
			}

			if res != tt.result {
				t.Fatalf("want %v, got %v", tt.result, res)
			}
		})
	}
}

func TestSumAggregatorNegative(t *testing.T) {
	var cases = []struct {
		name string
		typ  domain.DataType
		adds []domain.Value
	}{
		{
			name: "float value in int aggregator",
			typ:  domain.IntType,
			adds: []domain.Value{
				domain.FloatValue(10.5),
			},
		},
		{
			name: "int value in float aggregator",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.IntValue(10),
			},
		},
		{
			name: "string value in int aggregator",
			typ:  domain.IntType,
			adds: []domain.Value{
				domain.StringValue("100"),
			},
		},
		{
			name: "string value in float aggregator",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.StringValue("10.5"),
			},
		},
		{
			name: "bool value in int aggregator",
			typ:  domain.IntType,
			adds: []domain.Value{
				domain.BoolValue(true),
			},
		},
		{
			name: "nil value",
			typ:  domain.IntType,
			adds: []domain.Value{
				nil,
			},
		},
		{
			name: "positive int overflow",
			typ:  domain.IntType,
			adds: []domain.Value{
				domain.IntValue(math.MaxInt64),
				domain.IntValue(1),
			},
		},
		{
			name: "negative int overflow",
			typ:  domain.IntType,
			adds: []domain.Value{
				domain.IntValue(math.MinInt64),
				domain.IntValue(-1),
			},
		},
		{
			name: "positive float overflow",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.FloatValue(math.MaxFloat64),
				domain.FloatValue(math.MaxFloat64),
			},
		},
		{
			name: "negative float overflow",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.FloatValue(-math.MaxFloat64),
				domain.FloatValue(-math.MaxFloat64),
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			aggr := defaultSumAggregator(t, tt.typ)
			var err error
			for _, add := range tt.adds {
				if err = aggr.Add(add); err != nil {
					break
				}
			}

			if err == nil {
				t.Fatal("expect error, but got nil")
			}
		})
	}
}

func TestAvgAggregatorPositive(t *testing.T) {
	var cases = []struct {
		name   string
		typ    domain.DataType
		adds   []domain.Value
		result domain.Value
	}{
		// INT
		{
			name:   "int single value",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(10)},
			result: domain.FloatValue(10),
		},
		{
			name:   "int positive values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(10), domain.IntValue(20), domain.IntValue(30)},
			result: domain.FloatValue(20),
		},
		{
			name:   "int negative values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(-10), domain.IntValue(-20), domain.IntValue(-30)},
			result: domain.FloatValue(-20),
		},
		{
			name:   "int mixed signs",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(-10), domain.IntValue(20), domain.IntValue(50)},
			result: domain.FloatValue(20),
		},
		{
			name:   "int all zeros",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(0), domain.IntValue(0), domain.IntValue(0)},
			result: domain.FloatValue(0),
		},
		{
			name:   "int fractional average",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(1), domain.IntValue(2)},
			result: domain.FloatValue(1.5),
		},
		{
			name:   "int identical values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(42), domain.IntValue(42), domain.IntValue(42)},
			result: domain.FloatValue(42),
		},
		{
			name:   "int symmetric values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(-100), domain.IntValue(0), domain.IntValue(100)},
			result: domain.FloatValue(0),
		},
		{
			name:   "int large positive values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(1_000_000_000), domain.IntValue(3_000_000_000)},
			result: domain.FloatValue(2_000_000_000),
		},
		{
			name:   "int max and min",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(math.MinInt64), domain.IntValue(math.MaxInt64)},
			result: domain.FloatValue(0),
		},

		// FLOAT
		{
			name:   "float single value",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(2.5)},
			result: domain.FloatValue(2.5),
		},
		{
			name:   "float positive values",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(1.5), domain.FloatValue(2.5), domain.FloatValue(3.5)},
			result: domain.FloatValue(2.5),
		},
		{
			name:   "float negative values",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(-1.5), domain.FloatValue(-2.5), domain.FloatValue(-3.5)},
			result: domain.FloatValue(-2.5),
		},
		{
			name:   "float mixed signs",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(-5.5), domain.FloatValue(0.5), domain.FloatValue(5)},
			result: domain.FloatValue(0),
		},
		{
			name:   "float all zeros",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(0), domain.FloatValue(0), domain.FloatValue(0)},
			result: domain.FloatValue(0),
		},
		{
			name:   "float identical values",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(7.25), domain.FloatValue(7.25), domain.FloatValue(7.25)},
			result: domain.FloatValue(7.25),
		},
		{
			name:   "float small values",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(0.125), domain.FloatValue(0.375)},
			result: domain.FloatValue(0.25),
		},
		{
			name:   "float large positive values",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(1e100), domain.FloatValue(3e100)},
			result: domain.FloatValue(2e100),
		},
		{
			name:   "float opposite extremes",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(-math.MaxFloat64), domain.FloatValue(math.MaxFloat64)},
			result: domain.FloatValue(0),
		},
		{
			name:   "float mixed zero and fractions",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(0), domain.FloatValue(0.5), domain.FloatValue(1)},
			result: domain.FloatValue(0.5),
		},

		// TIME
		{
			name:   "time single value",
			typ:    domain.TimeType,
			adds:   []domain.Value{domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC))},
			result: domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)),
		},
		{
			name: "time two hours",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)),
		},
		{
			name: "time three values",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 16, 17, 0, 0, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2026, 10, 16, 13, 0, 0, 0, time.UTC)),
		},
		{
			name: "time identical timestamps",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 10, 16, 12, 30, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 16, 12, 30, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 16, 12, 30, 0, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2026, 10, 16, 12, 30, 0, 0, time.UTC)),
		},
		{
			name: "time across midnight",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 10, 16, 22, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 17, 2, 0, 0, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)),
		},
		{
			name: "time across months",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)),
		},
		{
			name: "time across years",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		},
		{
			name: "time minute precision",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 10, 16, 12, 10, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 16, 12, 20, 0, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2026, 10, 16, 12, 15, 0, 0, time.UTC)),
		},
		{
			name: "time second precision",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 10, 0, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 30, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 20, 0, time.UTC)),
		},
		{
			name: "time millisecond precision",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 0, 100_000_000, time.UTC)),
				domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 0, 300_000_000, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2026, 10, 16, 12, 0, 0, 200_000_000, time.UTC)),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			aggr := defaultAvgAggregator(t, tt.typ)

			for _, add := range tt.adds {
				if err := aggr.Add(add); err != nil {
					t.Fatalf("aggregate add: %v", err)
				}
			}

			res, err := aggr.Result()
			if err != nil {
				t.Fatalf("aggregate result: %v", err)
			}

			switch got := res.(type) {
			case domain.FloatValue:
				want := tt.result.(domain.FloatValue)

				if !almostEqual(float64(got), float64(want)) {
					t.Fatalf("want %v, got %v", want, got)
				}

			case domain.TimeValue:
				want := time.Time(tt.result.(domain.TimeValue))

				if !time.Time(got).Equal(want) {
					t.Fatalf("want %v, got %v", want, got)
				}
			}
		})
	}
}

func almostEqual(a, b float64) bool {
	const (
		absTol = 1e-12
		relTol = 1e-12
	)

	if math.IsNaN(a) || math.IsNaN(b) {
		return false
	}

	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return a == b
	}

	diff := math.Abs(a - b)

	return diff <= math.Max(
		absTol,
		relTol*math.Max(math.Abs(a), math.Abs(b)),
	)
}

func TestAvgAggregatorNegative(t *testing.T) {
	cases := []struct {
		name string
		typ  domain.DataType
		adds []domain.Value
	}{
		{
			name: "float value in int aggregator",
			typ:  domain.IntType,
			adds: []domain.Value{
				domain.FloatValue(10.5),
			},
		},
		{
			name: "int value in float aggregator",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.IntValue(10),
			},
		},
		{
			name: "time value in int aggregator",
			typ:  domain.IntType,
			adds: []domain.Value{
				domain.TimeValue(time.Now()),
			},
		},
		{
			name: "time value in float aggregator",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.TimeValue(time.Now()),
			},
		},
		{
			name: "int value in time aggregator",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.IntValue(100),
			},
		},
		{
			name: "float value in time aggregator",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.FloatValue(100.5),
			},
		},
		{
			name: "string value",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.StringValue("100"),
			},
		},
		{
			name: "bool value",
			typ:  domain.IntType,
			adds: []domain.Value{
				domain.BoolValue(true),
			},
		},
		{
			name: "nil value",
			typ:  domain.TimeType,
			adds: []domain.Value{
				nil,
			},
		},
		{
			name: "NaN value",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.FloatValue(math.NaN()),
			},
		},
		{
			name: "positive infinity",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.FloatValue(math.Inf(1)),
			},
		},
		{
			name: "negative infinity",
			typ:  domain.FloatType,
			adds: []domain.Value{
				domain.FloatValue(math.Inf(-1)),
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			aggr, err := NewAvgAggregator(tt.typ)
			if err != nil {
				t.Fatalf("new avg aggregator: %v", err)
			}

			var gotErr error

			for _, value := range tt.adds {
				gotErr = aggr.Add(value)
				if gotErr != nil {
					break
				}
			}

			if gotErr == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestAvgAggregatorStateAfterError(t *testing.T) {
	aggr, err := NewAvgAggregator(domain.IntType)
	if err != nil {
		t.Fatal(err)
	}

	if err := aggr.Add(domain.IntValue(10)); err != nil {
		t.Fatal(err)
	}

	if err := aggr.Add(domain.FloatValue(100.5)); err == nil {
		t.Fatal("expected type error")
	}

	if err := aggr.Add(domain.IntValue(20)); err != nil {
		t.Fatal(err)
	}

	got, err := aggr.Result()
	if err != nil {
		t.Fatal(err)
	}

	want := domain.FloatValue(15)

	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestAvgAggregatorEmpty(t *testing.T) {
	for _, typ := range []domain.DataType{
		domain.IntType,
		domain.FloatType,
		domain.TimeType,
	} {
		t.Run(fmt.Sprint(typ), func(t *testing.T) {
			aggr, err := NewAvgAggregator(typ)
			if err != nil {
				t.Fatal(err)
			}

			_, err = aggr.Result()
			if err == nil {
				t.Fatal("expected error for empty sequence")
			}
		})
	}
}

func TestExtremumAggregatorPositive(t *testing.T) {
	var minCases = []struct {
		name   string
		typ    domain.DataType
		adds   []domain.Value
		result domain.Value
	}{
		{
			name:   "int positive values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(10), domain.IntValue(20), domain.IntValue(30)},
			result: domain.IntValue(10),
		},
		{
			name:   "int negative values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(-10), domain.IntValue(-20), domain.IntValue(-30)},
			result: domain.IntValue(-30),
		},
		{
			name:   "int mixed signs",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(-100), domain.IntValue(0), domain.IntValue(100)},
			result: domain.IntValue(-100),
		},
		{
			name:   "int single value",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(42)},
			result: domain.IntValue(42),
		},
		{
			name:   "int extreme values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(math.MaxInt64), domain.IntValue(math.MinInt64)},
			result: domain.IntValue(math.MinInt64),
		},
		{
			name:   "float positive values",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(1.5), domain.FloatValue(0.5), domain.FloatValue(3.5)},
			result: domain.FloatValue(0.5),
		},
		{
			name:   "float negative values",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(-1.5), domain.FloatValue(-5.5), domain.FloatValue(-2.5)},
			result: domain.FloatValue(-5.5),
		},
		{
			name:   "string lexicographic",
			typ:    domain.StringType,
			adds:   []domain.Value{domain.StringValue("banana"), domain.StringValue("apple"), domain.StringValue("cherry")},
			result: domain.StringValue("apple"),
		},
		{
			name:   "string empty value",
			typ:    domain.StringType,
			adds:   []domain.Value{domain.StringValue("hello"), domain.StringValue(""), domain.StringValue("world")},
			result: domain.StringValue(""),
		},
		{
			name: "time earliest",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2025, 10, 17, 12, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2027, 10, 17, 12, 0, 0, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2025, 10, 17, 12, 0, 0, 0, time.UTC)),
		},
	}

	var maxCases = []struct {
		name   string
		typ    domain.DataType
		adds   []domain.Value
		result domain.Value
	}{
		{
			name:   "int positive values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(10), domain.IntValue(20), domain.IntValue(30)},
			result: domain.IntValue(30),
		},
		{
			name:   "int negative values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(-10), domain.IntValue(-20), domain.IntValue(-30)},
			result: domain.IntValue(-10),
		},
		{
			name:   "int mixed signs",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(-100), domain.IntValue(0), domain.IntValue(100)},
			result: domain.IntValue(100),
		},
		{
			name:   "int single value",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(42)},
			result: domain.IntValue(42),
		},
		{
			name:   "int extreme values",
			typ:    domain.IntType,
			adds:   []domain.Value{domain.IntValue(math.MaxInt64), domain.IntValue(math.MinInt64)},
			result: domain.IntValue(math.MaxInt64),
		},
		{
			name:   "float positive values",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(1.5), domain.FloatValue(0.5), domain.FloatValue(3.5)},
			result: domain.FloatValue(3.5),
		},
		{
			name:   "float negative values",
			typ:    domain.FloatType,
			adds:   []domain.Value{domain.FloatValue(-1.5), domain.FloatValue(-5.5), domain.FloatValue(-2.5)},
			result: domain.FloatValue(-1.5),
		},
		{
			name:   "string lexicographic",
			typ:    domain.StringType,
			adds:   []domain.Value{domain.StringValue("banana"), domain.StringValue("apple"), domain.StringValue("cherry")},
			result: domain.StringValue("cherry"),
		},
		{
			name:   "string empty value",
			typ:    domain.StringType,
			adds:   []domain.Value{domain.StringValue("hello"), domain.StringValue(""), domain.StringValue("world")},
			result: domain.StringValue("world"),
		},
		{
			name: "time latest",
			typ:  domain.TimeType,
			adds: []domain.Value{
				domain.TimeValue(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2025, 10, 17, 12, 0, 0, 0, time.UTC)),
				domain.TimeValue(time.Date(2027, 10, 17, 12, 0, 0, 0, time.UTC)),
			},
			result: domain.TimeValue(time.Date(2027, 10, 17, 12, 0, 0, 0, time.UTC)),
		},
	}

	for _, tt := range minCases {
		t.Run(tt.name, func(t *testing.T) {
			aggr, err := NewMinAggregator(tt.typ)
			if err != nil {
				t.Fatal(err)
			}

			for _, value := range tt.adds {
				if err := aggr.Add(value); err != nil {
					t.Fatal(err)
				}
			}

			got, err := aggr.Result()
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.result {
				t.Fatalf("want %v, got %v", tt.result, got)
			}
		})
	}

	for _, tt := range maxCases {
		t.Run(tt.name, func(t *testing.T) {
			aggr, err := NewMaxAggregator(tt.typ)
			if err != nil {
				t.Fatal(err)
			}

			for _, value := range tt.adds {
				if err := aggr.Add(value); err != nil {
					t.Fatal(err)
				}
			}

			got, err := aggr.Result()
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.result {
				t.Fatalf("want %v, got %v", tt.result, got)
			}
		})
	}
}

func TestExtremumAggregatorNegative(t *testing.T) {
	var negativeCases = []struct {
		name string
		typ  domain.DataType
		adds []domain.Value
	}{
		{
			name: "float in int aggregator",
			typ:  domain.IntType,
			adds: []domain.Value{domain.FloatValue(10.5)},
		},
		{
			name: "int in float aggregator",
			typ:  domain.FloatType,
			adds: []domain.Value{domain.IntValue(10)},
		},
		{
			name: "string in time aggregator",
			typ:  domain.TimeType,
			adds: []domain.Value{domain.StringValue("2026-10-17")},
		},
		{
			name: "time in string aggregator",
			typ:  domain.StringType,
			adds: []domain.Value{domain.TimeValue(time.Now())},
		},
		{
			name: "bool value",
			typ:  domain.IntType,
			adds: []domain.Value{domain.BoolValue(true)},
		},
		{
			name: "nil value",
			typ:  domain.IntType,
			adds: []domain.Value{nil},
		},
		{
			name: "NaN",
			typ:  domain.FloatType,
			adds: []domain.Value{domain.FloatValue(math.NaN())},
		},
		{
			name: "positive infinity",
			typ:  domain.FloatType,
			adds: []domain.Value{domain.FloatValue(math.Inf(1))},
		},
		{
			name: "negative infinity",
			typ:  domain.FloatType,
			adds: []domain.Value{domain.FloatValue(math.Inf(-1))},
		},
		{
			name: "invalid value after valid value",
			typ:  domain.IntType,
			adds: []domain.Value{
				domain.IntValue(10),
				domain.StringValue("invalid"),
			},
		},
	}

	for _, tt := range negativeCases {
		t.Run(tt.name, func(t *testing.T) {
			for _, constructor := range []struct {
				name string
				new  func(domain.DataType) (*ExtremumAggregator, error)
			}{
				{"MIN", NewMinAggregator},
				{"MAX", NewMaxAggregator},
			} {
				t.Run(constructor.name, func(t *testing.T) {
					aggr, err := constructor.new(tt.typ)
					if err != nil {
						t.Fatal(err)
					}

					var gotErr error
					for _, value := range tt.adds {
						gotErr = aggr.Add(value)
						if gotErr != nil {
							break
						}
					}

					if gotErr == nil {
						t.Fatal("expected error, got nil")
					}
				})
			}
		})
	}
}
