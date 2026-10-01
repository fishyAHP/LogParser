package index

import (
	"errors"
	"fmt"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/hash"
	"fishyAHP/LogParser.git/internal/features/index/set"
	"fishyAHP/LogParser.git/internal/features/index/text"
	"fishyAHP/LogParser.git/internal/features/index/timestamp"
)

type stringHashIndex struct {
	index *hash.Index[string]
}

type intHashIndex struct {
	index *hash.Index[int64]
}

type floatHashIndex struct {
	index *hash.Index[float64]
}

type boolHashIndex struct {
	index *hash.Index[bool]
}

type timeRangeIndex struct {
	index *timestamp.Index
}

type textIndex struct {
	index *text.Index
}

func newHashIndex(field domain.Field) (FieldIndex, error) {
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

func newRangeIndex(field domain.Field) (FieldIndex, error) {
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

func newTextIndex(field domain.Field) (FieldIndex, error) {
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

var (
	_ FieldIndex = (*stringHashIndex)(nil)
	_ FieldIndex = (*intHashIndex)(nil)
	_ FieldIndex = (*floatHashIndex)(nil)
	_ FieldIndex = (*boolHashIndex)(nil)
	_ FieldIndex = (*timeRangeIndex)(nil)
	_ FieldIndex = (*textIndex)(nil)
	_ ExactIndex = (*stringHashIndex)(nil)
	_ ExactIndex = (*intHashIndex)(nil)
	_ ExactIndex = (*floatHashIndex)(nil)
	_ ExactIndex = (*boolHashIndex)(nil)
	_ ExactIndex = (*timeRangeIndex)(nil)
	_ ExactIndex = (*textIndex)(nil)
	_ RangeIndex = (*timeRangeIndex)(nil)
)

func (sh *stringHashIndex) Add(
	value domain.Value,
	record domain.RecordData,
) error {
	strVal, ok := value.(domain.StringValue)
	if !ok {
		return errors.New("invalid type for string hash index")
	}

	sh.index.Add(string(strVal), record)
	return nil
}

func (ih *intHashIndex) Add(value domain.Value, record domain.RecordData) error {
	intVal, ok := value.(domain.IntValue)
	if !ok {
		return errors.New("invalid type for int hash index")
	}

	ih.index.Add(int64(intVal), record)
	return nil
}

func (fh *floatHashIndex) Add(value domain.Value, record domain.RecordData) error {
	floatVal, ok := value.(domain.FloatValue)
	if !ok {
		return errors.New("invalid type for float hash index")
	}

	fh.index.Add(float64(floatVal), record)
	return nil
}

func (bh *boolHashIndex) Add(value domain.Value, record domain.RecordData) error {
	boolVal, ok := value.(domain.BoolValue)
	if !ok {
		return errors.New("invalid type for bool hash index")
	}

	bh.index.Add(bool(boolVal), record)
	return nil
}

func (tr *timeRangeIndex) Add(value domain.Value, record domain.RecordData) error {
	timeVal, ok := value.(domain.TimeValue)
	if !ok {
		return errors.New("invalid type for bool hash index")
	}

	tr.index.Add(time.Time(timeVal), record)
	return nil
}

func (ti *textIndex) Add(value domain.Value, record domain.RecordData) error {
	textVal, ok := value.(domain.StringValue)
	if !ok {
		return errors.New("invalid type for text index")
	}

	ti.index.Add(string(textVal), record)
	return nil
}

func (sh *stringHashIndex) Exact(
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	strVal, ok := value.(domain.StringValue)
	if !ok {
		return nil, errors.New("invalid type for string hash index")
	}

	res, _ := sh.index.Get(string(strVal))
	return res, nil
}

func (fh *floatHashIndex) Exact(
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	floatVal, ok := value.(domain.FloatValue)
	if !ok {
		return nil, errors.New("invalid type for float hash index")
	}

	res, _ := fh.index.Get(float64(floatVal))
	return res, nil
}

func (bh *boolHashIndex) Exact(
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	boolVal, ok := value.(domain.BoolValue)
	if !ok {
		return nil, errors.New("invalid type for bool hash index")
	}

	res, _ := bh.index.Get(bool(boolVal))
	return res, nil
}

func (ih *intHashIndex) Exact(
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	intVal, ok := value.(domain.IntValue)
	if !ok {
		return nil, errors.New("invalid type for int hash index")
	}

	res, _ := ih.index.Get(int64(intVal))
	return res, nil
}

func (tr *timeRangeIndex) Exact(
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	timeVal, ok := value.(domain.TimeValue)
	if !ok {
		return nil, errors.New("invalid type for string hash index")
	}

	res, _ := tr.index.Get(time.Time(timeVal))
	return res, nil
}

func (ti *textIndex) Exact(
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	strVal, ok := value.(domain.StringValue)
	if !ok {
		return nil, errors.New("invalid type for string hash index")
	}

	return ti.index.Get(string(strVal)), nil
}

func (tr *timeRangeIndex) Range(
	from, to domain.Value,
) (*set.Set[domain.RecordData], error) {
	fromTime, ok := from.(domain.TimeValue)
	if !ok {
		return nil, errors.New("invalid type for time range index")
	}
	toTime, ok := to.(domain.TimeValue)
	if !ok {
		return nil, errors.New("invalid type for time range index")
	}

	res, _ := tr.index.Range(
		time.Time(fromTime),
		time.Time(toTime),
	)
	return res, nil
}

func (tr *timeRangeIndex) Min() *set.Set[domain.RecordData] {
	return tr.index.Min()
}

func (tr *timeRangeIndex) Max() *set.Set[domain.RecordData] {
	return tr.index.Max()
}
