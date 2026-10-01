package index

import (
	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type fieldIndex interface {
	Add(value domain.Value, record domain.RecordData) error
}

type Index[K comparable] interface {
	Add(K, domain.RecordData) bool
	Get(K) (*set.Set[domain.RecordData], bool)
	Remove(K) bool
	Delete(K, domain.RecordData) bool
	Len() int
	Clear()
}

type RangeIndex[K comparable] interface {
	Index[K]

	Range(from, to K) (*set.Set[domain.RecordData], bool)
	Min() *set.Set[domain.RecordData]
	Max() *set.Set[domain.RecordData]
}

type TextIndex interface {
	Add(string, domain.RecordData) bool
	Get(string) *set.Set[domain.RecordData]
	RemoveTokens(string) bool
	RemoveRecord(string, domain.RecordData) bool
	Len() int
	Clear()
}
