package index

import (
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
	"fishyAHP/LogParser.git/internal/features/index/set"
)

type Index[K comparable] interface {
	Add(K, domain.RecordData)
	Get(K) (*set.Set, bool)
	Remove(K) bool
	Delete(K, domain.RecordData) bool
	Len() int
	Clear()
}

type TimeIndex interface {
	Index[time.Time]

	Range(time.Time, time.Time) ([]domain.RecordData, bool)
	Min() *set.Set
	Max() *set.Set
}
