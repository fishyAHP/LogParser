package index

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type FieldIndex interface {
	Add(value domain.Value, record domain.RecordData) error
}

type ExactIndex interface {
	Exact(domain.Value) (*set.Set[domain.RecordData], error)
}

type RangeIndex interface {
	Range(from, to domain.Value) (*set.Set[domain.RecordData], error)
	Min() *set.Set[domain.RecordData]
	Max() *set.Set[domain.RecordData]
}
