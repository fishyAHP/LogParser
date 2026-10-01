package common

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Index interface {
	Len() int
	Clear()
	Add(domain.Value, domain.RecordData) error
	Remove(domain.Value, domain.RecordData) error
}

type ExactIndex interface {
	Index
	Exact(domain.Value) (*set.Set[domain.RecordData], error)
}

type Bound struct {
	Value     domain.Value
	Exclusive bool
}
type RangeIndex interface {
	ExactIndex
	Range(from, to *Bound) (*set.Set[domain.RecordData], error)
}

type TextIndex interface {
	Index
	Search(domain.Value) (*set.Set[domain.RecordData], error)
}
