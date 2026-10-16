package common

import (
	"errors"
	"iter"

	"fishyAHP/LogParser.git/internal/core/domain"
)

var (
	ErrInvalidType     = errors.New("invalid value type")
	ErrRecordNotFound  = errors.New("record not found")
	ErrUnsupportedType = errors.New("unsupported type")
	ErrKeyExist        = errors.New("key already exists")
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
	Exact(domain.Value) (iter.Seq[domain.RecordID], error)
}

type RangeIndex interface {
	ExactIndex
	Range(from, to *Bound) (iter.Seq[domain.RecordID], error)
}

type TextIndex interface {
	Index
	Search(domain.Value) (iter.Seq[domain.RecordID], error)
}

type Grouper interface {
	Group(seq iter.Seq[domain.RecordID]) (Groups, error)
}

type Aggregator interface {
	Min(seq iter.Seq[domain.RecordID]) (domain.Value, error)
	Max(seq iter.Seq[domain.RecordID]) (domain.Value, error)
	Avg(seq iter.Seq[domain.RecordID]) (domain.Value, error)
	Sum(seq iter.Seq[domain.RecordID]) (domain.Value, error)
}

type Group struct {
	Value   domain.Value
	Records iter.Seq[domain.RecordID]
}

type Groups []Group

type Bound struct {
	Value     domain.Value
	Inclusive bool
}
