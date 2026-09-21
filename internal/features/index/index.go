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
	Find(time.Time) ([]domain.RecordData, bool)
	Range(time.Time, time.Time) ([]domain.RecordData, bool)
	Remove(time.Time) bool

	Min() []domain.RecordData
	Max() []domain.RecordData

	Len() int
	Clear()
}
