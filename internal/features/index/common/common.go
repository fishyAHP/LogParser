package common

import (
	"errors"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/structs"
)

var (
	ErrInvalidType    = errors.New("invalid value type")
	ErrRecordNotFound = errors.New("record not found")
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
	Exact(domain.Value) (*structs.PostingList, error)
}

type RangeIndex interface {
	ExactIndex
	Range(from, to *Bound) (*structs.PostingList, error)
}

type TextIndex interface {
	Index
	Search(domain.Value) (*structs.PostingList, error)
}

type Grouper interface {
	Group(*structs.PostingList) (Groups, error)
}

type Aggregator interface {
	Min(*structs.PostingList) (domain.Value, error)
	Max(*structs.PostingList) (domain.Value, error)
	Avg(*structs.PostingList) (domain.Value, error)
	Sum(*structs.PostingList) (domain.Value, error)
}

type Group struct {
	Value   domain.Value
	Records *structs.PostingList
}

type Groups []Group

type Bound struct {
	Value     domain.Value
	Inclusive bool
}
