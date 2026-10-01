package index

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type FieldIndex struct {
	index Index
}

type Index interface {
	Len() int
	Clear()
	Exact(domain.Value) (*set.Set[domain.RecordData], error)
	Add(value domain.Value, record domain.RecordData) error
}

type RangeIndex interface {
	Index
	Range(from, to domain.Value) (*set.Set[domain.RecordData], error)
	Min() *set.Set[domain.RecordData]
	Max() *set.Set[domain.RecordData]
}
