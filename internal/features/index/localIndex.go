package index

import (
	"errors"
	"fmt"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/hash"
	"fishyAHP/LogParser.git/internal/features/index/text"
	"fishyAHP/LogParser.git/internal/features/index/timestamp"
)

type stringHashIndex struct {
	index Index[string]
}

type intHashIndex struct {
	index Index[int64]
}

type floatHashIndex struct {
	index Index[float64]
}

type boolHashIndex struct {
	index Index[bool]
}

type timeRangeIndex struct {
	index RangeIndex[time.Time]
}

type textIndex struct {
	index TextIndex
}

var (
	_ fieldIndex = (*stringHashIndex)(nil)
	_ fieldIndex = (*intHashIndex)(nil)
	_ fieldIndex = (*floatHashIndex)(nil)
	_ fieldIndex = (*boolHashIndex)(nil)
	_ fieldIndex = (*timeRangeIndex)(nil)
	_ fieldIndex = (*textIndex)(nil)
)

func (s *stringHashIndex) Add(
	value domain.Value,
	record domain.RecordData,
) error {
	strVal, ok := value.(domain.StringValue)
	if !ok {
		return errors.New("invalid type for string hash index")
	}

	s.index.Add(string(strVal), record)
	return nil
}

func (i *intHashIndex) Add(value domain.Value, record domain.RecordData) error {
	intVal, ok := value.(domain.IntValue)
	if !ok {
		return errors.New("invalid type for int hash index")
	}

	i.index.Add(int64(intVal), record)
	return nil
}

func (f *floatHashIndex) Add(value domain.Value, record domain.RecordData) error {
	floatVal, ok := value.(domain.FloatValue)
	if !ok {
		return errors.New("invalid type for float hash index")
	}

	f.index.Add(float64(floatVal), record)
	return nil
}

func (b *boolHashIndex) Add(value domain.Value, record domain.RecordData) error {
	boolVal, ok := value.(domain.BoolValue)
	if !ok {
		return errors.New("invalid type for bool hash index")
	}

	b.index.Add(bool(boolVal), record)
	return nil
}

func (t *timeRangeIndex) Add(value domain.Value, record domain.RecordData) error {
	timeVal, ok := value.(domain.TimeValue)
	if !ok {
		return errors.New("invalid type for bool hash index")
	}

	t.index.Add(time.Time(timeVal), record)
	return nil
}

func (t *textIndex) Add(value domain.Value, record domain.RecordData) error {
	textVal, ok := value.(domain.StringValue)
	if !ok {
		return errors.New("invalid type for text index")
	}

	t.index.Add(string(textVal), record)
	return nil
}

func newHashIndex(field domain.Field) (fieldIndex, error) {
	switch field.FieldType {
	case domain.StringType:
		return &stringHashIndex{
			index: hash.New[string](),
		}, nil
	case domain.BoolType:
		return &boolHashIndex{
			index: hash.New[bool](),
		}, nil
	case domain.IntType:
		return &intHashIndex{
			index: hash.New[int64](),
		}, nil
	case domain.FloatType:
		return &floatHashIndex{
			index: hash.New[float64](),
		}, nil
	default:
		return nil, fmt.Errorf(
			"hash index doesnt support %v type",
			field.FieldType,
		)
	}
}

func newRangeIndex(field domain.Field) (fieldIndex, error) {
	switch field.FieldType {
	case domain.IntType:
		return &intHashIndex{
			index: hash.New[int64](),
		}, nil
	case domain.FloatType:
		return &floatHashIndex{
			index: hash.New[float64](),
		}, nil
	case domain.TimeType:
		return &timeRangeIndex{
			index: timestamp.New(time.Millisecond),
		}, nil
	default:
		return nil, fmt.Errorf(
			"range index doesn't support %v type",
			field.FieldType,
		)
	}
}

func newTextIndex(field domain.Field) (fieldIndex, error) {
	switch field.FieldType {
	case domain.StringType:
		return &textIndex{
			index: text.New(),
		}, nil
	default:
		return nil, fmt.Errorf(
			"text index doesn't support %v type",
			field.FieldType)
	}
}
