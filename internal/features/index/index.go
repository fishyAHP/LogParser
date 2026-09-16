package index

import (
	"slices"
	"sync"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type Index[K comparable] struct {
	index map[K][]domain.RecordData
	mtx   sync.RWMutex
}

func New[K comparable]() *Index[K] {
	return &Index[K]{
		index: make(map[K][]domain.RecordData),
		mtx:   sync.RWMutex{},
	}
}

func (i *Index[K]) Len() int {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return len(i.index)
}

func (i *Index[K]) Add(comp K, data domain.RecordData) {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	i.index[comp] = append(i.index[comp], data)
}

func (i *Index[K]) Get(comp K) []domain.RecordData {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return slices.Clone(i.index[comp])
}
