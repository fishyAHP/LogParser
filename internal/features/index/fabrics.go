package index

import (
	"cmp"
	"fmt"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/hash"
	"fishyAHP/LogParser.git/internal/features/index/ranged"
	"fishyAHP/LogParser.git/internal/features/index/text"
)

var (
	_ common.ExactIndex = (*hash.Index[domain.StringValue])(nil)
	_ common.ExactIndex = (*hash.Index[domain.BoolValue])(nil)
	_ common.ExactIndex = (*hash.Index[domain.IntValue])(nil)
	_ common.ExactIndex = (*hash.Index[domain.FloatValue])(nil)

	_ common.TextIndex = (*text.Index)(nil)

	_ common.RangeIndex = (*ranged.Index[domain.IntValue])(nil)
	_ common.RangeIndex = (*ranged.Index[domain.FloatValue])(nil)
	_ common.RangeIndex = (*ranged.Index[domain.TimeValue])(nil)
)

func newHashIndex(field domain.Field) (common.Index, error) {
	switch field.FieldType {
	case domain.StringType:
		return hash.New[domain.StringValue](), nil
	case domain.BoolType:
		return hash.New[domain.BoolValue](), nil
	case domain.IntType:
		return hash.New[domain.IntValue](), nil
	case domain.FloatType:
		return hash.New[domain.FloatValue](), nil
	default:
		return nil, fmt.Errorf(
			"hash index doesnt support %v type",
			field.FieldType,
		)
	}
}

func newRangeIndex(field domain.Field) (common.Index, error) {
	switch field.FieldType {
	case domain.IntType:
		return ranged.New[domain.IntValue](
			cmp.Compare[domain.IntValue],
		), nil
	case domain.FloatType:
		return ranged.New[domain.FloatValue](
			cmp.Compare[domain.FloatValue],
		), nil
	case domain.TimeType:
		return ranged.New(
			func(k1, k2 domain.TimeValue) int {
				key1 := time.Time(k1).Truncate(time.Millisecond)
				key2 := time.Time(k2).Truncate(time.Millisecond)

				switch {
				case key1.Before(key2):
					return -1
				case key1.After(key2):
					return 1
				default:
					return 0
				}
			}), nil
	default:
		return nil, fmt.Errorf(
			"ranged index doesn't support %v type",
			field.FieldType,
		)
	}
}

func newTextIndex(field domain.Field) (common.Index, error) {
	switch field.FieldType {
	case domain.StringType:
		return text.New(), nil
	default:
		return nil, fmt.Errorf(
			"text index doesn't support %v type",
			field.FieldType)
	}
}
