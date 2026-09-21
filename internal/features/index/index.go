package index

import (
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Index[K comparable] interface {
	Add(K, domain.RecordData)
	Get(K) ([]domain.RecordData, bool)
	Remove(K) bool
	Len() int
	Clear()
}

type TimeIndex interface {
	Index[time.Time]

	Range(time.Time, time.Time) ([]domain.RecordData, bool)
	Min() []domain.RecordData
	Max() []domain.RecordData
}
