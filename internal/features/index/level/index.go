package level

import (
	"sync"

	"fishyAHP/LogParser.git/internal/core/domain"
)

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO           = "INFO"
	WARN           = "WARN"
	ERROR          = "ERROR"
	FATAL          = "FATAL"
)

type Index struct {
	index map[LogLevel][]domain.RecordData
	mtx   sync.RWMutex
}

func New() *Index {
	return &Index{
		index: make(map[LogLevel][]domain.RecordData, 5),
		mtx:   sync.RWMutex{},
	}
}

func (i *Index) Len() int {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return len(i.index)
}

func (i *Index) Add(comp LogLevel, data domain.RecordData) {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	i.index[comp] = append(i.index[comp], data)
}

func (i *Index) GetAll() map[LogLevel][]domain.RecordData {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return i.index
}

func (i *Index) Get(comp LogLevel) []domain.RecordData {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	return i.index[comp]
}
