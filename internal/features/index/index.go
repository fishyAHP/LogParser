package index

import (
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Index[K comparable] interface {
	Add(K, domain.RecordData)
	Contains(K) ([]domain.RecordData, bool)
	Remove(K) bool
	Len() int
	Clear()
}

type TimeIndex interface {
	Add(time.Time, domain.RecordData) error
	Find(time.Time)
	Range(time.Time, time.Time)
	Remove(time.Time)

	Min() []domain.RecordData
	Max() []domain.RecordData

	Len()
	Clear()
}
