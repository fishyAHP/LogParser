package common

import (
	"errors"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

var (
	ErrInvalidType    = errors.New("invalid value type")
	ErrNotFoundRecord = errors.New("not found record")
)

type Key interface {
	domain.Value
	comparable
}

type Index interface {
	Len() int
	Clear()
	Add(domain.Value, domain.RecordID) error
	Remove(domain.Value, domain.RecordID) error
}

type ExactIndex interface {
	Index
	Exact(domain.Value) (*structs.Set[domain.RecordID], error)
}

type RangeIndex interface {
	ExactIndex
	Range(from, to *Bound) (*structs.Set[domain.RecordID], error)
}

type TextIndex interface {
	Index
	Search(domain.Value) (*structs.Set[domain.RecordID], error)
}

type Bound struct {
	Value     domain.Value
	Inclusive bool
}
