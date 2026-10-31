package index

import (
	"encoding"
	"fmt"
	"iter"
	"slices"
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/core/errors"
	"fishyAHP/LogParser.git/internal/features/aggregation/interfaces"
	"fishyAHP/LogParser.git/internal/features/index/common"
)

type Manager struct {
	scheme  *domain.Scheme
	indexes map[string]common.Index
}

func New(scheme *domain.Scheme) (*Manager, error) {
	service := &Manager{
		scheme:  scheme,
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
		return nil, coreerrors.ErrUnknownIndexType
	}
}

func (m *Manager) Scheme() *domain.Scheme {
	return m.scheme
}

func (m *Manager) Index(
	record domain.RecordID,
	entry domain.LogEntry,
) error {
	if len(entry.Values) != len(m.scheme.Parameters) {
		return fmt.Errorf(
			"entry values count doesn't match Scheme: got %d, want %d",
			len(entry.Values),
			len(m.scheme.Parameters),
		)
	}

	for i, value := range entry.Values {
		field := m.scheme.Parameters[i]

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
) (iter.Seq[domain.RecordID], error) {
	idx, ok := m.indexes[fieldName]
	if !ok {
		return nil, coreerrors.ErrIndexNotFound
	}

	exact, ok := idx.(common.ExactIndex)
	if !ok {
		return nil, fmt.Errorf(
			"%w: want ExactIndex, got %T",
			coreerrors.ErrUnsupportedOperator,
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
	from, to *domain.Bound,
) (iter.Seq[domain.RecordID], error) {
	idx, ok := m.indexes[fieldName]
	if !ok {
		return nil, coreerrors.ErrIndexNotFound
	}

	rangeIdx, ok := idx.(common.RangeIndex)
	if !ok {
		return nil, fmt.Errorf(
			"%w: want RangeIndex, got %T",
			coreerrors.ErrUnsupportedOperator,
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
) (interfaces.Grouper, error) {
	idx, ok := m.indexes[name]
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q",
			coreerrors.ErrIndexNotFound,
			name,
		)
	}

	grouper, ok := idx.(interfaces.Grouper)
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q",
			coreerrors.ErrNotGrouper,
			name,
		)
	}

	return grouper, nil
}

func (m *Manager) Aggregator(
	name string,
) (interfaces.Grouper, error) {
	idx, ok := m.indexes[name]
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q",
			coreerrors.ErrIndexNotFound,
			name,
		)
	}

	aggregator, ok := idx.(interfaces.Grouper)
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q",
			coreerrors.ErrNotGrouper,
			name,
		)
	}

	return aggregator, nil
}

type Snapshot struct {
	Name string
	Data []byte
}

func (m *Manager) MakeSnapshots() iter.Seq2[Snapshot, error] {
	return func(yield func(Snapshot, error) bool) {
		for name, index := range m.indexes {
			marshaler, ok := index.(encoding.BinaryMarshaler)
			if !ok {
				yield(Snapshot{}, fmt.Errorf(
					"index %q doesnt implement %T",
					name,
					encoding.BinaryMarshaler(nil),
				))
				return
			}
			data, err := marshaler.MarshalBinary()
			if err != nil {
				yield(
					Snapshot{},
					fmt.Errorf(
						"marshal index %q: %w",
						name,
						err,
					),
				)

				return
			}

			if !yield(Snapshot{
				Name: name,
				Data: data,
			}, nil) {
				return
			}
		}
	}
}

func (m *Manager) FromSnapshots(
	snapshots iter.Seq2[Snapshot, error],
) error {
	tmp := make(map[string]common.Index)

	for snapshot, err := range snapshots {
		if err != nil {
			return fmt.Errorf(
				"failed to get snapshot: %w",
				err,
			)
		}

		if _, ok := tmp[snapshot.Name]; ok {
			return fmt.Errorf(
				"index %q already exist",
				snapshot.Name,
			)
		}

		idx := slices.IndexFunc(
			m.scheme.Parameters,
			func(field domain.Field) bool {
				return field.Name == snapshot.Name
			},
		)

		if idx == -1 {
			return fmt.Errorf(
				"index %q dont exist in schema",
				snapshot.Name,
			)
		}

		field := m.scheme.Parameters[idx]
		if field.IndexType == domain.NoIndex {
			return fmt.Errorf(
				"field %q is not indexed",
				snapshot.Name,
			)
		}

		index, err := newFieldIndex(field)
		if err != nil {
			return fmt.Errorf(
				"create index %q: %w",
				snapshot.Name,
				err,
			)
		}

		unmarshaler, ok := index.(encoding.BinaryUnmarshaler)
		if !ok {
			return fmt.Errorf(
				"index %q doesnt implement %T",
				snapshot.Name,
				encoding.BinaryUnmarshaler(nil),
			)
		}

		start := time.Now()

		if err = unmarshaler.UnmarshalBinary(
			snapshot.Data,
		); err != nil {
			return fmt.Errorf(
				"rebuilding index %q from snapshot: %w",
				snapshot.Name,
				err,
			)
		}

		fmt.Printf(
			"Index %s: size=%d MB, load=%s\n",
			snapshot.Name,
			len(snapshot.Data)/(1024*1024),
			time.Since(start),
		)

		tmp[snapshot.Name] = index
	}

	for _, field := range m.scheme.Parameters {
		if field.IndexType == domain.NoIndex {
			continue
		}

		if _, ok := tmp[field.Name]; !ok {
			return fmt.Errorf(
				"missing snapshot for index %q",
				field.Name,
			)
		}
	}

	m.indexes = tmp

	return nil
}
