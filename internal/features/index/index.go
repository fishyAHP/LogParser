package index

import (
	"time"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Index[K comparable] interface {
	Add(key K, value domain.RecordData)
	Contains(key K) ([]domain.RecordData, bool)
	Remove(key K) bool
	Len() int
	Clear()
}

type TimeIndex interface {
	Add(time.Time, domain.RecordData) error
	Min()
}
