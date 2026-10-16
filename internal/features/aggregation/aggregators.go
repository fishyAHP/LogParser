package aggregation

import (
	"errors"
	"fmt"
	"math"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

var (
	ErrUnexpectedType = errors.New("unexpected type for aggregator")
)

type SumAggregator struct {
	sum domain.Value
	typ domain.DataType
}

func NewSumAggregator(
	typ domain.DataType,
) (*SumAggregator, error) {
	switch typ {
	case domain.IntType:
		return &SumAggregator{
			sum: domain.IntValue(0),
			typ: domain.IntType,
		}, nil
	case domain.FloatType:
		return &SumAggregator{
			sum: domain.FloatValue(0),
			typ: domain.FloatType,
		}, nil
	default:
		return nil, fmt.Errorf(
			"unsupported type for sum aggregator: %v",
			typ,
		)
	}
}

func (s *SumAggregator) Add(
	value domain.Value,
) error {
	switch v := value.(type) {
	case domain.FloatValue:
		if s.typ != domain.FloatType {
			return fmt.Errorf(
				"%w: %v",
				ErrUnexpectedType,
				s.typ,
			)
		}
		sum := s.sum.(domain.FloatValue)
		next := sum + v
		if math.IsInf(float64(next), 0) ||
			math.IsNaN(float64(next)) {
			return fmt.Errorf("SUM overflow")
		}

		s.sum = next
	case domain.IntValue:
		if s.typ != domain.IntType {
			return fmt.Errorf(
				"%w: %v",
				ErrUnexpectedType,
				s.typ,
			)
		}

		sum := s.sum.(domain.IntValue)

		if (v > 0 && sum > domain.IntValue(math.MaxInt64)-v) ||
			(v < 0 && sum < domain.IntValue(math.MinInt64)-v) {
			return errors.New(
				"SUM overflow",
			)
		}

		sum += v
		s.sum = sum
	default:
		return fmt.Errorf(
			"%w: %T",
			ErrUnexpectedType,
			v,
		)
	}

	return nil
}

func (s *SumAggregator) Result() (domain.Value, error) {
	return s.sum, nil
}

type AvgAggregator struct {
	typ   domain.DataType
	avg   domain.Value
	count int
}

func NewAvgAggregator(
	typ domain.DataType,
) (*AvgAggregator, error) {
	switch typ {
	case domain.IntType:
		return &AvgAggregator{
			avg: domain.FloatValue(0),
			typ: domain.IntType,
		}, nil
	case domain.FloatType:
		return &AvgAggregator{
			avg: domain.FloatValue(0),
			typ: domain.FloatType,
		}, nil
	case domain.TimeType:
		return &AvgAggregator{
			avg: domain.TimeValue(time.Time{}),
			typ: domain.TimeType,
		}, nil
	default:
		return nil, fmt.Errorf(
			"%w: %v",
			ErrUnexpectedType,
			typ,
		)
	}
}

func (a *AvgAggregator) Add(
	value domain.Value,
) error {
	switch v := value.(type) {
	case domain.IntValue:
		if a.typ != domain.IntType {
			return fmt.Errorf(
				"%w: want %v, got %T",
				ErrUnexpectedType,
				a.typ,
				v,
			)
		}

		nextCount := a.count + 1
		nextAvg, err := updateAvgNumbers(
			a.avg.(domain.FloatValue),
			domain.FloatValue(v), nextCount,
		)
		if err != nil {
			return err
		}

		a.avg = nextAvg
		a.count = nextCount
	case domain.FloatValue:
		if a.typ != domain.FloatType {
			return fmt.Errorf(
				"%w: want %v, got %T",
				ErrUnexpectedType,
				a.typ,
				v,
			)
		}

		nextCount := a.count + 1
		nextAvg, err := updateAvgNumbers(
			a.avg.(domain.FloatValue),
			v, nextCount,
		)
		if err != nil {
			return err
		}

		a.avg = nextAvg
		a.count = nextCount
	case domain.TimeValue:
		if a.typ != domain.TimeType {
			return fmt.Errorf(
				"%w: want %v, got %T",
				ErrUnexpectedType,
				a.typ,
				v,
			)
		}

		if a.count == 0 {
			a.avg = v
			a.count = 1
			return nil
		}

		avg := time.Time(a.avg.(domain.TimeValue))
		diff := time.Time(v).Sub(avg)

		nextCount := a.count + 1
		nextAvg := avg.Add(diff / time.Duration(nextCount))

		a.avg = domain.TimeValue(nextAvg)
		a.count = nextCount
	default:
		return fmt.Errorf(
			"%w: %T",
			ErrUnexpectedType,
			v,
		)
	}

	return nil
}

func updateAvgNumbers(
	avg, value domain.FloatValue,
	count int,
) (domain.FloatValue, error) {
	if math.IsNaN(float64(avg)) ||
		math.IsInf(float64(avg), 0) {
		return 0, errors.New("invalid current AVG")
	}

	if math.IsInf(float64(value), 0) ||
		math.IsNaN(float64(value)) {
		return 0, errors.New("invalid AVG value")
	}

	var next domain.FloatValue
	n := domain.FloatValue(count)

	if math.Signbit(float64(avg)) != math.Signbit(float64(value)) {
		next = avg + value/n - avg/n
	} else {
		next = avg + (value-avg)/n
	}

	if math.IsInf(float64(next), 0) ||
		math.IsNaN(float64(next)) {
		return 0, errors.New("AVG overflow")
	}

	return next, nil
}

func (a *AvgAggregator) Result() (domain.Value, error) {
	if a.count == 0 {
		return nil, errors.New(
			"empty result",
		)
	}

	return a.avg, nil
}

type ExtremumAggregator struct {
	res     domain.Value
	typ     domain.DataType
	started bool
	compare func(a, b domain.Value) domain.Value
}

func newExtremumAggregator(
	typ domain.DataType,
) (*ExtremumAggregator, error) {
	switch typ {
	case domain.IntType, domain.FloatType,
		domain.StringType, domain.TimeType:
		return &ExtremumAggregator{
			typ: typ,
		}, nil
	default:
		return nil, fmt.Errorf(
			"%w: %v",
			ErrUnexpectedType,
			typ,
		)
	}
}

func NewMinAggregator(
	typ domain.DataType,
) (*ExtremumAggregator, error) {
	aggr, err := newExtremumAggregator(typ)
	if err != nil {
		return nil, err
	}

	switch typ {
	case domain.TimeType:
		aggr.compare = func(a, b domain.Value) domain.Value {
			varA := a.(domain.TimeValue)
			varB := b.(domain.TimeValue)
			if time.Time(varB).Before(time.Time(varA)) {
				return b
			}
			return a
		}
	case domain.IntType:
		aggr.compare = func(a, b domain.Value) domain.Value {
			return min(a.(domain.IntValue), b.(domain.IntValue))
		}
	case domain.FloatType:
		aggr.compare = func(a, b domain.Value) domain.Value {
			return min(a.(domain.FloatValue), b.(domain.FloatValue))
		}
	case domain.StringType:
		aggr.compare = func(a, b domain.Value) domain.Value {
			return min(a.(domain.StringValue), b.(domain.StringValue))
		}
	}

	return aggr, nil
}

func NewMaxAggregator(
	typ domain.DataType,
) (*ExtremumAggregator, error) {
	aggr, err := newExtremumAggregator(typ)
	if err != nil {
		return nil, err
	}

	switch typ {
	case domain.TimeType:
		aggr.compare = func(a, b domain.Value) domain.Value {
			varA := a.(domain.TimeValue)
			varB := b.(domain.TimeValue)
			if time.Time(varB).After(time.Time(varA)) {
				return b
			}
			return a
		}
	case domain.IntType:
		aggr.compare = func(a, b domain.Value) domain.Value {
			return max(a.(domain.IntValue), b.(domain.IntValue))
		}
	case domain.FloatType:
		aggr.compare = func(a, b domain.Value) domain.Value {
			return max(a.(domain.FloatValue), b.(domain.FloatValue))
		}
	case domain.StringType:
		aggr.compare = func(a, b domain.Value) domain.Value {
			return max(a.(domain.StringValue), b.(domain.StringValue))
		}
	}

	return aggr, nil
}

func (m *ExtremumAggregator) Add(
	value domain.Value,
) error {
	if value == nil {
		return fmt.Errorf(
			"%w: nil value",
			ErrUnexpectedType,
		)
	}
	typ := value.Type()
	if m.typ != typ {
		return fmt.Errorf(
			"%w: want %v, got %v",
			ErrUnexpectedType,
			m.typ,
			typ,
		)
	}

	if v, ok := value.(domain.FloatValue); ok {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return errors.New("invalid extremum value")
		}
	}

	if !m.started {
		m.res = value
		m.started = true
		return nil
	}

	m.res = m.compare(m.res, value)

	return nil
}

func (m *ExtremumAggregator) Result() (domain.Value, error) {
	if !m.started {
		return nil, errors.New(
			"empty result",
		)
	}

	return m.res, nil
}
