package index

import (
	"errors"
	"fmt"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Service struct {
	scheme domain.Scheme

	indexes map[string]fieldIndex
}

func New(scheme domain.Scheme) (*Service, error) {
	service := &Service{
		scheme:  scheme,
		indexes: make(map[string]fieldIndex, len(scheme.Parameters)),
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

func newFieldIndex(field domain.Field) (fieldIndex, error) {
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
