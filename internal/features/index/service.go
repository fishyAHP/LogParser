package index

import (
	"errors"
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Service struct {
	scheme domain.Scheme

	indexes map[string]FieldIndex
}

func New(scheme domain.Scheme) (*Service, error) {
	service := &Service{
		scheme:  scheme,
		indexes: make(map[string]FieldIndex, len(scheme.Parameters)),
	}

	for _, field := range scheme.Parameters {
		idx, err := newFieldIndex(field)
		if err != nil {
			return nil, fmt.Errorf("new field index: %w", err)
		}

		service.indexes[field.Name] = idx
	}

	return service, nil
}

var ErrUnknownIndexType = errors.New("unknown index type in field")

func newFieldIndex(field domain.Field) (FieldIndex, error) {
	switch field.IndexType {
	case domain.NoIndex:
		return nil, nil
	case domain.HashIndex:
		return newHashIndex(field)
	case domain.TextIndex:
		return newTextIndex(field)
	case domain.RangeIndex:
		return newRangeIndex(field)
	default:
		return nil, ErrUnknownIndexType
	}
}

func (s *Service) Index(record domain.RecordData, entry domain.LogEntry) error {
	if len(entry.Values) != len(s.scheme.Parameters) {
		return fmt.Errorf(
			"entry values count doesn't match scheme: got %d, want %d",
			len(entry.Values),
			len(s.scheme.Parameters),
		)
	}

	for i, value := range entry.Values {
		field := s.scheme.Parameters[i]

		if idx, ok := s.indexes[field.Name]; ok {
			err := idx.Add(value, record)
			if err != nil {
				return fmt.Errorf("index field %s: %w",
					field.Name,
					err,
				)
			}
		}
	}
	return nil
}

var (
	ErrIndexNotFound       = errors.New("index wasn't found")
	ErrUnsupportedOperator = errors.New("unsupported operator")
)

func (s *Service) Exact(
	fieldName string,
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	idx, ok := s.indexes[fieldName]
	if !ok {
		return nil, ErrIndexNotFound
	}

	exactIdx, ok := idx.(ExactIndex)
	if !ok {
		return nil, ErrUnsupportedOperator
	}
	return exactIdx.Exact(value)
}

func (s *Service) Range(
	fieldName string,
	from, to domain.Value,
) (*set.Set[domain.RecordData], error) {
	idx, ok := s.indexes[fieldName]
	if !ok {
		return nil, ErrIndexNotFound
	}

	rangeIdx, ok := idx.(RangeIndex)
	if !ok {
		return nil, ErrUnsupportedOperator
	}

	return rangeIdx.Range(from, to)
}
