package index

import (
	"errors"
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/common"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

var (
	ErrIndexNotFound       = errors.New("index wasn't found")
	ErrUnsupportedOperator = errors.New("unsupported operator")
	ErrUnknownIndexType    = errors.New("unknown index type in field")
	ErrNotGrouper          = errors.New("index doesn't support grouping")
	ErrNotAggregator       = errors.New("index doesn't support aggregating")
)

type Manager struct {
	Scheme  *domain.Scheme
	indexes map[string]common.Index
}

func New(scheme *domain.Scheme) (*Manager, error) {
	service := &Manager{
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

func (m *Manager) Index(
	record domain.RecordID,
	entry domain.LogEntry,
) error {
	if len(entry.Values) != len(m.Scheme.Parameters) {
		return fmt.Errorf(
			"entry values count doesn't match Scheme: got %d, want %d",
			len(entry.Values),
			len(m.Scheme.Parameters),
		)
	}

	for i, value := range entry.Values {
		field := m.Scheme.Parameters[i]

		if idx, ok := m.indexes[field.Name]; ok {
			if err := idx.Add(value, record); err != nil {
				return fmt.Errorf("index field %q: %w",
					field.Name,
					err,
				)
			}
		}
	}
	return nil
}

func (m *Manager) Exact(
	fieldName string,
	value domain.Value,
) (*structs.PostingList, error) {
	idx, ok := m.indexes[fieldName]
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

func (m *Manager) Range(
	fieldName string,
	from, to *common.Bound,
) (*structs.PostingList, error) {
	idx, ok := m.indexes[fieldName]
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

func (m *Manager) Clear() {
	for _, idx := range m.indexes {
		idx.Clear()
	}
}

func (m *Manager) Grouper(
	name string,
) (common.Grouper, error) {
	idx, ok := m.indexes[name]
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q",
			ErrIndexNotFound,
			name,
		)
	}

	grouper, ok := idx.(common.Grouper)
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q",
			ErrNotGrouper,
			name,
		)
	}

	return grouper, nil
}

func (m *Manager) Aggregator(
	name string,
) (common.Aggregator, error) {
	idx, ok := m.indexes[name]
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q",
			ErrIndexNotFound,
			name,
		)
	}

	aggregator, ok := idx.(common.Aggregator)
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q",
			ErrNotGrouper,
			name,
		)
	}

	return aggregator, nil
}
