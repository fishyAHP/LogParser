package interfaces

import (
	"iter"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Grouper interface {
	Group(iter.Seq[domain.RecordID]) (Groups, error)
}

type Aggregator interface {
	Add(domain.Value) error
	Result() (domain.Value, error)
}

type Group struct {
	Value   domain.Value
	Records iter.Seq[domain.RecordID]
}

type Groups []Group
