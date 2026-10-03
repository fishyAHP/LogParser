package index

import (
	"errors"
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

var (
	ErrIndexNotFound       = errors.New("index wasn't found")
	ErrUnsupportedOperator = errors.New("unsupported operator")
	ErrUnknownIndexType    = errors.New("unknown index type in field")
)

type Service struct {
	Scheme  *domain.Scheme
	indexes map[string]common.Index
}

func New(scheme *domain.Scheme) (*Service, error) {
	service := &Service{
		Scheme:  scheme,
		indexes: make(map[string]common.Index, len(scheme.Parameters)),
	}

	for _, field := range scheme.Parameters {
		idx, err := newFieldIndex(field)
		if err != nil {
			return nil, fmt.Errorf("new field index: %w", err)
		}

		if idx != nil {
			service.indexes[field.Name] = idx
		}
	}

	return service, nil
}

func newFieldIndex(field domain.Field) (common.Index, error) {
	switch field.IndexType {
	case domain.NoIndex:
		return nil, nil
	case domain.HashIndex:
		return newHashIndex(field.FieldType)
	case domain.TextIndex:
		return newTextIndex(field.FieldType)
	case domain.RangeIndex:
		return newRangeIndex(field.FieldType)
	default:
		return nil, ErrUnknownIndexType
	}
}

func (s *Service) Index(
	record domain.RecordData,
	entry domain.LogEntry,
) error {
	if len(entry.Values) != len(s.Scheme.Parameters) {
		return fmt.Errorf(
			"entry values count doesn't match Scheme: got %d, want %d",
			len(entry.Values),
			len(s.Scheme.Parameters),
		)
	}

	for i, value := range entry.Values {
		field := s.Scheme.Parameters[i]

		if idx, ok := s.indexes[field.Name]; ok {
			if err := idx.Add(value, record); err != nil {
				return fmt.Errorf("index field %s: %w",
					field.Name,
					err,
				)
			}
		}
	}
	return nil
}

func (s *Service) Exact(
	fieldName string,
	value domain.Value,
) (*set.Set[domain.RecordData], error) {
	idx, ok := s.indexes[fieldName]
	if !ok {
		return nil, ErrIndexNotFound
	}

	exact, ok := idx.(common.ExactIndex)
	if !ok {
		return nil, fmt.Errorf(
			"%w: want ExactIndex, got %T",
			ErrUnsupportedOperator,
			idx,
		)
	}

	res, err := exact.Exact(value)
	if err != nil {
		return nil, fmt.Errorf("index exact: %w", err)
	}
	return res, nil
}

func (s *Service) Range(
	fieldName string,
	from, to *common.Bound,
) (*set.Set[domain.RecordData], error) {
	idx, ok := s.indexes[fieldName]
	if !ok {
		return nil, ErrIndexNotFound
	}

	rangeIdx, ok := idx.(common.RangeIndex)
	if !ok {
		return nil, fmt.Errorf(
			"%w: want RangeIndex, got %T",
			ErrUnsupportedOperator,
			idx,
		)
	}
	return rangeIdx.Range(from, to)
}

func (s *Service) Clear() {
	for _, idx := range s.indexes {
		idx.Clear()
	}
}
