package component

import (
	"sync"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type LogComponent = string

type Index struct {
	index map[LogComponent][]domain.RecordData
	mtx   sync.RWMutex
}

func New() *Index {
	return &Index{
		index: make(map[LogComponent][]domain.RecordData),
		mtx:   sync.RWMutex{},
	}
}

func (i *Index) Len() int {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return len(i.index)
}

func (i *Index) Add(comp LogComponent, data domain.RecordData) {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	i.index[comp] = append(i.index[comp], data)
}

func (i *Index) GetAll() map[LogComponent][]domain.RecordData {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return i.index
}

func (i *Index) Get(comp LogComponent) []domain.RecordData {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return i.index[comp]
}
